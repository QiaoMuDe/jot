package aicli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	api "github.com/ollama/ollama/api"
)

// ollamaChatStream 调用 Ollama 原生 API 的流式接口 (/api/chat)
func (c *Client) ollamaChatStream(ctx context.Context, messages []Message, thinkingEnabled bool, callbacks StreamCallbacks) error {
	ollamaClient, err := c.newOllamaClient()
	if err != nil {
		return err
	}

	req := &api.ChatRequest{
		Model:    c.Model,
		Messages: convertToOllamaMessages(messages),
	}

	// 深度思考：显式设置 true/false，关闭时也从根源不让模型输出 thinking
	req.Think = &api.ThinkValue{Value: thinkingEnabled}

	// 流式回调
	err = ollamaClient.Chat(ctx, req, func(resp api.ChatResponse) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg := resp.Message

		// 推送 thinking
		if msg.Thinking != "" && callbacks.OnThinking != nil {
			callbacks.OnThinking(msg.Thinking)
		}

		// 推送 content
		if msg.Content != "" && callbacks.OnChunk != nil {
			callbacks.OnChunk(msg.Content)
		}

		return nil
	})

	if err != nil {
		if ae := ClassifyError(err); ae != nil {
			return &AIErrorWrapper{Err: ae}
		}
		return err
	}

	return nil
}

// ollamaChat 调用 Ollama 原生 API 的非流式接口
func (c *Client) ollamaChat(ctx context.Context, messages []Message, thinkingEnabled bool) (string, string, error) {
	ollamaClient, err := c.newOllamaClient()
	if err != nil {
		return "", "", err
	}

	req := &api.ChatRequest{
		Model:    c.Model,
		Messages: convertToOllamaMessages(messages),
	}

	// 深度思考：显式设置 true/false
	req.Think = &api.ThinkValue{Value: thinkingEnabled}

	var fullContent, fullThinking string

	err = ollamaClient.Chat(ctx, req, func(resp api.ChatResponse) error {
		msg := resp.Message
		if msg.Thinking != "" {
			fullThinking += msg.Thinking
		}
		if msg.Content != "" {
			fullContent += msg.Content
		}
		return nil
	})

	if err != nil {
		if ae := ClassifyError(err); ae != nil {
			return "", "", &AIErrorWrapper{Err: ae}
		}
		return "", "", fmt.Errorf("ollama 调用失败: %w", err)
	}

	return fullContent, fullThinking, nil
}

// newOllamaClient 创建 Ollama 客户端
func (c *Client) newOllamaClient() (*api.Client, error) {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析 Ollama URL 失败: %w", err)
	}

	return api.NewClient(parsedURL, http.DefaultClient), nil
}

// convertToOllamaMessages 将 aicli.Message 转换为 ollama api.Message
func convertToOllamaMessages(msgs []Message) []api.Message {
	result := make([]api.Message, len(msgs))
	for i, m := range msgs {
		result[i] = api.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}
