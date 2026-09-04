package tools

// 本文件实现 manage_memory 全局记忆管理工具：模型在 ReAct 循环中调用它保存、更新、
// 删除、查询或列出长期记忆（跨会话持久记忆，用于保存用户偏好、重要事实等），
// 底层复用 services.MemoryService，不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分五个动作：
//   - create：新增记忆（summary 必填、限制 MaxMemorySummaryRunes；content 可选、
//     超 MaxMemoryContentRunes 时截断；同名 summary 已存在时返回提示而非错误，
//     避免触发 WrapWithError 的失败回填）；
//   - update：更新记忆（id 必填、正整数；summary/content 均可提供）；
//   - delete：删除记忆（ids 必填、正整数数组，可一次删除多条）；
//   - get：查询单条记忆详情（id 必填、正整数）；
//   - list：列出全部记忆（created_at 倒序），模型据此知道有哪些记忆及其 id。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/models"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// MaxMemorySummaryRunes 记忆 summary（简短描述，用于注入）的最大长度（按 rune 计）。
const MaxMemorySummaryRunes = 200

// MaxMemoryContentRunes 记忆 content（详情）的最大长度（按 rune 计），超长时截断。
const MaxMemoryContentRunes = 2000

// memoryTool 工具结构体：声明所需依赖。
type memoryTool struct {
	memSvc *services.MemoryService // 记忆服务
	ctx    *Context                // 日志依赖，仅用于 Debugw 记录
}

// 编译期断言：确保实现 tool.InvokableTool。
var _ tool.InvokableTool = (*memoryTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *memoryTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "create":
		return "新增记忆"
	case "update":
		return "更新记忆"
	case "delete":
		return "删除记忆"
	case "get":
		return "查询记忆"
	case "list":
		return "列出记忆"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（模型据此决定是否调用）。
func (m *memoryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_memory",
		Desc: "管理长期记忆（跨会话持久记忆，用于保存用户偏好、重要事实、约定等需要在后续对话中长期记住的信息）。当用户明确要求记住某信息、询问你记得/保存了什么，或需要删除/修改一条此前保存的记忆时调用；当任务只需本条会话的一次性信息、或信息来源可从本地笔记/待办等既有数据读取时不要调用（会污染长期记忆）。通过 action 参数区分动作：create=新增记忆（需提供 summary 简短描述，用于注入；可提供 content 详情，均可为空）；update=更新记忆（需提供 id 记忆编号，可提供新 summary 或 content，至少其一）；delete=删除记忆（需提供 ids 记忆编号列表，可一次删除多条）；get=查询单条记忆详情（需提供 id 记忆编号）；list=列出全部记忆（无额外参数，返回每条记忆的编号/描述/详情摘要）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=新增记忆；update=更新记忆；delete=删除记忆；get=查询单条记忆；list=列出全部记忆",
				Enum:     []string{"create", "update", "delete", "get", "list"},
				Required: true,
			},
			"summary": {
				Type:     schema.String,
				Desc:     "记忆的简短描述（用于注入，唯一，用于去重），action=create 时必填、action=update 时可提供新描述（留空则保留原值）",
				Required: false,
			},
			"content": {
				Type:     schema.String,
				Desc:     "记忆的详情内容，action=create/update 时可选（留空则保留原值）",
				Required: false,
			},
			"id": {
				Type:     schema.Number,
				Desc:     "记忆编号（正整数，action=update/get 时必填，来自 list 返回的编号）",
				Required: false,
			},
			"ids": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Number,
					Desc: "记忆编号（正整数）",
				},
				Desc:     "记忆编号列表（正整数数组，action=delete 时必填，来自 list 返回的编号；支持一次删除多条）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发。
