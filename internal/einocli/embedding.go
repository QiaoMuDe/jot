package einocli

import (
	"context"
	"fmt"
	"net/http"

	"jot/internal/aierrors"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
)

// Embed 批量生成文本向量（eino EmbeddingClient），返回与 texts 一一对应的向量列表
// texts 为空时返回空切片且不调用外部 API
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	ec, err := c.newEmbeddingClient(ctx)
	if err != nil {
		return nil, err
	}

	emb64, err := ec.EmbedStrings(ctx, texts)
	if err != nil {
		if ae := aierrors.ClassifyError(err); ae != nil {
			return nil, &aierrors.AIErrorWrapper{Err: ae}
		}
		return nil, fmt.Errorf("embedding 调用失败: %w", err)
	}

	// float64 → float32（向量库存储为 float32 小端 BLOB）
	result := make([][]float32, len(emb64))
	for i, vec64 := range emb64 {
		vec32 := make([]float32, len(vec64))
		for j, v := range vec64 {
			vec32[j] = float32(v)
		}
		result[i] = vec32
	}

	// 校验返回数量与请求一致
	if len(result) != len(texts) {
		return nil, fmt.Errorf("embedding 返回数量不完整: 期望 %d 条，实际 %d 条", len(texts), len(result))
	}
	return result, nil
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

		batchVec, err := c.Embed(ctx, texts[start:end])
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

// newEmbeddingClient 创建 eino OpenAI EmbeddingClient
// 注意：必须显式设置 HTTPClient。eino-ext 的 NewEmbeddingClient 中
// clientConf.HTTPClient = config.HTTPClient 将 nil 的 *http.Client 赋给 HTTPDoer 接口后
// 成为 typed-nil，== nil 判断失效，不会回退 http.DefaultClient，导致 go-openai 发送请求时
// 对 nil 调用 Do() 触发 panic（invalid memory address or nil pointer dereference）。
func (c *Client) newEmbeddingClient(ctx context.Context) (*openai.EmbeddingClient, error) {
	ec, err := openai.NewEmbeddingClient(ctx, &openai.EmbeddingConfig{
		APIKey:     c.APIKey,
		BaseURL:    c.BaseURL,
		Model:      c.Model,
		HTTPClient: http.DefaultClient,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 EmbeddingClient 失败: %w", err)
	}
	return ec, nil
}
