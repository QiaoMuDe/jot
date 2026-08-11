// Package embed 封装 Ollama 本地 embedding 服务的调用。
package embed

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	api "github.com/ollama/ollama/api"
)

// batchSize 每次调用 Ollama embedding 接口的最大文本条数。
const batchSize = 32

// BatchProgress 是每批文本向量化完成后的进度回调。
type BatchProgress struct {
	Done       int           // 已处理总块数
	Total      int           // 总块数
	BatchSize  int           // 本批块数
	BatchCost  time.Duration // 本批耗时
	AvgPerItem time.Duration // 本批平均每块耗时
}

// Client 是 Ollama embedding 客户端。
type Client struct {
	baseURL string // Ollama 服务地址
	model   string // embedding 模型名
	// OnBatch 每完成一批文本的向量化后回调；可为 nil。
	OnBatch func(p BatchProgress)
}

// NewClient 创建 Ollama embedding 客户端；baseURL 为空时默认本地地址。
func NewClient(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Client{baseURL: baseURL, model: model}
}

// Embed 批量计算文本 embedding：texts 按每批 32 条拆分调用 Ollama。
// 返回与 texts 一一对应的向量列表；每批完成后触发 OnBatch 回调（若设置），包含该批耗时与单块平均耗时。
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("Ollama 地址解析失败 (%s): %w", c.baseURL, err)
	}
	// 设置较长超时，本地模型首次加载可能较慢
	client := api.NewClient(base, &http.Client{Timeout: 5 * time.Minute})

	var out [][]float32
	done := 0
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		req := &api.EmbedRequest{
			Model: c.model,
			Input: texts[i:end],
		}
		batchStart := time.Now()
		resp, err := client.Embed(ctx, req)
		batchCost := time.Since(batchStart)
		if err != nil {
			return nil, fmt.Errorf("Ollama embedding 调用失败 (baseURL=%s, model=%s): %w", c.baseURL, c.model, err)
		}
		out = append(out, resp.Embeddings...)
		batchSize := end - i
		done += batchSize
		if c.OnBatch != nil {
			var avg time.Duration
			if batchSize > 0 {
				avg = batchCost / time.Duration(batchSize)
			}
			c.OnBatch(BatchProgress{
				Done:       done,
				Total:      len(texts),
				BatchSize:  batchSize,
				BatchCost:  batchCost,
				AvgPerItem: avg,
			})
		}
	}
	return out, nil
}

// EmbedOne 便捷方法：计算单条文本的 embedding 向量。
func (c *Client) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("Ollama 返回空的 embedding 结果 (baseURL=%s, model=%s)", c.baseURL, c.model)
	}
	return vecs[0], nil
}
