package services

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
)

// 上下文 token 预算与摘要压缩相关常量
const (
	// DefaultContextTokenBudget 默认上下文 token 预算（128K）
	DefaultContextTokenBudget = 131072
	// MinContextTokenBudget 预算下限
	MinContextTokenBudget = 32768
	// MaxContextTokenBudget 预算上限
	MaxContextTokenBudget = 524288
	// DefaultSummaryTriggerRatio 默认的摘要压缩触发比例（tail 达预算 80%）
	DefaultSummaryTriggerRatio = 0.8
	// MinSummaryTriggerRatio 触发比例下限（设置页可配，低于该值视为非法并重置为默认值）
	MinSummaryTriggerRatio = 0.1
	// MaxSummaryTriggerRatio 触发比例上限
	MaxSummaryTriggerRatio = 0.9
	// CompactKeepRatio 压缩后保留最近该比例预算的 tail
	CompactKeepRatio = 0.5
	// SummaryRegionTokenCap 单次摘要生成的输入区间 token 上限（防止摘要调用自身超模型上下文）
	SummaryRegionTokenCap = 40000
	// MaxSummaryRunes 摘要文本长度上限（rune）
	MaxSummaryRunes = 2000
)

// EstimateTokens 估算文本的 token 数（中文按 1.5 字/token，其他按 4 字符/token，
// 与前端 estimateTokens 算法一致）
func EstimateTokens(text string) int {
	chineseCount := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}
	runes := []rune(text)
	otherCount := len(runes) - chineseCount
	return int(math.Ceil(float64(chineseCount)/1.5 + float64(otherCount)/4))
}

// GetContextTokenBudget 获取 AI 上下文 token 预算。
// 从 setting 表读取 ai_context_token_budget，不存在或非法时返回默认值 131072（128K）。
func (a *AIService) GetContextTokenBudget() int {
	svc := NewSettingService(a.db)
	val := svc.Get("ai_context_token_budget")
	n, err := strconv.Atoi(val)
	if err != nil || n < MinContextTokenBudget {
		return DefaultContextTokenBudget
	}
	if n > MaxContextTokenBudget {
		return MaxContextTokenBudget
	}
	return n
}

// GetContextSummaryTriggerRatio 获取摘要压缩触发比例（设置键 ai_context_summary_trigger_ratio）。
// 该比例在 [MinSummaryTriggerRatio, MaxSummaryTriggerRatio] 即 [0.1, 0.9] 内合法，
// 否则返回默认值 0.8（DefaultSummaryTriggerRatio）。
func (a *AIService) GetContextSummaryTriggerRatio() float64 {
	svc := NewSettingService(a.db)
	val := svc.Get("ai_context_summary_trigger_ratio")
	n, err := strconv.ParseFloat(val, 64)
	if err != nil || n < MinSummaryTriggerRatio {
		return DefaultSummaryTriggerRatio
	}
	if n > MaxSummaryTriggerRatio {
		return MaxSummaryTriggerRatio
	}
	return n
}

// SelectTailByTokenBudget 从消息尾部按 token 预算选取 tail：
//   - 从后往前累计 EstimateTokens(Content)，加入下一条会超预算时停止
//   - 最后一条消息始终保留（单条超预算时 tail 允许超出预算）
//   - 轮次对齐：tail 首条必须是 user 消息（边界落在 assistant 时向后丢弃直至 user）
//
// 调用方应先剔除 system 消息，本函数只处理 user/assistant 消息。
// 返回 (tail 消息切片, tail 在传入切片中的起始下标)。
func SelectTailByTokenBudget(messages []Message, budget int) ([]Message, int) {
	n := len(messages)
	if n == 0 {
		return nil, 0
	}
	total := 0
	start := n
	for i := n - 1; i >= 0; i-- {
		t := EstimateTokens(messages[i].Content)
		// 最后一条始终保留，其余超预算即停止
		if i != n-1 && total+t > budget {
			break
		}
		total += t
		start = i
	}
	// 轮次对齐：tail 首条必须是 user
	for start < n-1 && messages[start].Role != "user" {
		start++
	}
	return messages[start:], start
}

// SelectKeepTailByTokenBudget 摘要压缩时从 tail 头部丢弃旧消息，
// 使保留区估算 token ≤ budget*keepRatio，且保留区首条对齐为 user 消息。
// 最后一条消息始终保留。返回保留区（tail 的后缀切片）。
func SelectKeepTailByTokenBudget(tail []Message, budget int, keepRatio float64) []Message {
	n := len(tail)
	if n == 0 {
		return tail
	}
	keepBudget := int(float64(budget) * keepRatio)
	total := 0
	start := n
	for i := n - 1; i >= 0; i-- {
		t := EstimateTokens(tail[i].Content)
		if i != n-1 && total+t > keepBudget {
			break
		}
		total += t
		start = i
	}
	// 轮次对齐：保留区首条必须是 user（至少保留最后一条）
	for start < n-1 && tail[start].Role != "user" {
		start++
	}
	return tail[start:]
}

