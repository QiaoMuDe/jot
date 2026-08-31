package einocli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"jot/internal/aierrors"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// filterEmpty 过滤掉内容为空的消息，避免底层库 omitempty 导致 API 报 "content field is a required field"
func filterEmpty(messages []Message) []Message {
	filtered := make([]Message, 0, len(messages))
	for _, m := range messages {
		if m.Content != "" {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// toSchemaMessages 将 Message 转为 eino schema.Message
func toSchemaMessages(messages []Message) []*schema.Message {
	ret := make([]*schema.Message, 0, len(messages))
	for _, m := range messages {
		ret = append(ret, &schema.Message{
			Role:    roleFromString(m.Role),
			Content: m.Content,
		})
	}
	return ret
}

// roleFromString 将 role 字符串转为 eino 角色类型
func roleFromString(role string) schema.RoleType {
	switch role {
	case "user":
		return schema.User
	case "assistant":
		return schema.Assistant
	case "system":
		return schema.System
	case "tool":
		return schema.Tool
	default:
		return schema.User
	}
}

// Chat 非流式调用 AI 接口（eino Generate）
func (c *Client) Chat(ctx context.Context, messages []Message, thinkingEnabled bool) (string, string, error) {
	filtered := filterEmpty(messages)
	if len(filtered) == 0 {
		return "", "", errors.New("没有有效消息可发送")
	}

	cm, err := c.newChatModel(ctx)
	if err != nil {
		return "", "", err
	}

	out, err := cm.Generate(ctx, toSchemaMessages(filtered))
	if err != nil {
		if ae := aierrors.ClassifyError(err); ae != nil {
			return "", "", &aierrors.AIErrorWrapper{Err: ae}
		}
		return "", "", fmt.Errorf("非流式调用失败: %w", err)
	}
	if out == nil {
		return "", "", errors.New("API 返回空响应")
	}
	return out.Content, out.ReasoningContent, nil
}

// Stream 流式调用 AI 接口（eino Stream）
// thinkingEnabled 通过 WithExtraFields 传递 enable_thinking（Qwen3 / DeepSeek 等兼容端点）
func (c *Client) Stream(ctx context.Context, messages []Message, thinkingEnabled bool, callbacks StreamCallbacks) {
	streamStart := time.Now()
	var thinkingStart time.Time
	var hasThinking bool

	var fullContent strings.Builder
	var fullThinking strings.Builder

	// 过滤空消息并转为 eino 消息
	filtered := filterEmpty(messages)
	if len(filtered) == 0 {
		if callbacks.OnError != nil {
			callbacks.OnError("AI 调用失败: 没有有效消息可发送")
		}
		return
	}

	cm, err := c.newChatModel(ctx)
	if err != nil {
		if callbacks.OnError != nil {
			callbacks.OnError(classifyErrorString(err))
		}
		return
	}

	stream, err := cm.Stream(ctx, toSchemaMessages(filtered),
		openai.WithExtraFields(map[string]any{"enable_thinking": thinkingEnabled}))
	if err != nil {
		if callbacks.OnError != nil {
			callbacks.OnError(classifyErrorString(err))
		}
		return
	}
	defer stream.Close()

	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			if ctx.Err() != nil {
				// 用户主动取消，不报错
			} else if callbacks.OnError != nil {
				callbacks.OnError(classifyErrorString(recvErr))
			}
			return
		}
		if chunk == nil {
			continue
		}

		// 推送 reasoning_content（深度思考模型的思维链）
		if chunk.ReasoningContent != "" {
			if !hasThinking {
				thinkingStart = time.Now()
				hasThinking = true
			}
			fullThinking.WriteString(chunk.ReasoningContent)
			if callbacks.OnThinking != nil {
				callbacks.OnThinking(chunk.ReasoningContent)
			}
		}

		// 推送 content
		if chunk.Content != "" {
			fullContent.WriteString(chunk.Content)
			if callbacks.OnChunk != nil {
				callbacks.OnChunk(chunk.Content)
			}
		}
	}

	// 计算耗时
	var elapsedThinking float64
	if hasThinking {
		elapsedThinking = time.Since(thinkingStart).Seconds()
	}
	elapsedTotal := time.Since(streamStart).Seconds()

	if callbacks.OnDone != nil {
		callbacks.OnDone(fullContent.String(), fullThinking.String(), elapsedThinking, elapsedTotal)
	}
}

// newChatModel 创建 eino OpenAI ChatModel
func (c *Client) newChatModel(ctx context.Context) (*openai.ChatModel, error) {
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  c.APIKey,
		BaseURL: c.BaseURL,
		Model:   c.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ChatModel 失败: %w", err)
	}
	return cm, nil
}

// classifyErrorString 将错误分类为中文提示 JSON；无法分类时返回通用错误文本
func classifyErrorString(err error) string {
	if ae := aierrors.ClassifyError(err); ae != nil {
		return ae.ToJSON()
	}
	return "AI 调用失败: " + err.Error()
}
