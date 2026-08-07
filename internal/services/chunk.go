package services

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// ChunkContent 将笔记内容按 Markdown 结构切块，并为每个块补充所属标题链（结构感知切片）
// 规则（对齐 LangChain MarkdownHeaderTextSplitter / LlamaIndex header_stack 语义）：
//   - 标题级别 1-6 级（# ~ ######），标题行作为"节的开始标记"开启新块
//   - 标题与内容强制耦合：标题行后跟空行时不落块，等待与后续正文合并，杜绝孤立标题块
//   - 空节（无正文的孤立标题）直接丢弃，不产生噪音块；标题保留在链栈中作为后续子节父级指引
//   - 块首补全所属标题链：块首非标题补完整链，块首已是标题仅补更高级父级
//   - ``` / ~~~ 围栏代码块保护：块内空行、伪标题行不触发切块
//   - 单块超过 maxRunes 时按 rune 硬切，硬切非首段同样补链
//
// 返回的每块长度（含标题链）不超过 maxRunes；maxRunes<=0 时使用默认值 500
func ChunkContent(content string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 500
	}

	var chunks []string
	var cur []string   // 当前块原始行
	var stack []string // 当前所属标题链栈（多级，如 ["# 大标题", "## 目录"]）
	inCode := false    // 是否处于围栏代码块内

	// flush 将当前累积行合并为一块；纯标题块（空节）丢弃；超长硬切；块首补父级标题链
	flush := func() {
		text := strings.TrimSpace(strings.Join(cur, "\n"))
		cur = nil
		if text == "" {
			return
		}
		// 纯标题块（空节：单行且是标题）不落块，标题已留在 stack 中作后续父级指引
		if !strings.Contains(text, "\n") && headingLevel(text) > 0 {
			return
		}
		// 补全所属标题链（父级链）
		text = prependChain(text, stack)
		if runeLen(text) > maxRunes {
			chunks = append(chunks, splitWithHeading(text, maxRunes, stack)...)
			return
		}
		chunks = append(chunks, text)
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case inCode:
			// 代码块内：空行/伪标题不切块，原样累积（超限留待最终 flush 硬切，保证代码块完整性）
			cur = append(cur, line)
			if isCodeFence(trimmed) {
				inCode = false
			}
		case isCodeFence(trimmed):
			// 围栏代码块开启：进入代码模式，开启行保留
			inCode = true
			cur = append(cur, line)
		case headingLevel(trimmed) > 0:
			// 标题行：结束当前块，更新标题链栈，以标题开启新块
			flush()
			stack = pushHeadingStack(stack, trimmed)
			cur = append(cur, line)
		case trimmed == "":
			// 空行：段落分隔。当前块仅含标题行时不落块（标题+空行等待与正文合并）
			if len(cur) != 1 || headingLevel(strings.TrimSpace(cur[0])) <= 0 {
				flush()
			}
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

// headingLevel 判断 Markdown ATX 标题级别（1-6），非标题返回 0
// 规则：行首连续 1-6 个 '#' 且其后紧跟空格（"#tag"、"##"分隔线不算标题）
func headingLevel(line string) int {
	if line == "" {
		return 0
	}
	n := 0
	for n < len(line) && n < 6 && line[n] == '#' {
		n++
	}
	if n == 0 || n == len(line) || line[n] != ' ' {
		return 0
	}
	return n
}

// pushHeadingStack 将新标题压入标题链栈：保留所有级别 < 新标题的父级，同级/更深级被新标题取代
// 语义对齐 LlamaIndex header_stack：
// 例：[##A,###B] 遇 ##C → [##C]；[##A,###B] 遇 ###D → [##A,###D]；[##A,###B] 遇 #E → [#E]
func pushHeadingStack(stack []string, heading string) []string {
	lvl := headingLevel(heading)
	out := make([]string, 0, len(stack)+1)
	for _, h := range stack {
		if headingLevel(h) < lvl {
			out = append(out, h)
		}
	}
	return append(out, heading)
}

// prependChain 为文本补充所属标题链指引：
// 块首非标题 → 补完整标题链；块首已是标题 → 仅补更高级父级（避免重复自身）
// 例：stack=[##A,###B]，text="正文" → "##A\n###B\n正文"；text="###B\n正文" → "##A\n###B\n正文"
func prependChain(text string, stack []string) string {
	if len(stack) == 0 {
		return text
	}
	first := text
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		first = text[:idx]
	}
	fl := headingLevel(strings.TrimSpace(first))
	if fl == 0 {
		return strings.Join(stack, "\n") + "\n" + text
	}
	// 块首已是标题：拼接更高级父级
	var parents []string
	for _, h := range stack {
		if headingLevel(h) < fl {
			parents = append(parents, h)
		}
	}
	if len(parents) == 0 {
		return text
	}
	return strings.Join(parents, "\n") + "\n" + text
}

// isCodeFence 判断是否为围栏代码块的开闭行（``` 或 ~~~，含语言标注如 ```go）
func isCodeFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

// splitWithHeading 对超长块硬切，并为非首段补充所属标题链指引
// 首段可能已以标题开头或已由 flush 补链，无需处理
func splitWithHeading(text string, maxRunes int, stack []string) []string {
	segs := hardSplit(text, maxRunes)
	if len(segs) <= 1 {
		return segs
	}
	for i := 1; i < len(segs); i++ {
		segs[i] = prependChain(segs[i], stack)
	}
	return segs
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
