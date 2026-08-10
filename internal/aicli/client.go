package aicli

import (
	"context"
	"errors"
	"strings"
	"time"

	"jot/internal/aierrors"
)

// Client 统一 AI API 适配层
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
}

// NewClient 创建新的适配层客户端
func NewClient(cfg Config) *Client {
	return &Client{
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}
}

// Stream 流式调用 AI 接口，调用 OpenAI 兼容 API 流式接口
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

	err := c.openaiChatStream(ctx, messages, thinkingEnabled, wrappedCallbacks)

	if err != nil {
		if ctx.Err() != nil {
			// 用户取消，不报错
		} else if callbacks.OnError != nil {
			var aiErr *aierrors.AIErrorWrapper
			if errors.As(err, &aiErr) {
				callbacks.OnError(aiErr.Err.ToJSON())
			} else {
				callbacks.OnError("AI 调用失败: " + err.Error())
			}
		}
	}

	// 计算耗时
	var elapsedThinking float64
	if hasThinking {
		elapsedThinking = time.Since(thinkingStart).Seconds()
	}
	elapsedTotal := time.Since(streamStart).Seconds()

	if err == nil && callbacks.OnDone != nil {
		callbacks.OnDone(fullContent.String(), elapsedThinking, elapsedTotal)
	}
}

// Chat 非流式调用 AI 接口
func (c *Client) Chat(ctx context.Context, messages []Message, thinkingEnabled bool) (string, string, error) {
	return c.openaiChat(ctx, messages)
}

// Embed 批量生成文本向量，调用 OpenAI 兼容 embeddings 接口
// texts 为空时返回空切片且不调用外部 API；返回与 texts 一一对应的向量列表
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	return c.openaiEmbed(ctx, texts)
}

// EmbedWithProgress 按批次批量生成文本向量：每完成一批回调一次块级进度
// batchSize 为每批文本数（<=0 时整批一次调用）；cb(done, total) 中 total 为总文本数
// 返回与 texts 一一对应的向量列表；任一批失败即返回错误
func (c *Client) EmbedWithProgress(ctx context.Context, texts []string, batchSize int, cb func(done, total int)) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	if batchSize <= 0 {
		batchSize = len(texts)
	}

	result := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		var batchVec [][]float32
		var err error
		batchVec, err = c.openaiEmbed(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		result = append(result, batchVec...)

		if cb != nil {
			cb(end, len(texts))
		}
	}
	return result, nil
}
