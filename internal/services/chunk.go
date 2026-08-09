package services

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

// ChunkMeta 笔记元数据，用于在每个分块前注入前缀以提升检索命中率
type ChunkMeta struct {
	Title     string
	Tags      []string
	CreatedAt time.Time
}

// formatMetaPrefix 按统一模板生成分块元数据前缀：
//   - 笔记标题：{title}
//   - 分类标签：{tag1、tag2}（无标签时整行省略；多个标签用中文顿号「、」分隔）
//   - 创建时间：{2006-01-02}（精确到天）
//   - 笔记核心内容：
//
// 末尾以「笔记核心内容：」结尾，不带尾随换行（后续拼接标题链+正文时由调用方加换行）
func formatMetaPrefix(meta ChunkMeta) string {
	var b strings.Builder
	b.WriteString("笔记标题：")
	b.WriteString(meta.Title)
	b.WriteString("\n")
	if len(meta.Tags) > 0 {
		b.WriteString("分类标签：")
		b.WriteString(strings.Join(meta.Tags, "、"))
		b.WriteString("\n")
	}
	b.WriteString("创建时间：")
	b.WriteString(meta.CreatedAt.Format("2006-01-02"))
	b.WriteString("\n")
	b.WriteString("笔记核心内容：")
	return b.String()
}

// stripMetaPrefix 剥离分块元数据前缀，返回正文部分（标题链+内容）
// 前缀由 formatMetaPrefix 生成，以 "笔记核心内容：\n" 为分隔标记；找该标记首次出现位置取其后内容
// 找不到标记时（旧数据/无前缀）原样返回，保证向后兼容
func stripMetaPrefix(text string) string {
	const marker = "笔记核心内容：\n"
	if idx := strings.Index(text, marker); idx >= 0 {
		return text[idx+len(marker):]
	}
	return text
}

// normalizeChunkSource 归一化分块源文本，压缩非代码围栏行的连续空白：
//   - 连续 2+ 个 ASCII 空格/制表符折叠为 1 个空格
//   - 行尾空白去除
//   - 代码围栏（``` / ~~~）内的行不做处理，保护代码缩进
//
// 背景：PPT/PDF 转换出的 markdown 表格常带大量填充空格（如 "| 小时数据 |·····| 2061 |"），
// 既稀释嵌入向量质量，又挤占 maxRunes 预算；归一化后提升表格类块的检索命中率
func normalizeChunkSource(content string) string {
	lines := strings.Split(content, "\n")
	inCode := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inCode {
			// 围栏内：检查是否闭合，内容原样保留
			if isCodeFence(trimmed) {
				inCode = false
			}
			continue
		}
		if isCodeFence(trimmed) {
			inCode = true
			continue
		}
		lines[i] = collapseSpaces(line)
	}
	return strings.Join(lines, "\n")
}

// collapseSpaces 将连续 2+ 个空格/制表符折叠为 1 个空格，并去除行尾空白
func collapseSpaces(line string) string {
	var b strings.Builder
	b.Grow(len(line))
	prevSpace := false
	for _, r := range line {
		if r == ' ' || r == '\t' {
			if prevSpace {
				continue
			}
			prevSpace = true
			b.WriteByte(' ')
		} else {
			prevSpace = false
			b.WriteRune(r)
		}
	}
	return strings.TrimRight(b.String(), " \t")
}

// isTableRowLine 判断是否为 markdown 管道表格行（行首以 | 开头）
func isTableRowLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// isTableSeparatorLine 判断是否为 markdown 表格分隔线（仅含 | - : 空格/制表符且含 -）
func isTableSeparatorLine(line string) bool {
	hasDash := false
	for _, r := range line {
		switch r {
		case '|', '-', ':', ' ', '\t':
			if r == '-' {
				hasDash = true
			}
		default:
			return false
		}
	}
	return hasDash
}

// hasTableDataRow 判断文本中是否含表格数据行（排除表头行本身与分隔线）
// 用于决定是否需要给块补充表头上下文
func hasTableDataRow(text, header string) bool {
	headerTrim := strings.TrimSpace(header)
	for _, ln := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(ln)
		if !isTableRowLine(trimmed) || isTableSeparatorLine(trimmed) {
			continue
		}
		if trimmed == headerTrim {
			continue
		}
		return true
	}
	return false
}

