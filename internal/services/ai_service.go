package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gitee.com/MM-Q/fastlog"
	"jot/internal/aicli"
	"jot/internal/models"

	"gorm.io/gorm"
)

// Message 表示 AI 对话中的一条消息
type Message struct {
	ID               uint    `json:"id"`
	Role             string  `json:"role"`
	Content          string  `json:"content"`
	ReasoningContent string  `json:"reasoning_content"`
	ThinkingElapsed  float64 `json:"thinking_elapsed"`
	TotalElapsed     float64 `json:"total_elapsed"`
	Tokens           int     `json:"tokens"`
	SearchSources    string  `json:"search_sources"`
	RecallCards      string  `json:"recall_cards"`
}

// AIConfig 表示 AI 服务配置
type AIConfig struct {
	Provider          string `json:"provider"`
	BaseURL           string `json:"base_url"`
	APIKey            string `json:"api_key"`
	Model             string `json:"model"`
	TavilyAPIKey      string `json:"tavily_api_key"`
	ZhihuAccessSecret string `json:"zhihu_access_secret"`
}

// SessionConfig 表示 AI 会话的操作栏配置，用于前端交互
type SessionConfig struct {
	ModelName                string `json:"model_name"`
	EnableThinking           bool   `json:"enable_thinking"`
	ZhihuSearchEnabled       bool   `json:"zhihu_search_enabled"`
	ZhihuGlobalSearchEnabled bool   `json:"zhihu_global_search_enabled"`
	TavilySearchEnabled      bool   `json:"tavily_search_enabled"`
	EnableCardRecall         bool   `json:"enable_card_recall"`
	ReferencedNotes          string `json:"referenced_notes"`
	EnabledSkills            string `json:"enabled_skills"`
	RoleplayNotes            string `json:"roleplay_notes"`
}

// AIService 封装 AI 相关的业务逻辑操作
type AIService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

// NewAIService 创建一个新的 AIService 实例
func NewAIService(db *gorm.DB, logger *fastlog.Logger) *AIService {
	return &AIService{db: db, logger: logger}
}

