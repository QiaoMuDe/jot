package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Message 表示 AI 对话中的一条消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

// chatRequest 表示 Chat Completion 请求体
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
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
// 通过 onChunk/onDone/onError 回调逐块推送内容
func (a *AIService) CallAIStream(messages []Message, onChunk func(string), onDone func(string), onError func(string)) {
	cfg := a.GetConfig()

	type streamChoice struct {
		Delta struct {
			Content string `json:"content"`
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

	var fullContent strings.Builder

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
