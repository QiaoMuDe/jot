package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jot/internal/models"

	"gorm.io/gorm"
)

// Message 表示 AI 对话中的一条消息
type Message struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
}

// AIConfig 表示 AI 服务配置
type AIConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
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
	return AIConfig{
		BaseURL: svc.Get("ai_base_url"),
		APIKey:  svc.Get("ai_api_key"),
		Model:   svc.Get("ai_model"),
	}
}

// SaveConfig 保存 AI 配置到 SettingService
func (a *AIService) SaveConfig(cfg AIConfig) error {
	svc := NewSettingService(a.db)
	if err := svc.Set("ai_base_url", cfg.BaseURL); err != nil {
		return err
	}
	if err := svc.Set("ai_api_key", cfg.APIKey); err != nil {
		return err
	}
	return svc.Set("ai_model", cfg.Model)
}

// thinkingParam 思考模式参数
type thinkingParam struct {
	Type string `json:"type"`
}

// chatRequest 表示 Chat Completion 请求体
type chatRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Thinking *thinkingParam `json:"thinking,omitempty"`
}

// chatResponse 表示 Chat Completion 响应体
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CallAI 调用 OpenAI 兼容的 Chat Completion API
func (a *AIService) CallAI(messages []Message) (string, error) {
	cfg := a.GetConfig()

	body := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	url := cfg.BaseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI 调用失败: HTTP %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 调用失败: 响应中没有 choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// CallAIStream 流式调用 OpenAI 兼容的 Chat Completion API
// 通过 onChunk/onThinking/onDone/onError 回调逐块推送内容
// onThinking 用于深度思考模型的 reasoning_content 字段
func (a *AIService) CallAIStream(ctx context.Context, messages []Message, thinkingEnabled bool, onChunk func(string), onThinking func(string), onDone func(string), onError func(string)) {
	cfg := a.GetConfig()

	type streamChoice struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		Index int `json:"index"`
	}
	type streamResponse struct {
		Choices []streamChoice `json:"choices"`
	}

	body := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   true,
	}
	if thinkingEnabled {
		body.Thinking = &thinkingParam{Type: "enabled"}
	} else {
		body.Thinking = &thinkingParam{Type: "disabled"}
	}

	var fullContent strings.Builder
	var fullThinking strings.Builder

	reqBody, err := json.Marshal(body)
	if err != nil {
		onError("请求序列化失败: " + err.Error())
		return
	}

	url := cfg.BaseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		onError("请求创建失败: " + err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		onError("连接失败: " + err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		onError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			onDone(fullContent.String())
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			onError("读取流失败: " + err.Error())
			return
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var sr streamResponse
		if err := json.Unmarshal([]byte(data), &sr); err != nil {
			continue
		}

		if len(sr.Choices) > 0 {
			chunk := sr.Choices[0].Delta.Content
			if chunk != "" {
				fullContent.WriteString(chunk)
				onChunk(chunk)
			}

			thinking := sr.Choices[0].Delta.ReasoningContent
			if thinking != "" {
				fullThinking.WriteString(thinking)
				if onThinking != nil {
					onThinking(thinking)
				}
			}
		}
	}

	onDone(fullContent.String())
}

// TestBaseURL 测试 Base URL 连通性（带 API Key 认证，5 秒超时）
func (a *AIService) TestBaseURL(baseURL, apiKey string) (bool, error) {
	url := baseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

// modelsResponse 表示 /models 接口的响应体
type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// FetchModels 获取可用模型列表
func (a *AIService) FetchModels(baseURL, apiKey string) ([]string, error) {
	url := baseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

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
		result[i] = Message{Role: m.Role, Content: m.Content, ReasoningContent: m.ReasoningContent}
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
