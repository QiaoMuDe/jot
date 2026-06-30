package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jot/internal/models"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
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
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
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
		Provider: provider,
		BaseURL:  svc.Get("ai_base_url"),
		APIKey:   svc.Get("ai_api_key"),
		Model:    svc.Get("ai_model"),
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
	return svc.Set("ai_model", cfg.Model)
}

// createLLM 根据配置创建 LangChainGo LLM 实例
func createLLM(ctx context.Context, cfg AIConfig) (llms.Model, error) {
	switch cfg.Provider {
	case "openai":
		opts := []openai.Option{}
		if cfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
		}
		if cfg.APIKey != "" {
			opts = append(opts, openai.WithToken(cfg.APIKey))
		}
		if cfg.Model != "" {
			opts = append(opts, openai.WithModel(cfg.Model))
		}
		return openai.New(opts...)
	case "ollama":
		opts := []ollama.Option{}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		if cfg.Model != "" {
			opts = append(opts, ollama.WithModel(cfg.Model))
		}
		return ollama.New(opts...)
	default:
		return nil, fmt.Errorf("不支持的 AI 服务商: %s", cfg.Provider)
	}
}

// convertMessages 将 services.Message 转换为 LangChainGo 的 MessageContent 列表
func convertMessages(messages []Message) []llms.MessageContent {
	result := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			result = append(result, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))
		case "assistant":
			result = append(result, llms.TextParts(llms.ChatMessageTypeAI, msg.Content))
		case "system":
			result = append(result, llms.TextParts(llms.ChatMessageTypeSystem, msg.Content))
		default:
			result = append(result, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))
		}
	}
	return result
}

// CallAI 调用 AI 接口（非流式）
func (a *AIService) CallAI(messages []Message) (string, error) {
	cfg := a.GetConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	llm, err := createLLM(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	msgContents := convertMessages(messages)

	resp, err := llm.GenerateContent(ctx, msgContents)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("AI 调用失败: 响应中没有 choices")
	}

	return resp.Choices[0].Content, nil
}

// CallAIStream 流式调用 AI 接口
// 通过 onChunk/onThinking/onDone/onError 回调逐块推送内容
// onThinking 用于深度思考模型的 reasoning_content 字段
func (a *AIService) CallAIStream(ctx context.Context, messages []Message, thinkingEnabled bool, onChunk func(string), onThinking func(string), onDone func(string, float64, float64), onError func(string)) {
	cfg := a.GetConfig()

	llm, err := createLLM(ctx, cfg)
	if err != nil {
		onError("创建 LLM 失败: " + err.Error())
		return
	}

	streamStart := time.Now()
	var thinkingStart time.Time
	var hasThinking bool

	var fullContent strings.Builder
	var fullThinking strings.Builder

	msgContents := convertMessages(messages)

	opts := []llms.CallOption{
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
			text := string(chunk)
			if text != "" {
				fullContent.WriteString(text)
				onChunk(text)
			}
			return nil
		}),
	}

	if thinkingEnabled {
		opts = append(opts,
			llms.WithStreamingReasoningFunc(func(_ context.Context, reasoningChunk, chunk []byte) error {
				if !hasThinking {
					thinkingStart = time.Now()
					hasThinking = true
				}
				rText := string(reasoningChunk)
				if rText != "" {
					fullThinking.WriteString(rText)
					if onThinking != nil {
						onThinking(rText)
					}
				}
				// content may come alongside reasoning
				if len(chunk) > 0 {
					text := string(chunk)
					if text != "" {
						fullContent.WriteString(text)
						onChunk(text)
					}
				}
				return nil
			}),
			llms.WithThinking(&llms.ThinkingConfig{
				Mode:           llms.ThinkingModeAuto,
				StreamThinking: true,
				ReturnThinking: true,
			}),
		)
	}

	_, err = llm.GenerateContent(ctx, msgContents, opts...)
	if err != nil {
		// 检查是否是 context.Canceled（用户取消）
		if strings.Contains(err.Error(), "context canceled") {
			onDone(fullContent.String(), 0, 0)
			return
		}
		onError("AI 调用失败: " + err.Error())
		return
	}

	var elapsedThinking float64
	if hasThinking {
		elapsedThinking = time.Since(thinkingStart).Seconds()
	}
	elapsedTotal := time.Since(streamStart).Seconds()
	onDone(fullContent.String(), elapsedThinking, elapsedTotal)
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

// testGenericConnection 通过创建 LLM 并执行简单调用来测试连通性
func testGenericConnection(cfg AIConfig) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	llm, err := createLLM(ctx, cfg)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	_, err = llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "test"),
	}, llms.WithMaxTokens(1))
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
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	LastMessage  string `json:"last_message"`
	MessageCount int    `json:"message_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// GetAISessions 获取所有会话，按 updated_at DESC 排序，附带最后一条消息摘要
func (a *AIService) GetAISessions() []AISessionSummary {
	var sessions []models.AISession
	a.db.Order("updated_at DESC").Find(&sessions)

	result := make([]AISessionSummary, 0, len(sessions))
	for _, s := range sessions {
		summary := AISessionSummary{
			ID:        s.ID,
			Title:     s.Title,
			CreatedAt: s.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02 15:04"),
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
