package services

import (
	"strings"
)

const searchQueryRefinePrompt = `你是一个搜索查询优化专家。你的任务是将用户的输入改写为更适合搜索引擎检索的查询语句。

规则：
- 理解用户真实意图，将口语化表达改写为简洁的搜索查询
- 保留核心实体词（人名、产品名、版本号、术语等），不要遗漏
- 如果用户描述比较模糊，适当补充上下文关键词使查询更精准
- 如果包含多个独立话题，用空格分隔形成一条综合查询
- 只输出优化后的查询语句，不要任何解释、前缀、引号或标点
- 使用与用户输入相同的语言

示例：
用户：帮我查一下最近 Go 1.22 有什么新特性
输出：Go 1.22 新特性 发布

用户：查查今天的天气和明天的股票行情
输出：今天 天气 明天 股市 行情

用户：那个新出的 SSR 框架怎么样，和 Next.js 比呢
输出：新 SSR 框架 Next.js 对比 2024

用户：帮我写一篇关于人工智能的文章
输出：人工智能 发展趋势 文章`

// RefineSearchQuery 调用 AI 模型将用户输入精炼为搜索引擎友好的关键词。
// 返回精炼后的 query 字符串。如果精炼失败（空输入、空结果或模型调用错误），
// 返回 error，由调用方决定是否终止流程。
func RefineSearchQuery(query string, aiService *AIService) (string, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return "", nil
	}

	messages := []Message{
		{Role: "system", Content: searchQueryRefinePrompt},
		{Role: "user", Content: trimmed},
	}

	result, err := aiService.CallAI(messages)
	if err != nil {
		return "", err
	}

	result = strings.TrimSpace(result)
	if result == "" {
		return "", nil
	}

	return result, nil
}