func (m *memoryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action  string    `json:"action"`
		Summary string    `json:"summary"`
		Content string    `json:"content"`
		ID      float64   `json:"id"`
		IDs     []float64 `json:"ids"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_memory 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent manage_memory 调用",
			fastlog.String("action", args.Action),
			fastlog.String("summary", args.Summary))
	}

	switch args.Action {
	case "create":
		return m.createMemory(ctx, args.Summary, args.Content)
	case "update":
		return m.updateMemory(ctx, args.ID, args.Summary, args.Content)
	case "delete":
		return m.deleteMemories(ctx, args.IDs)
	case "get":
		return m.getMemory(ctx, args.ID)
	case "list":
		return m.listMemories(ctx)
	}
	return "", fmt.Errorf("manage_memory 未知 action: %s", args.Action)
}

// createMemory 新增记忆：summary 必填、超限报错；content 可选、超限截断；
// 同名 summary 已存在返回提示而非错误（不重复新增）。
func (m *memoryTool) createMemory(ctx context.Context, summary, content string) (string, error) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", errors.New("manage_memory 参数缺少 summary")
	}
	if n := len([]rune(summary)); n > MaxMemorySummaryRunes {
		return "", fmt.Errorf("manage_memory summary 过长（%d 字符，上限 %d），请精简后重试", n, MaxMemorySummaryRunes)
	}
	note := ""
	if n := len([]rune(content)); n > MaxMemoryContentRunes {
		content = TruncateRunes(content, MaxMemoryContentRunes)
		note = "（详情过长已截断）"
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	mem, err := m.memSvc.Create(summary, content)
	if err != nil {
		// 同名记忆已存在：返回正常提示而非错误，避免进 WrapWithError 失败路径
		if errors.Is(err, services.ErrMemoryExists) {
			return m.existsHint(summary)
		}
		return "", err
	}
	return fmt.Sprintf("已保存记忆：%s%s", mem.Summary, note), nil
}

// existsHint 构造同名记忆已存在的提示文本（从 List 中查找现有记录的 id 与内容）。
func (m *memoryTool) existsHint(summary string) (string, error) {
	all, err := m.memSvc.List()
	if err != nil {
		return "", err
	}
	for i := range all {
		if all[i].Summary == summary {
			return fmt.Sprintf("该记忆已存在，id=%d，已有描述=%s，已有内容=%s，可用 update 更新，未重复新增",
				all[i].ID, all[i].Summary, all[i].Content), nil
		}
	}
	return "该记忆已存在，未重复新增；可用 update 更新", nil
}

// updateMemory 更新记忆：id 必填、正整数；支持部分更新——summary/content 未提供的
// 字段（为空）保留原值，只更新传入的字段，避免把已有内容清空。
func (m *memoryTool) updateMemory(ctx context.Context, id float64, summary, content string) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_memory 更新记忆缺少有效的 id")
	}
	summary = strings.TrimSpace(summary)
	// 部分更新：任一字段未提供时先取原记录，保留对应旧值
	if summary == "" || content == "" {
		cur, err := m.memSvc.Get(uint(id))
		if err != nil {
			return "", fmt.Errorf("manage_memory 更新记忆目标不存在或查询失败: %w", err)
		}
		if summary == "" {
			summary = cur.Summary
		}
		if content == "" {
			content = cur.Content
		}
	}
	note := ""
	if n := len([]rune(summary)); n > MaxMemorySummaryRunes {
		return "", fmt.Errorf("manage_memory summary 过长（%d 字符，上限 %d），请精简后重试", n, MaxMemorySummaryRunes)
	}
	if n := len([]rune(content)); n > MaxMemoryContentRunes {
		content = TruncateRunes(content, MaxMemoryContentRunes)
		note = "（详情过长已截断）"
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	mem, err := m.memSvc.Update(uint(id), summary, content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已更新记忆：%s%s", mem.Summary, note), nil
}

// deleteMemories 批量删除记忆：ids 至少一个、且均为正整数。
func (m *memoryTool) deleteMemories(ctx context.Context, ids []float64) (string, error) {
	if len(ids) == 0 {
		return "", errors.New("manage_memory 删除记忆缺少 ids（至少提供一个记忆编号）")
	}

	// 规范化：过滤非法编号（≤0）并去重，保证后续计数精确、不重复删除。
	var validIDs []float64
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		u := uint(id)
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		validIDs = append(validIDs, id)
	}
	if len(validIDs) == 0 {
		return "未删除任何记忆（提供的编号均无效）", nil
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	deletedCount := 0
	var deletedID []uint
	var failedID []uint
	for _, id := range validIDs {
		u := uint(id)
		if err := m.memSvc.Delete(u); err != nil {
			if m.ctx != nil && m.ctx.Logger != nil {
				m.ctx.Logger.Warnw("manage_memory 批量删除其中一条失败", fastlog.Float64("id", id), fastlog.Error(err))
			}
			failedID = append(failedID, u)
			continue
		}
		deletedID = append(deletedID, u)
		deletedCount++
	}

	if deletedCount == 0 {
		return fmt.Sprintf("删除记忆失败：%v", failedID), nil
	}
	if len(failedID) > 0 {
		return fmt.Sprintf("已删除 %d 条记忆（%v），另有 %d 条删除失败（%v）", deletedCount, deletedID, len(failedID), failedID), nil
	}
	return fmt.Sprintf("已删除 %d 条记忆（%v）", deletedCount, deletedID), nil
}

// getMemory 查询单条记忆详情：id 必填、正整数。
func (m *memoryTool) getMemory(ctx context.Context, id float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_memory 查询记忆缺少有效的 id")
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	mem, err := m.memSvc.Get(uint(id))
	if err != nil {
		return "", err
	}
	return formatMemoryDetail(mem), nil
}

// listMemories 列出全部记忆：created_at 倒序，每行 id、描述、内容摘要。
func (m *memoryTool) listMemories(ctx context.Context) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	mems, err := m.memSvc.List()
	if err != nil {
		return "", err
	}
	if len(mems) == 0 {
		return "暂无记忆", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "当前记忆列表（共 %d 条）：\n", len(mems))
	for i := range mems {
		mm := mems[i]
		summary := mm.Summary
		content := mm.Content
		if len([]rune(content)) > 100 {
			content = TruncateRunes(content, 100)
		}
		fmt.Fprintf(&b, "id=%d · %s\n  内容：%s\n", mm.ID, summary, content)
	}
	return b.String(), nil
}

// formatMemoryDetail 格式化单条记忆详情（id/summary/content/时间）。
func formatMemoryDetail(mem *models.AIMemory) string {
	return fmt.Sprintf("记忆详情：\nid=%d\ndescription=%s\ncontent=%s\ncreated_at=%s\nupdated_at=%s",
		mem.ID, mem.Summary, mem.Content,
		mem.CreatedAt.Format(time.RFC3339), mem.UpdatedAt.Format(time.RFC3339))
}

// NewManageMemory 创建工具。构造器签名 = 工具的全部依赖。
func NewManageMemory(memSvc *services.MemoryService, ctx *Context) tool.InvokableTool {
	return &memoryTool{memSvc: memSvc, ctx: ctx}
}
