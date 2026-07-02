package aicli

import (
	"context"
	"errors"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// openaiChatStream 调用 OpenAI 兼容 API 的流式接口
func (c *Client) openaiChatStream(ctx context.Context, messages []Message, thinkingEnabled bool, callbacks StreamCallbacks) error {
	client := c.newOpenAIClient()

	// 过滤掉内容为空的消息，避免 go-openai omitempty 导致 Api 报 "content field is a required field"
	filtered := make([]Message, 0, len(messages))
	for _, m := range messages {
		if m.Content != "" {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return errors.New("没有有效消息可发送")
	}

	req := openai.ChatCompletionRequest{
		Model:  c.Model,
		Stream: true,
	}
	req.Messages = make([]openai.ChatCompletionMessage, len(filtered))
	for i, m := range filtered {
		req.Messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// 对支持 enable_thinking 的模型（如 Qwen3 / Ollama OpenAI 兼容接口）传递思考参数
	// 关闭时也显式设为 false，从根源上不让后端请求思考内容
	req.ChatTemplateKwargs = map[string]any{
		"enable_thinking": thinkingEnabled,
	}

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		if ae := ClassifyError(err); ae != nil {
			return &AIErrorWrapper{Err: ae}
		}
		return fmt.Errorf("创建流失败: %w", err)
	}
	defer func() { _ = stream.Close() }()

	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return ctx.Err()
			}
			if errors.Is(err, io.EOF) {
				// 流正常结束
				return nil
			}
			// 其他错误：尝试分类
			if ae := ClassifyError(err); ae != nil {
				return &AIErrorWrapper{Err: ae}
			}
			return nil
		}

		if len(resp.Choices) == 0 {
			continue
		}

		delta := resp.Choices[0].Delta

		// 推送 reasoning_content（来自 DeepSeek 等推理模型的思维链）
		if delta.ReasoningContent != "" && callbacks.OnThinking != nil {
			callbacks.OnThinking(delta.ReasoningContent)
		}

		// 推送 content
		if delta.Content != "" && callbacks.OnChunk != nil {
			callbacks.OnChunk(delta.Content)
		}
	}
}

// openaiChat 调用 OpenAI 兼容 API 的非流式接口
func (c *Client) openaiChat(ctx context.Context, messages []Message) (string, string, error) {
	client := c.newOpenAIClient()

	// 过滤掉内容为空的消息
	filtered := make([]Message, 0, len(messages))
	for _, m := range messages {
		if m.Content != "" {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return "", "", errors.New("没有有效消息可发送")
	}

	req := openai.ChatCompletionRequest{
		Model: c.Model,
	}
	req.Messages = make([]openai.ChatCompletionMessage, len(filtered))
	for i, m := range filtered {
		req.Messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		if ae := ClassifyError(err); ae != nil {
			return "", "", &AIErrorWrapper{Err: ae}
		}
		return "", "", fmt.Errorf("非流式调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", "", errors.New("API 返回空 choices")
	}

	content := resp.Choices[0].Message.Content
	reasoning := resp.Choices[0].Message.ReasoningContent

	return content, reasoning, nil
}

// newOpenAIClient 创建 OpenAI 客户端
func (c *Client) newOpenAIClient() *openai.Client {
	cfg := openai.DefaultConfig(c.APIKey)
	if c.BaseURL != "" {
		cfg.BaseURL = c.BaseURL
	}
	return openai.NewClientWithConfig(cfg)
}
