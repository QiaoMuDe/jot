// Package llm 封装 OpenAI 兼容接口的互联网模型对话客户端。
package llm

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"vec-poc/internal/store"
)

// Client 是 OpenAI 兼容接口的对话客户端。
type Client struct {
	baseURL string // OpenAI 兼容服务地址
	apiKey  string // API Key
	model   string // 模型名
}

// NewClient 创建对话客户端。
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, model: model}
}

// Chat 以 system + user 两条消息调用模型，返回回答文本。
func (c *Client) Chat(ctx context.Context, systemPrompt string, userQuestion string) (string, error) {
	config := openai.DefaultConfig(c.apiKey)
	if c.baseURL != "" {
		config.BaseURL = c.baseURL
	}
	client := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: userQuestion},
	}
	if systemPrompt != "" {
		// system 提示放在最前
		messages = append([]openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		}, messages...)
	}

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败 (baseURL=%s, model=%s): %w", c.baseURL, c.model, err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM 未返回任何回答 (baseURL=%s, model=%s)", c.baseURL, c.model)
	}
	return resp.Choices[0].Message.Content, nil
}

// BuildRecallPrompt 将召回片段组装为 system 提示词，指导模型基于片段回答。
func BuildRecallPrompt(hits []store.SearchHit) string {
	var sb strings.Builder
	sb.WriteString("以下是知识库中与问题相关的片段（按相关度排序），请优先基于这些片段回答问题；若片段不足以回答，请如实说明。\n\n")
	for _, h := range hits {
		fmt.Fprintf(&sb, "--- 来源: %s 块%d ---\n%s\n", h.DocName, h.ChunkIndex, h.Text)
	}
	return sb.String()
}
