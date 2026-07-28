package services

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"gitee.com/MM-Q/fastlog"
)

// 停用词表，过滤 2-gram 中的高频无意义碎片
var stopWords = map[rune]struct{}{
	// 高频单字停用词
	'的': {}, '了': {}, '是': {}, '在': {}, '有': {}, '我': {}, '他': {}, '她': {},
	'它': {}, '们': {}, '这': {}, '那': {}, '什': {}, '么': {}, '怎': {}, '哪': {},
	'你': {}, '之': {}, '于': {}, '其': {}, '着': {}, '过': {},
	'里': {}, '为': {}, '因': {}, '所': {}, '以': {}, '但': {}, '如': {}, '果': {},
	'虽': {}, '然': {}, '而': {}, '且': {}, '或': {}, '与': {}, '和': {}, '同': {},
	'及': {}, '又': {}, '也': {}, '对': {}, '就': {}, '被': {}, '把': {}, '让': {},
	'向': {}, '往': {}, '从': {}, '到': {}, '去': {}, '能': {}, '会': {}, '要': {},
	'可': {}, '没': {}, '不': {}, '很': {}, '太': {}, '更': {}, '最': {}, '都': {},
	'只': {}, '还': {}, '再': {}, '才': {}, '刚': {}, '已': {}, '正': {}, '将': {},
	'该': {}, '应': {}, '需': {}, '必': {}, '须': {}, '够': {}, '出': {}, '入': {},
	'上': {}, '下': {}, '大': {}, '小': {}, '多': {}, '少': {}, '来': {}, '做': {},
	'用': {}, '问': {}, '说': {}, '看': {}, '想': {}, '知': {}, '道': {}, '给': {},
	'跟': {}, '比': {}, '次': {}, '个': {}, '种': {}, '些': {}, '点': {}, '等': {},
	'第': {}, '每': {}, '各': {}, '几': {}, '两': {}, '百': {}, '千': {}, '万': {},
	'亿': {}, '哦': {}, '啊': {}, '嗯': {}, '呢': {}, '吧': {}, '吗': {}, '呀': {},
	'嘛': {}, '哈': {}, '哇': {}, '呵': {}, '嘿': {}, '喔': {},
}

// isStopWord 判断 rune 是否为停用词
func isStopWord(r rune) bool {
	_, ok := stopWords[r]
	return ok
}

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
// - 中文（CJK 字符）：每两个连续字符作为一个 gram，过滤包含停用词的无效 gram
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
			// 生成中文 2-gram，过滤包含停用词的噪音
			chunk := runes[i:j]
			for k := 0; k < len(chunk)-1; k++ {
				// 如果 gram 中任意字符是停用词则跳过（如"有什"、"什么"、"么新"）
				if isStopWord(chunk[k]) || isStopWord(chunk[k+1]) {
					continue
				}
				gram := string(chunk[k]) + string(chunk[k+1])
				if _, ok := seen[gram]; !ok {
					seen[gram] = struct{}{}
					grams = append(grams, gram)
				}
			}
			// 如果只有一个中文字且不是停用词，也作为关键词
			if len(chunk) == 1 && !isStopWord(chunk[0]) {
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

// TruncateRecallCardsPreview 截断召回卡片列表的 Content 字段用于前端预览
// 对每条 card 的 Content 按 maxLen 截断（rune 安全），不修改原切片，返回新切片
func TruncateRecallCardsPreview(cards []RecallCard, maxLen int) []RecallCard {
	if maxLen <= 0 {
		return cards
	}
	result := make([]RecallCard, len(cards))
	for i, card := range cards {
		runes := []rune(card.Content)
		if len(runes) > maxLen {
			card.Content = string(runes[:maxLen])
		}
		result[i] = card
	}
	return result
}

// TruncateSearchSourcesPreview 截断搜索来源列表的 Content 字段用于前端预览
// 对每条 source 的 Content 按 maxLen 截断（rune 安全），不修改原切片，返回新切片
func TruncateSearchSourcesPreview(sources []SearchSource, maxLen int) []SearchSource {
	if maxLen <= 0 {
		return sources
	}
	result := make([]SearchSource, len(sources))
	for i, src := range sources {
		runes := []rune(src.Content)
		if len(runes) > maxLen {
			src.Content = string(runes[:maxLen])
		}
		result[i] = src
	}
	return result
}

// CardRecallSearch 执行卡片召回
// 对 query 做 2-gram 分词 → 多关键词 OR 搜索笔记 → 格式化上下文 + 返回结构化卡片数据
func CardRecallSearch(ctx context.Context, query string, limit int, noteService *NoteService) *CardRecallResult {
	if query == "" || limit <= 0 {
		return nil
	}

	keywords := tokenize2Gram(query)
	if len(keywords) == 0 {
		return nil
	}

	// 日志记录分词结果
	noteService.logger.Infow("CardRecallSearch 分词结果",
		fastlog.String("query", query),
		fastlog.Any("keywords", keywords),
	)

	// 搜索笔记（全量内容）
	notes, err := noteService.SearchFull(keywords, limit)
	if err != nil || len(notes) == 0 {
		return nil
	}

	// 构建格式化上下文文本
	var b strings.Builder
	b.WriteString("以下是用户笔记库中与问题相关的笔记，请参考这些笔记内容回答用户的问题：\n\n")

	cards := make([]RecallCard, 0, len(notes))
	for _, note := range notes {
		content := note.Content
		truncated := false

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
