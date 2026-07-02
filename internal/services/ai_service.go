package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jot/internal/aicli"
	"jot/internal/models"

	"gorm.io/gorm"
)

// Message 表示 AI 对话中的一条消息
type Message struct {
	Role             string  `json:"role"`
	Content          string  `json:"content"`
	ReasoningContent string  `json:"reasoning_content"`
	ThinkingElapsed  float64 `json:"thinking_elapsed"`
	TotalElapsed     float64 `json:"total_elapsed"`
}

// AIConfig 表示 AI 服务配置
type AIConfig struct {
	Provider     string `json:"provider"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	TavilyAPIKey string `json:"tavily_api_key"`
}

// AIService 封装 AI 相关的业务逻辑操作
type AIService struct {
	db *gorm.DB
}

// NewAIService 创建一个新的 AIService 实例
func NewAIService(db *gorm.DB) *AIService {
	return &AIService{db: db}
}

// GetConfig 从 SettingService 读取 AI 配置
func (a *AIService) GetConfig() AIConfig {
	svc := NewSettingService(a.db)
	provider := svc.Get("ai_provider")
	if provider == "" {
		provider = "openai"
	}
	return AIConfig{
		Provider:     provider,
		BaseURL:      svc.Get("ai_base_url"),
		APIKey:       svc.Get("ai_api_key"),
		Model:        svc.Get("ai_model"),
		TavilyAPIKey: svc.Get("tavily_api_key"),
	}
}

// SaveConfig 保存 AI 配置到 SettingService
func (a *AIService) SaveConfig(cfg AIConfig) error {
	svc := NewSettingService(a.db)
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	if err := svc.Set("ai_provider", cfg.Provider); err != nil {
		return err
	}
	if err := svc.Set("ai_base_url", cfg.BaseURL); err != nil {
		return err
	}
	if err := svc.Set("ai_api_key", cfg.APIKey); err != nil {
		return err
	}
	if err := svc.Set("ai_model", cfg.Model); err != nil {
		return err
	}
	return svc.Set("tavily_api_key", cfg.TavilyAPIKey)
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

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
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

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
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
	LastMessage   string `json:"last_message"`
	MessageCount  int    `json:"message_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// GetAISessions 获取所有会话，按 updated_at DESC 排序，附带最后一条消息摘要
func (a *AIService) GetAISessions() []AISessionSummary {
	var sessions []models.AISession
	a.db.Order("updated_at DESC").Find(&sessions)

	result := make([]AISessionSummary, 0, len(sessions))
	for _, s := range sessions {
		summary := AISessionSummary{
			ID:            s.ID,
			Title:         s.Title,
			ContextTokens: s.ContextTokens,
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

// CreateAISession 创建新会话，返回会话 ID
func (a *AIService) CreateAISession() uint {
	session := models.AISession{Title: "新对话"}
	a.db.Create(&session)
	return session.ID
}

// DeleteAISession 删除会话及其所有消息
func (a *AIService) DeleteAISession(id uint) error {
	// 级联删除消息
	if err := a.db.Where("session_id = ?", id).Delete(&models.AIMessage{}).Error; err != nil {
		return err
	}
	return a.db.Delete(&models.AISession{}, id).Error
}

// RenameAISession 重命名会话
func (a *AIService) RenameAISession(id uint, title string) error {
	return a.db.Model(&models.AISession{}).Where("id = ?", id).Update("title", title).Error
}

// UpdateSessionContextTokens 更新会话的上下文 Token 数
func (a *AIService) UpdateSessionContextTokens(sessionID uint, tokens int) error {
	return a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("context_tokens", tokens).Error
}

// LoadAISessionMessages 加载会话的所有消息（按 created_at ASC）
func (a *AIService) LoadAISessionMessages(id uint) []Message {
	var msgs []models.AIMessage
	a.db.Where("session_id = ?", id).Order("created_at ASC").Find(&msgs)

	result := make([]Message, len(msgs))
	for i, m := range msgs {
		result[i] = Message{Role: m.Role, Content: m.Content, ReasoningContent: m.ReasoningContent, ThinkingElapsed: m.ThinkingElapsed, TotalElapsed: m.TotalElapsed}
	}
	return result
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
			CreatedAt:        now.Add(time.Duration(i) * time.Millisecond),
		}
		if err := a.db.Create(&m).Error; err != nil {
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
				return a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("title", title).Error
			}
		}
	}

	return nil
}

// ClearAISessionMessages 清空指定会话的所有消息（不删除会话本身）
func (a *AIService) ClearAISessionMessages(sessionID uint) error {
	return a.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{}).Error
}

// UpdateAIMessageContent 更新指定 AI 消息的 content 字段
func (a *AIService) UpdateAIMessageContent(id uint, content string) error {
	return a.db.Model(&models.AIMessage{}).Where("id = ?", id).Update("content", content).Error
}

// DeleteAIMessage 按 ID 删除单条 AI 消息
func (a *AIService) DeleteAIMessage(id uint) error {
	return a.db.Delete(&models.AIMessage{}, id).Error
}

// DeleteAIMessagesAfter 删除指定会话中在指定消息之后的所有消息（按 created_at 比较）
func (a *AIService) DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error) {
	// 先查目标消息的 created_at
	var msg models.AIMessage
	if err := a.db.Select("created_at").First(&msg, messageID).Error; err != nil {
		return 0, err
	}
	// 删除该 session 中 created_at 大于目标消息的所有记录
	result := a.db.Where("session_id = ? AND created_at > ?", sessionID, msg.CreatedAt).Delete(&models.AIMessage{})
	return result.RowsAffected, result.Error
}

// CountSessions 获取 AI 会话总数（不含软删除）
func (a *AIService) CountSessions() (int64, error) {
	var count int64
	err := a.db.Model(&models.AISession{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

// CountMessages 获取 AI 消息总数
func (a *AIService) CountMessages() (int64, error) {
	var count int64
	err := a.db.Model(&models.AIMessage{}).Count(&count).Error
	return count, err
}
