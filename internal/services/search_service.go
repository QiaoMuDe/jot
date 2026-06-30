package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hekmon/tavily"
)

// SearchSource 单条搜索来源，用于前端展示
type SearchSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// SearchWebResult 搜索返回结果，包含格式化文本和结构化来源列表
type SearchWebResult struct {
	FormattedText string         // 格式化后的搜索结果文本，注入 system message
	Sources       []SearchSource // 结构化来源列表，用于前端展示
}

// SearchWeb 使用 Tavily 搜索互联网，返回结构化的搜索结果
// 如果 apiKey 为空或搜索失败，返回 nil（不阻塞调用方）
// ctx 用于支持超时和取消（与 CallAIStream 共用 cancel，停止按钮可中断搜索）
func SearchWeb(ctx context.Context, query string, apiKey string) *SearchWebResult {
	if apiKey == "" {
		return nil
	}

	// 用传入的 ctx 派生超时子上下文（5s 超时，但 ctx 取消时立即传播）
	searchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := tavily.NewClient(apiKey, tavily.APIKeyTypeDev, http.DefaultClient)
	searchQuery := tavily.SearchQuery{
		Query:       query,
		SearchDepth: tavily.SearchQueryDepthAdvanced,
		MaxResults:  5,
	}
	answer, err := client.Search(searchCtx, searchQuery)
	if err != nil {
		return nil
	}

	if len(answer.Results) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("以下是从互联网搜索到的相关信息（请基于这些信息回答用户的问题，并在回答时标注引用来源）：\n\n")

	sources := make([]SearchSource, 0, len(answer.Results))

	for i, r := range answer.Results {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, r.Title)
		fmt.Fprintf(&b, "    来源: %s\n", r.URL.String())
		content := strings.TrimSpace(r.Content)
		if content != "" {
			fmt.Fprintf(&b, "    内容: %s\n", content)
		}
		b.WriteString("\n")

		sources = append(sources, SearchSource{
			Title:   r.Title,
			URL:     r.URL.String(),
			Content: content,
		})
	}

	return &SearchWebResult{
		FormattedText: b.String(),
		Sources:       sources,
	}
}
