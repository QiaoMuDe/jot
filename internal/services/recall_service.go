package services

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// RecallCard 单条召回卡片，用于前端展示
type RecallCard struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`   // 笔记内容（可能被截断）
	FileExt   string `json:"file_ext"`  // 文件后缀，如 .md / .txt
	Truncated bool   `json:"truncated"` // 内容是否被截断
}

// CardRecallResult 卡片召回结果
type CardRecallResult struct {
	FormattedText string       // 注入 system message 的格式化文本
	Cards         []RecallCard // 结构化卡片列表，用于前端展示
}

// tokenize2Gram 对输入文本做 2-gram 分词
// - 中文（CJK 字符）：每两个连续字符作为一个 gram
// - 英文/数字：按空格和标点切分成单词
// - 去重
func tokenize2Gram(text string) []string {
	runes := []rune(text)
	seen := make(map[string]struct{})
	var grams []string

	// 按连续的类型（中文/非中文）分段处理
	i := 0
	for i < len(runes) {
		if isCJK(runes[i]) {
			// 中文 2-gram
			j := i
			for j < len(runes) && isCJK(runes[j]) {
				j++
			}
			// 生成中文 2-gram
			chunk := runes[i:j]
			for k := 0; k < len(chunk)-1; k++ {
				gram := string(chunk[k]) + string(chunk[k+1])
				if _, ok := seen[gram]; !ok {
					seen[gram] = struct{}{}
					grams = append(grams, gram)
				}
			}
			// 如果只有一个中文字，也作为关键词
			if len(chunk) == 1 {
				gram := string(chunk[0])
				if _, ok := seen[gram]; !ok {
					seen[gram] = struct{}{}
					grams = append(grams, gram)
				}
			}
			i = j
		} else {
			// 非中文：按空格和标点切分成单词
			j := i
			for j < len(runes) && !isCJK(runes[j]) {
				j++
			}
			words := splitWords(string(runes[i:j]))
			for _, w := range words {
				if w == "" {
					continue
				}
				if _, ok := seen[w]; !ok {
					seen[w] = struct{}{}
					grams = append(grams, w)
				}
			}
			i = j
		}
	}

	return grams
}

// isCJK 判断是否为中日韩文字
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r)
}

// splitWords 按非字母数字字符切分英文/数字
func splitWords(text string) []string {
	var words []string
	var current []rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// extractContext 在笔记内容中定位第一个匹配的关键词，截取上下文片段。
// contextChars 控制截取关键词前后各多少字符。
// 如果找不到任何关键词，返回空串。
func extractContext(content string, keywords []string, contextChars int) string {
	runes := []rune(content)
	contentLen := len(runes)
	if contentLen == 0 || len(keywords) == 0 {
		return ""
	}

	for _, kw := range keywords {
		byteIdx := strings.Index(content, kw)
		if byteIdx < 0 {
			continue
		}
		// 将字节偏移转为字符偏移
		pos := len([]rune(content[:byteIdx]))
		kwLen := len([]rune(kw))

		start := pos - contextChars
		if start < 0 {
			start = 0
		}
		end := pos + kwLen + contextChars
		if end > contentLen {
			end = contentLen
		}

		var b strings.Builder
		if start > 0 {
			b.WriteString("...(内容已截断)")
		}
		b.WriteString(string(runes[start:end]))
		if end < contentLen {
			b.WriteString("...(内容已截断)")
		}
		return b.String()
	}

	return ""
}

// CardRecallSearch 执行卡片召回
// 对 query 做 2-gram 分词 → 多关键词 OR 搜索笔记 → 格式化上下文 + 返回结构化卡片数据
// maxChars 控制单条笔记最大字符数（字节），超过时截取关键词上下文；<=0 表示不截断
func CardRecallSearch(ctx context.Context, query string, limit int, maxChars int, noteService *NoteService) *CardRecallResult {
	if query == "" || limit <= 0 {
		return nil
	}

	keywords := tokenize2Gram(query)
	if len(keywords) == 0 {
		return nil
	}

	// 搜索笔记（全量内容）
	notes, err := noteService.SearchFull(keywords, limit)
	if err != nil || len(notes) == 0 {
		return nil
	}

	// 构建格式化上下文文本
	var b strings.Builder
	b.WriteString("以下是用户笔记库中与问题相关的笔记，请参考这些笔记内容回答用户的问题：\n\n")

	const contextChars = 200
	cards := make([]RecallCard, 0, len(notes))
	for _, note := range notes {
		content := note.Content
		truncated := false

		if maxChars > 0 && len(content) > maxChars {
			snippet := extractContext(content, keywords, contextChars)
			if snippet != "" {
				content = snippet
				truncated = true
			} else {
				// 回退：从头截取 maxChars 字节
				content = content[:maxChars] + "\n...(内容已截断)"
				truncated = true
			}
		}

		fmt.Fprintf(&b, "--- 📄 《%s》 ---\n%s\n\n", note.Title, content)
		cards = append(cards, RecallCard{
			ID:        note.ID,
			Title:     note.Title,
			Content:   content,
			FileExt:   note.FileExt,
			Truncated: truncated,
		})
	}

	b.WriteString("请基于以上笔记内容回答用户的问题。如果笔记内容不足以回答，请如实说明。")

	return &CardRecallResult{
		FormattedText: b.String(),
		Cards:         cards,
	}
}
