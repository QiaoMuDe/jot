// Package chunk 提供文本切块工具，供向量召回前对文档进行分块。
package chunk

import "strings"

// ChunkDefault 使用默认单块上限 500 字对文本切块。
func ChunkDefault(content string) []string {
	return ChunkText(content, 500)
}

// ChunkText 将文本按 Markdown 标题（行首 "## " 或 "### "）与空行分段，
// 单块内容不超过 maxRunes 个 rune（按 Unicode 字符计数，安全处理多字节字符）；
// 超过上限的段落会被硬切。每块会 trim 空白，空块被跳过。
func ChunkText(content string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 500
	}

	var chunks []string
	var cur []string // 当前块的原始行

	// flush 将当前累积的行合并为一块并写入结果；超长时硬切。
	flush := func() {
		text := strings.TrimSpace(strings.Join(cur, "\n"))
		cur = nil
		if text == "" {
			return
		}
		if runeLen(text) > maxRunes {
			chunks = append(chunks, hardSplit(text, maxRunes)...)
			return
		}
		chunks = append(chunks, text)
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case isHeading(trimmed):
			// 标题行：结束当前块，标题作为新块的第一行
			flush()
			cur = append(cur, line)
		case trimmed == "":
			// 空行：段落分隔
			flush()
		default:
			cur = append(cur, line)
			// 当前块已超限则立即落块（避免单块无限膨胀）
			if runeLen(strings.Join(cur, "\n")) > maxRunes {
				flush()
			}
		}
	}
	flush()
	return chunks
}

// isHeading 判断一行是否为 Markdown 二级/三级标题。
func isHeading(line string) bool {
	return strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")
}

// runeLen 返回字符串的 rune 数量（Unicode 安全）。
func runeLen(s string) int {
	return len([]rune(s))
}

// hardSplit 将超长文本按 maxRunes 个 rune 硬切为多段，不会切断多字节字符。
func hardSplit(s string, maxRunes int) []string {
	runes := []rune(s)
	var out []string
	for i := 0; i < len(runes); i += maxRunes {
		end := i + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, strings.TrimSpace(string(runes[i:end])))
	}
	// 剔除硬切可能产生的空块
	filtered := out[:0]
	for _, seg := range out {
		if seg != "" {
			filtered = append(filtered, seg)
		}
	}
	return filtered
}
