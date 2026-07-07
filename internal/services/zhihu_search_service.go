package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee.com/MM-Q/zhihu-go"
)

// SearchZhihuContent 使用知乎搜索 API 搜索知乎内容
// 返回格式与 SearchWebResult 一致，Sources 中 SourceLabel 为 "zhihu_search"
// 失败时返回 error（不再静默跳过）
func SearchZhihuContent(ctx context.Context, query string, accessSecret string, maxResults int) (*SearchWebResult, error) {
	if accessSecret == "" {
		return nil, fmt.Errorf("知乎 Access Secret 未配置")
	}

	searchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := zhihu.NewClient(accessSecret)
	searchData, err := client.SearchZhihu(searchCtx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("知乎搜索失败: %w", err)
	}

	items := searchData.Items
	if len(items) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("以下是从知乎搜索到的相关信息（请基于这些信息回答用户的问题，并在回答时标注引用来源）：\n\n")

	sources := make([]SearchSource, 0, len(items))
	for i, item := range items {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, item.Title)
		fmt.Fprintf(&b, "    来源: %s\n", item.Url)
		content := strings.TrimSpace(item.ContentText)
		if content != "" {
			fmt.Fprintf(&b, "    内容: %s\n", content)
		}
		b.WriteString("\n")

		sources = append(sources, SearchSource{
			Title:       item.Title,
			URL:         item.Url,
			Content:     content,
			SourceLabel: "zhihu_search",
		})
	}

	return &SearchWebResult{
		FormattedText: b.String(),
		Sources:       sources,
	}, nil
}

// SearchGlobalContent 使用知乎全网搜索 API 搜索互联网内容
// 返回格式与 SearchWebResult 一致，Sources 中 SourceLabel 为 "zhihu_global"
// 失败时返回 error（不再静默跳过）
func SearchGlobalContent(ctx context.Context, query string, accessSecret string, maxResults int) (*SearchWebResult, error) {
	if accessSecret == "" {
		return nil, fmt.Errorf("知乎 Access Secret 未配置")
	}

	searchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := zhihu.NewClient(accessSecret)
	globalData, err := client.SearchGlobal(searchCtx, query, maxResults, "", "")
	if err != nil {
		return nil, fmt.Errorf("知乎全网搜索失败: %w", err)
	}

	items := globalData.Items
	if len(items) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("以下是从全网搜索到的相关信息（请基于这些信息回答用户的问题，并在回答时标注引用来源）：\n\n")

	sources := make([]SearchSource, 0, len(items))
	for i, item := range items {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, item.Title)
		fmt.Fprintf(&b, "    来源: %s\n", item.Url)
		content := strings.TrimSpace(item.ContentText)
		if content != "" {
			fmt.Fprintf(&b, "    内容: %s\n", content)
		}
		b.WriteString("\n")

		sources = append(sources, SearchSource{
			Title:       item.Title,
			URL:         item.Url,
			Content:     content,
			SourceLabel: "zhihu_global",
		})
	}

	return &SearchWebResult{
		FormattedText: b.String(),
		Sources:       sources,
	}, nil
}
