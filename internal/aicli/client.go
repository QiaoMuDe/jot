package aicli

import (
	"context"
	"strings"
	"time"
)

// Client 统一 AI API 适配层
type Client struct {
	Provider string // "openai" 或 "ollama"
	BaseURL  string
	APIKey   string
	Model    string
}

// NewClient 创建新的适配层客户端
func NewClient(cfg Config) *Client {
	return &Client{
		Provider: cfg.Provider,
		BaseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	}
}

// Stream 流式调用 AI 接口，自动根据 Provider 选择后端
func (c *Client) Stream(ctx context.Context, messages []Message, thinkingEnabled bool, callbacks StreamCallbacks) {
	streamStart := time.Now()
	var thinkingStart time.Time
	var hasThinking bool

	var fullContent strings.Builder
	var fullThinking strings.Builder

	// 包装回调以追踪全量内容和耗时
	wrappedCallbacks := StreamCallbacks{
		OnChunk: func(text string) {
			fullContent.WriteString(text)
			if callbacks.OnChunk != nil {
				callbacks.OnChunk(text)
			}
		},
		OnThinking: func(text string) {
			if !hasThinking {
				thinkingStart = time.Now()
				hasThinking = true
			}
			fullThinking.WriteString(text)
			if callbacks.OnThinking != nil {
				callbacks.OnThinking(text)
			}
		},
		OnDone:  callbacks.OnDone,
		OnError: callbacks.OnError,
	}

	var err error
	switch c.Provider {
	case "ollama":
		err = c.ollamaChatStream(ctx, messages, thinkingEnabled, wrappedCallbacks)
	default:
		err = c.openaiChatStream(ctx, messages, thinkingEnabled, wrappedCallbacks)
	}

	if err != nil {
		if ctx.Err() != nil {
			// 用户取消，不报错
		} else if callbacks.OnError != nil {
			callbacks.OnError("AI 调用失败: " + err.Error())
		}
	}

	// 计算耗时
	var elapsedThinking float64
	if hasThinking {
		elapsedThinking = time.Since(thinkingStart).Seconds()
	}
	elapsedTotal := time.Since(streamStart).Seconds()

	if callbacks.OnDone != nil {
		callbacks.OnDone(fullContent.String(), elapsedThinking, elapsedTotal)
	}
}

// Chat 非流式调用 AI 接口
func (c *Client) Chat(ctx context.Context, messages []Message, thinkingEnabled bool) (string, string, error) {
	switch c.Provider {
	case "ollama":
		return c.ollamaChat(ctx, messages, thinkingEnabled)
	default:
		return c.openaiChat(ctx, messages)
	}
}
