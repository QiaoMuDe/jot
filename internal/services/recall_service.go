package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-ego/gse"

	"gitee.com/MM-Q/fastlog"
)

// 停用词表，过滤分词结果中的高频无意义单字
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

// gse 分词器（懒初始化）
var (
	gseSeg     gse.Segmenter
	gseOnce    sync.Once
	gseInitErr error
)

// initGseSegmenter 初始化 gse 分词器，载入内置词典
func initGseSegmenter() {
	gseSeg = gse.Segmenter{}
	gseInitErr = gseSeg.LoadDictEmbed()
}

// tokenize 使用 gse 对输入文本做分词
// - 调用 gse.Cut 精确模式+HMM
// - 过滤停用词
// - 去重
func tokenize(text string) []string {
	gseOnce.Do(initGseSegmenter)
	if gseInitErr != nil {
		return nil
	}
	words := gseSeg.Cut(text, true)
	seen := make(map[string]struct{})
	var result []string
	for _, w := range words {
		if _, ok := seen[w]; !ok {
			// 过滤停用词（仅对单字词做停用词检查）
			runes := []rune(w)
			if len(runes) == 1 && isStopWord(runes[0]) {
				continue
			}
			seen[w] = struct{}{}
			result = append(result, w)
		}
	}
	return result
}

// maxRecallKeywords 卡片召回最大关键词数，防止超长 query 导致性能问题
const maxRecallKeywords = 20

// TruncateRecallCardsPreview 截断召回卡片列表的 Content 字段用于前端预览
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
// 对 query 使用 gse 分词 → 多关键词 OR 搜索笔记 → 格式化上下文 + 返回结构化卡片数据
func CardRecallSearch(ctx context.Context, query string, limit int, noteService *NoteService, notebookIDs ...uint) *CardRecallResult {
	if query == "" || limit <= 0 {
		return nil
	}

	keywords := tokenize(query)
	if len(keywords) == 0 {
		noteService.logger.Warnw("CardRecallSearch 分词结果为空",
			fastlog.String("query", query),
			fastlog.Any("gseInitErr", gseInitErr),
		)
		return nil
	}

	// 限制关键词数量，防止超长 LIKE 查询
	if len(keywords) > maxRecallKeywords {
		keywords = keywords[:maxRecallKeywords]
	}

	// 日志记录分词结果
	noteService.logger.Infow("CardRecallSearch 分词结果",
		fastlog.String("query", query),
		fastlog.Int("count", len(keywords)),
		fastlog.Any("keywords", keywords),
	)

	// 搜索笔记（全量内容）
	notes, err := noteService.SearchFull(keywords, limit, notebookIDs...)
	if err != nil || len(notes) == 0 {
		return nil
	}

	// 构建格式化上下文文本
	var b strings.Builder
	b.WriteString("以下是用户笔记库中与问题相关的笔记（来源：本地笔记，优先级最高），请优先参考这些笔记内容回答用户的问题：\n\n")

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
