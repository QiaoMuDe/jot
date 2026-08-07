package services

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

// MergeRecallCards 合并多路召回卡片并按 note_id 去重（保留先出现的），返回新切片
// 用于多路召回结果合并，避免前端覆盖式事件丢失任一路结果
func MergeRecallCards(lists ...[]RecallCard) []RecallCard {
	seen := make(map[uint]struct{})
	var merged []RecallCard
	for _, list := range lists {
		for _, card := range list {
			if _, ok := seen[card.ID]; ok {
				continue
			}
			seen[card.ID] = struct{}{}
			merged = append(merged, card)
		}
	}
	return merged
}

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
