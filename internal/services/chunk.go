package services

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// ChunkContent 将笔记内容按 Markdown 结构切块，并为每个块补充所属标题链（结构感知切片）
// 规则：按 ## / ### 标题分块，无标题的按空行分段，单块超过 maxRunes 时按 rune 硬切
// 标题指引：块首不是标题行时，自动把所属标题链（如 "## A" 或 "## A\n### B"）拼到块首，
// 使标题语义进入 embedding 向量与召回文本；硬切产生的非首段同样补链
// 返回的每块长度（含标题链）不超过 maxRunes；maxRunes<=0 时使用默认值 500
func ChunkContent(content string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 500
	}

	var chunks []string
	var cur []string   // 当前块的原始行
	var chain []string // 当前所属标题链（如 ["## A", "### B"]）

	// flush 将当前累积的行合并为一块写入结果；块首非标题时补标题链，超长时按 rune 硬切
	flush := func() {
		text := strings.TrimSpace(strings.Join(cur, "\n"))
		cur = nil
		if text == "" {
			return
		}
		// 块首不是标题行时，补上所属标题链作为指引
		if !startsWithHeading(text) && len(chain) > 0 {
			text = strings.Join(chain, "\n") + "\n" + text
		}
		if runeLen(text) > maxRunes {
			chunks = append(chunks, splitWithHeading(text, maxRunes, chain)...)
			return
		}
		chunks = append(chunks, text)
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case isMarkdownHeading(trimmed):
			// 标题行：结束当前块，更新标题链，并以标题作为新块的第一行
			flush()
			chain = updateHeadingChain(chain, trimmed)
			cur = append(cur, line)
		case trimmed == "":
			// 空行：段落分隔
			flush()
		default:
			cur = append(cur, line)
			// 当前块已超限则立即落块，避免单块无限膨胀
			if runeLen(strings.Join(cur, "\n")) > maxRunes {
				flush()
			}
		}
	}
	flush()
	return chunks
}

// updateHeadingChain 根据标题行更新所属标题链
// 三级标题 "### Y" 保留链中的二级标题后追加；二级标题 "## X" 重置链
func updateHeadingChain(chain []string, heading string) []string {
	if strings.HasPrefix(heading, "### ") {
		var out []string
		for _, h := range chain {
			if !strings.HasPrefix(h, "### ") {
				out = append(out, h)
			}
		}
		return append(out, heading)
	}
	return []string{heading}
}

// startsWithHeading 判断文本首行是否为二级/三级标题
func startsWithHeading(text string) bool {
	first := text
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		first = text[:idx]
	}
	return isMarkdownHeading(strings.TrimSpace(first))
}

// splitWithHeading 对超长块硬切，并为非首段补充所属标题链指引
// 首段可能已以标题开头或已由 flush 补链，无需处理
func splitWithHeading(text string, maxRunes int, chain []string) []string {
	segs := hardSplit(text, maxRunes)
	if len(segs) <= 1 || len(chain) == 0 {
		return segs
	}
	prefix := strings.Join(chain, "\n")
	for i := 1; i < len(segs); i++ {
		if !startsWithHeading(segs[i]) {
			segs[i] = prefix + "\n" + segs[i]
		}
	}
	return segs
}

// isMarkdownHeading 判断一行是否为 Markdown 二级/三级标题
func isMarkdownHeading(line string) bool {
	return strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")
}

// runeLen 返回字符串的 rune 数量（Unicode 安全）
func runeLen(s string) int {
	return len([]rune(s))
}

// hardSplit 将超长文本按 maxRunes 个 rune 硬切为多段，不会切断多字节字符
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

// Float32ToBlob 将 float32 切片序列化为小端字节序 BLOB
func Float32ToBlob(vec []float32) []byte {
	b := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}

// BlobToFloat32 反序列化 BLOB 为 float32 切片；BLOB 长度不是 4 的倍数时返回错误
func BlobToFloat32(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("BLOB 长度 %d 不是 4 的倍数", len(b))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}