// limitRegionByTokens 限制待摘要区间的输入体量：超过 capTokens 时只保留区间末尾
// 约 capTokens 的消息（更早内容已由旧摘要覆盖）。
func limitRegionByTokens(msgs []Message, capTokens int) []Message {
	n := len(msgs)
	if n == 0 {
		return msgs
	}
	total := 0
	start := n
	for i := n - 1; i >= 0; i-- {
		t := EstimateTokens(msgs[i].Content)
		if start < n && total+t > capTokens {
			break
		}
		total += t
		start = i
	}
	return msgs[start:]
}

// CompactSessionSummary 压缩会话摘要：
//   - 加载会话取旧摘要（SummaryContent）
//   - 输入为待摘要区间消息（调用方已按边界与 tail 选取切好），
//     区间 token 超 SummaryRegionTokenCap 时仅取末尾部分
//   - 调用 GenerateSessionSummary（旧摘要 + 区间消息）生成新摘要
//   - 成功后单次 Updates 持久化 summary_content + summary_up_to_msg_id
//
// 返回 (新摘要文本, 是否更新成功)；失败时返回 ("", false)，调用方沿用旧摘要。
func (a *AIService) CompactSessionSummary(ctx context.Context, sessionID uint, msgs []Message, newBoundaryMsgID uint) (string, bool) {
	if len(msgs) == 0 {
		return "", false
	}
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err != nil {
		a.logger.Warnw("会话摘要压缩失败：加载会话出错", fastlog.Error(err))
		return "", false
	}

	region := limitRegionByTokens(msgs, SummaryRegionTokenCap)

	// 使用传入的 ctx（含 AI 流取消信号），用户取消时摘要生成也随之取消。
	// 超时 90s：摘要输入可达 40K token，部分网关非流式响应较慢（实测 13s~30s+），
	// 30s 会频繁超时；等待期间前端状态条可见（generating），用户可随时手动停止
	genCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	newSummary := a.GenerateSessionSummary(genCtx, session.SummaryContent, region)
	if newSummary == "" {
		return "", false
	}

	if err := a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"summary_content":      newSummary,
		"summary_up_to_msg_id": newBoundaryMsgID,
	}).Error; err != nil {
		a.logger.Warnw("会话摘要持久化失败", fastlog.Error(err))
		return "", false
	}

	a.logger.Infow("会话摘要已压缩更新",
		fastlog.Uint("session_id", sessionID),
		fastlog.Uint("summary_up_to_msg_id", newBoundaryMsgID),
		fastlog.Int("region_msgs", len(region)))
	return newSummary, true
}

// GenerateSessionSummary 基于旧摘要 + 待摘要消息生成新的会话摘要。
// 调用方传入旧摘要（可能为空）和待摘要的 user/assistant 消息列表，
// 调用 AI 模型生成新的结构化要点摘要。
// 失败时返回空字符串，调用方回退沿用旧摘要。
func (a *AIService) GenerateSessionSummary(ctx context.Context, oldSummary string, newMessages []Message) string {
	prompt := buildSummaryPrompt(oldSummary, newMessages)

	summary, err := a.CallAI(ctx, []Message{{Role: "user", Content: prompt}})
	if err != nil {
		a.logger.Warnw("会话摘要生成失败，沿用旧摘要", fastlog.Error(err))
		return ""
	}

	summary = strings.TrimSpace(summary)
	runes := []rune(summary)
	if len(runes) > MaxSummaryRunes {
		summary = string(runes[:MaxSummaryRunes])
	}
	return summary
}

// buildSummaryPrompt 构建摘要生成提示词
func buildSummaryPrompt(oldSummary string, newMessages []Message) string {
	var b strings.Builder
	b.WriteString("你是一个对话摘要专家。请将以下对话内容压缩为结构化要点摘要。\n\n")
	b.WriteString("规则：\n")
	b.WriteString("- 提取核心信息：用户意图、关键决定、重要事实、偏好设定、行动项\n")
	b.WriteString("- 每条消息用 1~2 句话概括，不要大段复制原文\n")
	b.WriteString("- 数字、日期、人名、术语必须准确，不得编造\n")
	b.WriteString("- 保留用户明确表达的偏好和设置（如语言偏好、常用格式等）\n")
	b.WriteString("- 输出为结构化要点列表，用小节标题组织\n")
	b.WriteString("- 只输出摘要本身，不要任何解释、开场白或结尾语\n\n")

	if oldSummary != "" {
		b.WriteString("【现有摘要】\n")
		b.WriteString(oldSummary)
		b.WriteString("\n\n")
		b.WriteString("【新增对话】\n")
		// 不变量：更早的对话均已覆盖在现有摘要中，逐条列出的只是待合并区间
		b.WriteString("（更早的对话未逐条列出，其要点已包含在现有摘要中，请勿遗漏其中的关键信息。）\n\n")
	}

	for _, msg := range newMessages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		b.WriteString(role + "：")
		runes := []rune(msg.Content)
		if len(runes) > 500 {
			b.WriteString(string(runes[:500]))
			b.WriteString("…（过长已截断）")
		} else {
			b.WriteString(msg.Content)
		}
		b.WriteString("\n\n")
	}

	b.WriteString("请基于现有摘要和新增对话，生成更新后的完整摘要：")
	return b.String()
}
