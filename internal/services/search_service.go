package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hekmon/tavily"
)

// SearchWeb 使用 Tavily 搜索互联网，返回格式化的搜索结果文本
// 如果 apiKey 为空或搜索失败，返回空字符串（不阻塞调用方）
// ctx 用于支持超时和取消（与 CallAIStream 共用 cancel，停止按钮可中断搜索）
func SearchWeb(ctx context.Context, query string, apiKey string) string {
	if apiKey == "" {
		return ""
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
		return ""
	}

	if len(answer.Results) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("以下是从互联网搜索到的相关信息（请基于这些信息回答用户的问题，并在回答时标注引用来源）：\n\n")

	for i, r := range answer.Results {
		b.WriteString(fmt.Sprintf("[%d] %s\n", i+1, r.Title))
		b.WriteString(fmt.Sprintf("    来源: %s\n", r.URL.String()))
		content := strings.TrimSpace(r.Content)
		if content != "" {
			b.WriteString(fmt.Sprintf("    内容: %s\n", content))
		}
		b.WriteString("\n")
	}

	return b.String()
}