// GetSkillPrompts 根据 skill key 列表从数据库查询并拼接技能提示词
// translateArgs 用于翻译技能的 {source}/{target} 占位符替换, 可为 nil
func (s *AIService) GetSkillPrompts(skillIds []string, translateArgs map[string]string) (string, error) {
	if len(skillIds) == 0 {
		return "", nil
	}
	var prompts []models.AIPrompt
	if err := s.db.Where("key IN ?", skillIds).Find(&prompts).Error; err != nil {
		s.logger.Errorw("AIService.GetSkillPrompts 失败", fastlog.Error(err))
		return "", fmt.Errorf("查询技能提示词失败: %w", err)
	}
	if len(prompts) == 0 {
		return "", nil
	}
	var parts []string
	for _, p := range prompts {
		content := p.Content
		// 翻译技能: 替换 {source} 和 {target} 占位符
		if p.Key == "skill_translate" {
			source := "English"
			target := "Chinese"
			if translateArgs != nil {
				if v, ok := translateArgs["source"]; ok {
					source = v
				}
				if v, ok := translateArgs["target"]; ok {
					target = v
				}
			}
			content = strings.ReplaceAll(content, "{source}", source)
			content = strings.ReplaceAll(content, "{target}", target)
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n"), nil
}

// GetConfig 从 SettingService 读取 AI 配置
func (a *AIService) GetConfig() AIConfig {
	svc := NewSettingService(a.db)
	cfg := AIConfig{
		Provider:          svc.Get("ai_provider"),
		BaseURL:           svc.Get("ai_base_url"),
		APIKey:            svc.Get("ai_api_key"),
		Model:             svc.Get("ai_model"),
		TavilyAPIKey:      svc.Get("tavily_api_key"),
		ZhihuAccessSecret: svc.Get("zhihu_access_secret"),
	}
	cfg.APIKey = DecodeB64(cfg.APIKey)
	cfg.TavilyAPIKey = DecodeB64(cfg.TavilyAPIKey)
	cfg.ZhihuAccessSecret = DecodeB64(cfg.ZhihuAccessSecret)
	return cfg
}

// SaveConfig 保存 AI 配置到 SettingService
func (a *AIService) SaveConfig(cfg AIConfig) error {
	svc := NewSettingService(a.db)
	if err := svc.Set("ai_provider", cfg.Provider); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("ai_base_url", cfg.BaseURL); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("ai_api_key", EncodeB64(cfg.APIKey)); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("ai_model", cfg.Model); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("tavily_api_key", EncodeB64(cfg.TavilyAPIKey)); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("zhihu_access_secret", EncodeB64(cfg.ZhihuAccessSecret)); err != nil {
		a.logger.Errorw("AIService.SaveConfig 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// CallAI 调用 AI 接口（非流式）
func (a *AIService) CallAI(messages []Message) (string, error) {
	cfg := a.GetConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := aicli.NewClient(aicli.Config{
		Provider: cfg.Provider,
		BaseURL:  cfg.BaseURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})

	// Convert services.Message to aicli.Message
	aicliMsgs := make([]aicli.Message, len(messages))
	for i, m := range messages {
		aicliMsgs[i] = aicli.Message{Role: m.Role, Content: m.Content}
	}

	content, _, err := client.Chat(ctx, aicliMsgs, false)
	if err != nil {
		a.logger.Errorw("AIService.CallAI 失败", fastlog.Error(err))
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	return content, nil
}

// CallAIStream 流式调用 AI 接口
// 通过 onChunk/onThinking/onDone/onError 回调逐块推送内容
// onThinking 用于深度思考模型的 reasoning_content 字段
func (a *AIService) CallAIStream(ctx context.Context, messages []Message, thinkingEnabled bool, onChunk func(string), onThinking func(string), onDone func(string, float64, float64), onError func(string)) {
	cfg := a.GetConfig()

	client := aicli.NewClient(aicli.Config{
		Provider: cfg.Provider,
		BaseURL:  cfg.BaseURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})

	// Convert services.Message to aicli.Message
	aicliMsgs := make([]aicli.Message, len(messages))
	for i, m := range messages {
		aicliMsgs[i] = aicli.Message{Role: m.Role, Content: m.Content}
	}

	callbacks := aicli.StreamCallbacks{
		OnChunk:    onChunk,
		OnThinking: onThinking,
		OnDone:     onDone,
		OnError:    onError,
	}

	client.Stream(ctx, aicliMsgs, thinkingEnabled, callbacks)
}

// TestConnection 测试 AI 服务连通性
// - openai: 调用 /models 端点
// - ollama: 调用 /api/tags 端点
// - other: 尝试创建一个简单调用
func (a *AIService) TestConnection(cfg AIConfig) (bool, error) {
	switch cfg.Provider {
	case "openai":
		return testOpenAIConnection(cfg)
	case "ollama":
		return testOllamaConnection(cfg)
	default:
		return testGenericConnection(cfg)
	}
}

// testOpenAIConnection 测试 OpenAI 兼容 API 的连通性
func testOpenAIConnection(cfg AIConfig) (bool, error) {
	url := cfg.BaseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, fmt.Errorf("服务器返回状态码 %d", resp.StatusCode)
}

// testOllamaConnection 测试 Ollama 连通性（调用 /api/tags）
func testOllamaConnection(cfg AIConfig) (bool, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	url := baseURL + "/api/tags"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, fmt.Errorf("服务器返回状态码 %d", resp.StatusCode)
}

// testGenericConnection 通过创建 AI 客户端并执行简单调用来测试连通性
func testGenericConnection(cfg AIConfig) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := aicli.NewClient(aicli.Config{
		Provider: cfg.Provider,
		BaseURL:  cfg.BaseURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})

	msgs := []aicli.Message{{Role: "user", Content: "test"}}
	_, _, err := client.Chat(ctx, msgs, false)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	return true, nil
}

// modelsResponse 表示 /models 接口的响应体
type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// FetchModels 获取可用模型列表
// - openai: 调用 /models 端点
// - ollama: 调用 /api/tags 端点
// - other: 返回空列表
func (a *AIService) FetchModels(cfg AIConfig) ([]string, error) {
	switch cfg.Provider {
	case "openai":
		return fetchOpenAIModels(cfg)
	case "ollama":
		return fetchOllamaModels(cfg)
	default:
		return []string{}, nil
	}
}

// fetchOpenAIModels 从 OpenAI 兼容 API 获取模型列表
func fetchOpenAIModels(cfg AIConfig) ([]string, error) {
	url := cfg.BaseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI 调用失败: HTTP %d", resp.StatusCode)
	}

	var modelsResp modelsResponse
	if err := json.Unmarshal(respBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	models := make([]string, 0, len(modelsResp.Data))
	for _, item := range modelsResp.Data {
		models = append(models, item.ID)
	}

	return models, nil
}

// ollamaTagResponse Ollama /api/tags 响应结构
type ollamaTagResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// fetchOllamaModels 从 Ollama API 获取本地模型列表
func fetchOllamaModels(cfg AIConfig) ([]string, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	url := baseURL + "/api/tags"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取模型列表失败: HTTP %d", resp.StatusCode)
	}

	var tagResp ollamaTagResponse
	if err := json.Unmarshal(respBody, &tagResp); err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	models := make([]string, 0, len(tagResp.Models))
	for _, item := range tagResp.Models {
		models = append(models, item.Name)
	}

	return models, nil
}

// AISessionSummary 会话列表项（含最后一条消息摘要）
type AISessionSummary struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	ContextTokens int    `json:"context_tokens"`
	IsPinned      bool   `json:"is_pinned"`
	LastMessage   string `json:"last_message"`
	MessageCount  int    `json:"message_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// GetAISessions 获取所有会话，置顶优先，然后按标题、更新时间排序，附带最后一条消息摘要
func (a *AIService) GetAISessions() []AISessionSummary {
	var sessions []models.AISession
	a.db.Order("CASE WHEN is_pinned = 1 THEN 0 ELSE 1 END, updated_at DESC").Find(&sessions)

	result := make([]AISessionSummary, 0, len(sessions))
	for _, s := range sessions {
		summary := AISessionSummary{
			ID:            s.ID,
			Title:         s.Title,
			ContextTokens: s.ContextTokens,
			IsPinned:      s.IsPinned,
			CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt:     s.UpdatedAt.Format("2006-01-02 15:04"),
		}

		// 查消息数
		var count int64
		a.db.Model(&models.AIMessage{}).Where("session_id = ?", s.ID).Count(&count)
		summary.MessageCount = int(count)

		// 取最后一条消息内容（截断作为摘要）
		var lastMsg models.AIMessage
		if err := a.db.Where("session_id = ?", s.ID).Order("created_at DESC").First(&lastMsg).Error; err == nil {
			text := lastMsg.Content
			if len([]rune(text)) > 60 {
				text = string([]rune(text)[:60]) + "..."
			}
			summary.LastMessage = text
		}

		result = append(result, summary)
	}
	return result
}

// CreateAISession 创建新会话，返回会话 ID，并自动创建默认配置
func (a *AIService) CreateAISession() uint {
	session := models.AISession{Title: "新对话"}
	a.db.Create(&session)
	// 自动创建默认会话配置
	if err := a.CreateDefaultSessionConfig(session.ID); err != nil {
		// 静默失败，不影响会话创建
		_ = err
	}
	return session.ID
}

// CreateDefaultSessionConfig 从全局设置为指定会话创建默认配置
func (a *AIService) CreateDefaultSessionConfig(sessionID uint) error {
	svc := NewSettingService(a.db)
	cfg := SessionConfig{
		ModelName:                svc.Get("ai_model"),
		EnableThinking:           parseBoolSetting(svc.Get("ai_thinking_enabled")),
		ZhihuSearchEnabled:       parseBoolSetting(svc.Get("zhihu_search_enabled")),
		ZhihuGlobalSearchEnabled: parseBoolSetting(svc.Get("zhihu_global_search_enabled")),
		TavilySearchEnabled:      parseBoolSetting(svc.Get("tavily_search_enabled")),
		EnableCardRecall:         parseBoolSetting(svc.Get("ai_card_recall_enabled")),
		ReferencedNotes:          "[]",
		EnabledSkills:            "{}",
		RoleplayNotes:            "[]",
	}
	record := models.AISessionConfig{
		SessionID:                sessionID,
		ModelName:                cfg.ModelName,
		EnableThinking:           cfg.EnableThinking,
		ZhihuSearchEnabled:       cfg.ZhihuSearchEnabled,
		ZhihuGlobalSearchEnabled: cfg.ZhihuGlobalSearchEnabled,
		TavilySearchEnabled:      cfg.TavilySearchEnabled,
		EnableCardRecall:         cfg.EnableCardRecall,
		ReferencedNotes:          cfg.ReferencedNotes,
		EnabledSkills:            cfg.EnabledSkills,
		RoleplayNotes:            cfg.RoleplayNotes,
	}
	if err := a.db.Create(&record).Error; err != nil {
		a.logger.Errorw("AIService.CreateDefaultSessionConfig 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// SaveSessionConfig 保存会话配置
func (a *AIService) SaveSessionConfig(sessionID uint, cfg SessionConfig) error {
	err := a.db.Where("session_id = ?", sessionID).Assign(map[string]interface{}{
		"model_name":                  cfg.ModelName,
		"enable_thinking":             cfg.EnableThinking,
		"zhihu_search_enabled":        cfg.ZhihuSearchEnabled,
		"zhihu_global_search_enabled": cfg.ZhihuGlobalSearchEnabled,
		"tavily_search_enabled":       cfg.TavilySearchEnabled,
		"enable_card_recall":          cfg.EnableCardRecall,
		"referenced_notes":            cfg.ReferencedNotes,
		"enabled_skills":              cfg.EnabledSkills,
		"roleplay_notes":              cfg.RoleplayNotes,
	}).FirstOrCreate(&models.AISessionConfig{SessionID: sessionID}).Error
	if err != nil {
		a.logger.Errorw("AIService.SaveSessionConfig 失败", fastlog.Error(err))
	}
	return err
}

// LoadSessionConfig 加载会话配置，如果不存在则自动创建默认配置并返回
func (a *AIService) LoadSessionConfig(sessionID uint) SessionConfig {
	var record models.AISessionConfig
	err := a.db.Where("session_id = ?", sessionID).First(&record).Error
	if err != nil {
		// 配置不存在时自动创建默认配置
		_ = a.CreateDefaultSessionConfig(sessionID)
		// 重新读取
		if err := a.db.Where("session_id = ?", sessionID).First(&record).Error; err != nil {
			// 极端情况仍失败时返回空默认值
			return SessionConfig{
				ReferencedNotes: "[]",
				EnabledSkills:   "{}",
				RoleplayNotes:   "[]",
			}
		}
	}
	return SessionConfig{
		ModelName:                record.ModelName,
		EnableThinking:           record.EnableThinking,
		ZhihuSearchEnabled:       record.ZhihuSearchEnabled,
		ZhihuGlobalSearchEnabled: record.ZhihuGlobalSearchEnabled,
		TavilySearchEnabled:      record.TavilySearchEnabled,
		EnableCardRecall:         record.EnableCardRecall,
		ReferencedNotes:          record.ReferencedNotes,
		EnabledSkills:            record.EnabledSkills,
		RoleplayNotes:            record.RoleplayNotes,
	}
}

// DeleteAISession 删除会话及其所有消息和配置
func (a *AIService) DeleteAISession(id uint) error {
	// 级联删除消息
	if err := a.db.Where("session_id = ?", id).Delete(&models.AIMessage{}).Error; err != nil {
		a.logger.Errorw("AIService.DeleteAISession 失败", fastlog.Error(err))
		return err
	}
	// 删除会话配置
	if err := a.db.Where("session_id = ?", id).Delete(&models.AISessionConfig{}).Error; err != nil {
		a.logger.Errorw("AIService.DeleteAISession 失败", fastlog.Error(err))
		return err
	}
	if err := a.db.Delete(&models.AISession{}, id).Error; err != nil {
		a.logger.Errorw("AIService.DeleteAISession 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// DeleteEmptyAISessions 删除没有关联消息的 AI 会话（同时清理对应的会话配置）
func (a *AIService) DeleteEmptyAISessions() int64 {
	// 使用子查询找出没有消息的会话并删除
	var count int64
	a.db.Raw("SELECT COUNT(*) FROM ai_sessions s WHERE NOT EXISTS (SELECT 1 FROM ai_messages m WHERE m.session_id = s.id)").Scan(&count)

	// 删除对应的会话配置
	a.db.Exec("DELETE FROM ai_session_configs WHERE session_id IN (SELECT s.id FROM ai_sessions s WHERE NOT EXISTS (SELECT 1 FROM ai_messages m WHERE m.session_id = s.id))")
	// 删除会话
	a.db.Exec("DELETE FROM ai_sessions WHERE id IN (SELECT s.id FROM ai_sessions s WHERE NOT EXISTS (SELECT 1 FROM ai_messages m WHERE m.session_id = s.id))")

	return count
}

// DeleteOrphanMessages 删除指向不存在会话的孤儿 AI 消息
func (a *AIService) DeleteOrphanMessages() int64 {
	result := a.db.Where("session_id NOT IN (SELECT id FROM ai_sessions)").Delete(&models.AIMessage{})
	return result.RowsAffected
}

// RenameAISession 重命名会话
func (a *AIService) RenameAISession(id uint, title string) error {
	err := a.db.Model(&models.AISession{}).Where("id = ?", id).Update("title", title).Error
	if err != nil {
		a.logger.Errorw("AIService.RenameAISession 失败", fastlog.Error(err))
	}
	return err
}

// TogglePinAISession 切换会话置顶状态
func (a *AIService) TogglePinAISession(id uint) error {
	var session models.AISession
	if err := a.db.First(&session, id).Error; err != nil {
		a.logger.Errorw("AIService.TogglePinAISession 失败", fastlog.Error(err))
		return err
	}
	err := a.db.Model(&session).Update("is_pinned", gorm.Expr("NOT is_pinned")).Error
	if err != nil {
		a.logger.Errorw("AIService.TogglePinAISession 失败", fastlog.Error(err))
	}
	return err
}

// UpdateSessionContextTokens 更新会话的上下文 Token 数
func (a *AIService) UpdateSessionContextTokens(sessionID uint, tokens int) error {
	err := a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("context_tokens", tokens).Error
	if err != nil {
		a.logger.Errorw("AIService.UpdateSessionContextTokens 失败", fastlog.Error(err))
	}
	return err
}

// LoadAISessionMessages 加载会话的所有消息（按 created_at ASC）
func (a *AIService) LoadAISessionMessages(id uint) []Message {
	var msgs []models.AIMessage
	a.db.Where("session_id = ?", id).Order("created_at ASC").Find(&msgs)

	result := make([]Message, len(msgs))
	for i, m := range msgs {
		result[i] = Message{ID: m.ID, Role: m.Role, Content: m.Content, ReasoningContent: m.ReasoningContent, ThinkingElapsed: m.ThinkingElapsed, TotalElapsed: m.TotalElapsed, Tokens: m.Tokens, SearchSources: m.SearchSources, RecallCards: m.RecallCards}
	}
	return result
}

// LoadAISessionMessagesPaginated 分页加载会话消息（游标分页，基于 ID 降序取 N 条后反转返回）
// beforeID=0 表示取最新消息，否则取比 beforeID 更早的消息
func (a *AIService) LoadAISessionMessagesPaginated(sessionID uint, limit int, beforeID uint) []Message {
	var msgs []models.AIMessage
	query := a.db.Where("session_id = ?", sessionID)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}
	query.Order("id DESC").Limit(limit).Find(&msgs)

	// 反转为 ASC 顺序
	result := make([]Message, len(msgs))
	for i, m := range msgs {
		result[len(msgs)-1-i] = Message{ID: m.ID, Role: m.Role, Content: m.Content, ReasoningContent: m.ReasoningContent, ThinkingElapsed: m.ThinkingElapsed, TotalElapsed: m.TotalElapsed, Tokens: m.Tokens, SearchSources: m.SearchSources, RecallCards: m.RecallCards}
	}
	return result
}

// TruncateAISessionAtMessage 删除指定消息及该消息之后的所有消息（用于删除/重发操作）
// 事务内：先查目标消息的 created_at → 删除该 session 中 created_at >= 该时间戳的所有消息
func (a *AIService) TruncateAISessionAtMessage(sessionID uint, msgID uint) error {
	var msg models.AIMessage
	if err := a.db.Select("created_at").First(&msg, msgID).Error; err != nil {
		a.logger.Errorw("AIService.TruncateAISessionAtMessage 查消息失败", fastlog.Error(err))
		return fmt.Errorf("消息不存在: %w", err)
	}
	return a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("session_id = ? AND created_at >= ?", sessionID, msg.CreatedAt).Delete(&models.AIMessage{})
		if result.Error != nil {
			a.logger.Errorw("AIService.TruncateAISessionAtMessage 删除失败", fastlog.Error(result.Error))
			return result.Error
		}
		// 更新会话 updated_at
		if err := tx.Model(&models.AISession{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error; err != nil {
			a.logger.Errorw("AIService.TruncateAISessionAtMessage 更新会话时间失败", fastlog.Error(err))
			return err
		}
		return nil
	})
}

// TruncateAISessionAfterMessage 删除指定消息之后的所有消息，保留该消息本身（用于编辑/重新生成操作）
// 事务内：先查目标消息的 created_at → 删除该 session 中 created_at > 该时间戳的所有消息
func (a *AIService) TruncateAISessionAfterMessage(sessionID uint, msgID uint) error {
	var msg models.AIMessage
	if err := a.db.Select("created_at").First(&msg, msgID).Error; err != nil {
		a.logger.Errorw("AIService.TruncateAISessionAfterMessage 查消息失败", fastlog.Error(err))
		return fmt.Errorf("消息不存在: %w", err)
	}
	return a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("session_id = ? AND created_at > ?", sessionID, msg.CreatedAt).Delete(&models.AIMessage{})
		if result.Error != nil {
			a.logger.Errorw("AIService.TruncateAISessionAfterMessage 删除失败", fastlog.Error(result.Error))
			return result.Error
		}
		// 更新会话 updated_at
		if err := tx.Model(&models.AISession{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error; err != nil {
			a.logger.Errorw("AIService.TruncateAISessionAfterMessage 更新会话时间失败", fastlog.Error(err))
			return err
		}
		return nil
	})
}

// GetSessionContextTokens 返回指定会话的 context_tokens 字段值
func (a *AIService) GetSessionContextTokens(sessionID uint) (int, error) {
	var session models.AISession
	if err := a.db.Select("context_tokens").First(&session, sessionID).Error; err != nil {
		a.logger.Errorw("AIService.GetSessionContextTokens 失败", fastlog.Error(err))
		return 0, err
	}
	return session.ContextTokens, nil
}

// SumSessionTokens 返回指定会话所有消息的 tokens 累计和
func (a *AIService) SumSessionTokens(sessionID uint) (int, error) {
	var total int64
	err := a.db.Model(&models.AIMessage{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(SUM(tokens), 0)").
		Scan(&total).Error
	if err != nil {
		a.logger.Errorw("AIService.SumSessionTokens 失败", fastlog.Error(err))
		return 0, err
	}
	return int(total), nil
}

// UpdateAIMessageTokens 更新指定消息的 tokens 字段
func (a *AIService) UpdateAIMessageTokens(id uint, tokens int) error {
	err := a.db.Model(&models.AIMessage{}).Where("id = ?", id).Update("tokens", tokens).Error
	if err != nil {
		a.logger.Errorw("AIService.UpdateAIMessageTokens 失败", fastlog.Error(err))
	}
	return err
}

// SaveAIMessage 保存单条 AI 消息到指定会话，返回消息 ID
// 同时更新会话 updated_at，如果是首条用户消息则自动生成标题
func (a *AIService) SaveAIMessage(sessionID uint, msg Message) (uint, error) {
	now := time.Now()
	m := models.AIMessage{
		SessionID:        sessionID,
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ThinkingElapsed:  msg.ThinkingElapsed,
		TotalElapsed:     msg.TotalElapsed,
		Tokens:           msg.Tokens,
		SearchSources:    msg.SearchSources,
		RecallCards:      msg.RecallCards,
		CreatedAt:        now,
	}
	if err := a.db.Create(&m).Error; err != nil {
		a.logger.Errorw("AIService.SaveAIMessage 失败", fastlog.Error(err))
		return 0, err
	}

	// 更新会话 updated_at
	if err := a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error; err != nil {
		a.logger.Errorw("AIService.SaveAIMessage 更新会话时间失败", fastlog.Error(err))
	}

	// 首轮对话自动生成标题（取第一条用户消息前 30 字）
	if msg.Role == "user" {
		var session models.AISession
		if err := a.db.First(&session, sessionID).Error; err == nil && session.Title == "新对话" {
			runes := []rune(msg.Content)
			title := msg.Content
			if len(runes) > 30 {
				title = string(runes[:30]) + "..."
			}
			if err := a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("title", title).Error; err != nil {
				a.logger.Errorw("AIService.SaveAIMessage 自动标题失败", fastlog.Error(err))
			}
		}
	}

	return m.ID, nil
}

// SaveAIMessages 保存一轮对话消息（user + assistant）到指定会话
// 同时更新会话 updated_at，如果是首轮对话则自动生成标题
// 使用显式 CreatedAt 偏移确保同轮消息时间戳严格递增，避免 ORDER BY 歧义
func (a *AIService) SaveAIMessages(sessionID uint, messages []Message) error {
	now := time.Now()
	for i, msg := range messages {
		m := models.AIMessage{
			SessionID:        sessionID,
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
			ThinkingElapsed:  msg.ThinkingElapsed,
			TotalElapsed:     msg.TotalElapsed,
			Tokens:           msg.Tokens,
			SearchSources:    msg.SearchSources,
			RecallCards:      msg.RecallCards,
			CreatedAt:        now.Add(time.Duration(i) * time.Millisecond),
		}
		if err := a.db.Create(&m).Error; err != nil {
			a.logger.Errorw("AIService.SaveAIMessages 失败", fastlog.Error(err))
			return err
		}
	}

	// 更新会话 updated_at
	var s models.AISession
	a.db.First(&s, sessionID)
	a.db.Model(&s).Update("updated_at", time.Now())

	// 如果是首轮对话，自动生成标题（取第一条 user 消息前 30 字）
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err != nil {
		a.logger.Errorw("AIService.SaveAIMessages 失败", fastlog.Error(err))
		return err
	}
	if session.Title == "新对话" {
		for _, msg := range messages {
			if msg.Role == "user" {
				title := msg.Content
				runes := []rune(title)
				if len(runes) > 30 {
					title = string(runes[:30]) + "..."
				}
				if err := a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("title", title).Error; err != nil {
					a.logger.Errorw("AIService.SaveAIMessages 失败", fastlog.Error(err))
					return err
				}
				return nil
			}
		}
	}

	return nil
}

// ReplaceAISessionMessages 原子替换指定会话的所有消息（清空 + 批量写入）
// 整个操作在事务中完成，失败时自动回滚
func (a *AIService) ReplaceAISessionMessages(sessionID uint, messages []Message) error {
	err := a.db.Transaction(func(tx *gorm.DB) error {
		// 清空现有消息
		if err := tx.Where("session_id = ?", sessionID).Delete(&models.AIMessage{}).Error; err != nil {
			a.logger.Errorw("AIService.ReplaceAISessionMessages 清空失败", fastlog.Error(err))
			return err
		}

		// 批量写入新消息
		now := time.Now()
		for i, msg := range messages {
			m := models.AIMessage{
				SessionID:        sessionID,
				Role:             msg.Role,
				Content:          msg.Content,
				ReasoningContent: msg.ReasoningContent,
				ThinkingElapsed:  msg.ThinkingElapsed,
				TotalElapsed:     msg.TotalElapsed,
				Tokens:           msg.Tokens,
				SearchSources:    msg.SearchSources,
				RecallCards:      msg.RecallCards,
				CreatedAt:        now.Add(time.Duration(i) * time.Millisecond),
			}
			if err := tx.Create(&m).Error; err != nil {
				a.logger.Errorw("AIService.ReplaceAISessionMessages 写入失败", fastlog.Error(err))
				return err
			}
		}

		// 更新会话 updated_at
		if err := tx.Model(&models.AISession{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error; err != nil {
			a.logger.Errorw("AIService.ReplaceAISessionMessages 更新会话时间失败", fastlog.Error(err))
			return err
		}

		// 自动生成标题（首轮对话）
		var session models.AISession
		if err := tx.First(&session, sessionID).Error; err != nil {
			a.logger.Errorw("AIService.ReplaceAISessionMessages 查会话失败", fastlog.Error(err))
			return err
		}
		if session.Title == "新对话" {
			for _, msg := range messages {
				if msg.Role == "user" {
					title := msg.Content
					runes := []rune(title)
					if len(runes) > 30 {
						title = string(runes[:30]) + "..."
					}
					if err := tx.Model(&models.AISession{}).Where("id = ?", sessionID).Update("title", title).Error; err != nil {
						a.logger.Errorw("AIService.ReplaceAISessionMessages 自动标题失败", fastlog.Error(err))
						return err
					}
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		a.logger.Errorw("AIService.ReplaceAISessionMessages 事务失败", fastlog.Error(err))
		return fmt.Errorf("替换会话消息失败: %w", err)
	}
	return nil
}

// ClearAISessionMessages 清空指定会话的所有消息（不删除会话本身）
func (a *AIService) ClearAISessionMessages(sessionID uint) error {
	err := a.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{}).Error
	if err != nil {
		a.logger.Errorw("AIService.ClearAISessionMessages 失败", fastlog.Error(err))
	}
	return err
}

// UpdateAIMessageContent 更新指定 AI 消息的 content 字段
func (a *AIService) UpdateAIMessageContent(id uint, content string) error {
	err := a.db.Model(&models.AIMessage{}).Where("id = ?", id).Update("content", content).Error
	if err != nil {
		a.logger.Errorw("AIService.UpdateAIMessageContent 失败", fastlog.Error(err))
	}
	return err
}

// UpdateLastUserMessageTokens 更新指定会话中最后一条用户消息的 tokens（用于编辑后同步）
func (a *AIService) UpdateLastUserMessageTokens(sessionID uint, tokens int) error {
	var msg models.AIMessage
	result := a.db.Where("session_id = ? AND role = 'user'", sessionID).Order("id DESC").Limit(1).Find(&msg)
	if result.RowsAffected == 0 {
		return nil
	}
	err := a.db.Model(&msg).Update("tokens", tokens).Error
	if err != nil {
		a.logger.Errorw("AIService.UpdateLastUserMessageTokens 失败", fastlog.Error(err))
	}
	return err
}

// DeleteAIMessage 按 ID 删除单条 AI 消息
func (a *AIService) DeleteAIMessage(id uint) error {
	err := a.db.Delete(&models.AIMessage{}, id).Error
	if err != nil {
		a.logger.Errorw("AIService.DeleteAIMessage 失败", fastlog.Error(err))
	}
	return err
}

// DeleteAIMessagesAfter 删除指定会话中在指定消息之后的所有消息（按 created_at 比较）
func (a *AIService) DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error) {
	// 先查目标消息的 created_at
	var msg models.AIMessage
	if err := a.db.Select("created_at").First(&msg, messageID).Error; err != nil {
		a.logger.Errorw("AIService.DeleteAIMessagesAfter 失败", fastlog.Error(err))
		return 0, err
	}
	// 删除该 session 中 created_at 大于目标消息的所有记录
	result := a.db.Where("session_id = ? AND created_at > ?", sessionID, msg.CreatedAt).Delete(&models.AIMessage{})
	if result.Error != nil {
		a.logger.Errorw("AIService.DeleteAIMessagesAfter 失败", fastlog.Error(result.Error))
	}
	return result.RowsAffected, result.Error
}

// ClearAllAISessions 清空所有 AI 会话、消息及会话配置
func (a *AIService) ClearAllAISessions() error {
	if err := a.db.Where("1 = 1").Delete(&models.AIMessage{}).Error; err != nil {
		a.logger.Errorw("AIService.ClearAllAISessions 失败", fastlog.Error(err))
		return err
	}
	if err := a.db.Where("1 = 1").Delete(&models.AISessionConfig{}).Error; err != nil {
		a.logger.Errorw("AIService.ClearAllAISessions 失败", fastlog.Error(err))
		return err
	}
	if err := a.db.Where("1 = 1").Delete(&models.AISession{}).Error; err != nil {
		a.logger.Errorw("AIService.ClearAllAISessions 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// CountSessions 获取 AI 会话总数（不含软删除）
func (a *AIService) CountSessions() (int64, error) {
	var count int64
	err := a.db.Model(&models.AISession{}).Where("deleted_at IS NULL").Count(&count).Error
	if err != nil {
		a.logger.Errorw("AIService.CountSessions 失败", fastlog.Error(err))
	}
	return count, err
}

// CountMessages 获取 AI 消息总数
func (a *AIService) CountMessages() (int64, error) {
	var count int64
	err := a.db.Model(&models.AIMessage{}).Count(&count).Error
	if err != nil {
		a.logger.Errorw("AIService.CountMessages 失败", fastlog.Error(err))
	}
	return count, err
}

// SumTokens 获取 AI 消息的总 token 数
func (a *AIService) SumTokens() (int64, error) {
	var total int64
	err := a.db.Model(&models.AIMessage{}).Select("COALESCE(SUM(tokens), 0)").Scan(&total).Error
	if err != nil {
		a.logger.Errorw("AIService.SumTokens 失败", fastlog.Error(err))
	}
	return total, err
}

// AvgResponseTime 获取 AI 消息的平均响应时间（毫秒）
func (a *AIService) AvgResponseTime() (float64, error) {
	var avg float64
	err := a.db.Model(&models.AIMessage{}).Select("COALESCE(AVG(total_elapsed), 0)").Where("total_elapsed > 0").Scan(&avg).Error
	if err != nil {
		a.logger.Errorw("AIService.AvgResponseTime 失败", fastlog.Error(err))
	}
	return avg, err
}

// AvgThinkingTime 获取 AI 消息的平均思考时间（毫秒）
func (a *AIService) AvgThinkingTime() (float64, error) {
	var avg float64
	err := a.db.Model(&models.AIMessage{}).Select("COALESCE(AVG(thinking_elapsed), 0)").Where("thinking_elapsed > 0").Scan(&avg).Error
	if err != nil {
		a.logger.Errorw("AIService.AvgThinkingTime 失败", fastlog.Error(err))
	}
	return avg, err
}

// MaxResponseTime 获取 AI 消息的最大响应时间（毫秒）
func (a *AIService) MaxResponseTime() (float64, error) {
	var max float64
	err := a.db.Model(&models.AIMessage{}).Select("COALESCE(MAX(total_elapsed), 0)").Scan(&max).Error
	if err != nil {
		a.logger.Errorw("AIService.MaxResponseTime 失败", fastlog.Error(err))
	}
	return max, err
}