// ChunkContent 将笔记内容按 Markdown 结构切块，并为每个块补充所属标题链（结构感知切片）
// 规则（对齐 LangChain MarkdownHeaderTextSplitter / LlamaIndex header_stack 语义）：
//   - 标题级别 1-6 级（# ~ ######），标题行作为"节的开始标记"开启新块
//   - 段落聚合：空行不触发切块，作为段落分隔保留在块内，多段落累积直到接近 maxRunes 才落块
//   - 空节（无正文的孤立标题）直接丢弃，不产生噪音块；标题保留在链栈中作为后续子节父级指引
//   - 块首补全所属标题链：块首非标题补完整链，块首已是标题仅补更高级父级
//   - ``` / ~~~ 围栏代码块保护：块内空行、伪标题行不触发切块
//   - 单块超过 maxRunes 时按 rune 硬切，硬切非首段同样补链
//
// 返回的每块长度（含元数据前缀+标题链）不超过 maxRunes；maxRunes<=0 时使用默认值 500
func ChunkContent(content string, maxRunes int, meta ChunkMeta) []string {
	if maxRunes <= 0 {
		maxRunes = 500
	}
	// 归一化源文本：压缩非代码围栏行的连续空白（表格填充空格/行尾空白），提升嵌入质量
	content = normalizeChunkSource(content)

	var chunks []string
	var cur []string   // 当前块原始行
	var stack []string // 当前所属标题链栈（多级，如 ["# 大标题", "## 目录"]）
	inCode := false    // 是否处于围栏代码块内
	// tableHeader 当前 markdown 表格的表头行；表头与数据行常被切到不同块，
	// 记录后在 flush 时给含数据行的块补上表头，让"列名语义"进入每个表格行块的嵌入
	tableHeader := ""

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
		// 表格行块补充表头上下文：块内含表格数据行但缺表头时，在块首补一行表头
		if tableHeader != "" && !strings.Contains(text, tableHeader) && hasTableDataRow(text, tableHeader) {
			text = tableHeader + "\n" + text
		}
		// 拼接元数据前缀；maxRunes 预算包含前缀长度（最终块总长 runeLen(prefix+text) 不超过 maxRunes）
		prefix := formatMetaPrefix(meta)
		if runeLen(prefix)+1+runeLen(text) > maxRunes {
			// 超限硬切：传入未含前缀的标题链+正文，每段独立拼接 prefix
			chunks = append(chunks, splitWithHeading(text, maxRunes, stack, prefix)...)
			return
		}
		chunks = append(chunks, prefix+"\n"+text)
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
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
			// 段落聚合：空行作为段落分隔保留在块内，不触发切块
			// 块首空行跳过（避免块首留空行）；累积后超限才落块
			if len(cur) == 0 {
				continue
			}
			cur = append(cur, line)
			if runeLen(strings.Join(cur, "\n")) > maxRunes {
				flush()
			}
		default:
			// 表格表头识别：本行是表格行且下一行是分隔线 → 记录为新表头（新表头覆盖旧表头）
			if isTableRowLine(trimmed) && i+1 < len(lines) && isTableSeparatorLine(strings.TrimSpace(lines[i+1])) {
				tableHeader = trimmed
			}
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

// splitWithHeading 对超长块硬切，并为每段补充元数据前缀 + 所属标题链指引
// text 为未含前缀的标题链+正文；prefix 为元数据前缀，硬切后每段都拼接 prefix+"\n"+prependChain(段, stack)
// 首段同样适用（首段也加 prefix，保持一致）；maxRunes 预算包含前缀长度
func splitWithHeading(text string, maxRunes int, stack []string, prefix string) []string {
	// 预留元数据前缀 + 换行的 rune 预算，硬切作用于标题链+正文部分
	budget := maxRunes - runeLen(prefix) - 1
	if budget < 1 {
		budget = 1
	}
	segs := hardSplit(text, budget)
	for i := range segs {
		segs[i] = prefix + "\n" + prependChain(segs[i], stack)
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
