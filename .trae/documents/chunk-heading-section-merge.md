# 方案 A：标题块合并重构切块逻辑（更完善版）

## Summary

重写 `internal/services/chunk.go` 的切块逻辑，对齐主流 RAG 分块语义（LangChain MarkdownHeaderTextSplitter / LlamaIndex header\_stack）：

1. **标题与内容强制耦合**：标题行不再因"标题 + 空行"而孤立成块，标题必须与后续正文合并（治本：杜绝纯标题/索引块抢占召回名额）
2. **一级标题纳入标题链**：`# X` 参与分块与链栈，不再被当普通文本
3. **标题链升级为多级栈**（1-6 级）：嵌套子节（`###`）的块自动补全父级标题（`##`）
4. **围栏代码块保护**：\`\`\` / \~\~\~ 代码块内空行、伪标题行不触发切块
5. **空节丢弃**：无正文的孤立标题不产生噪音块

改动范围：仅 `chunk.go` + `chunk_test.go`。⚠️ 分块规则变更后**已有量化数据需清空重新量化**。

## Current State Analysis

现状 [chunk.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go) 主循环：

```
标题行(##/###) → flush() → updateHeadingChain → 标题作为新块第一行
空行          → flush()   ← 问题根源：标题+空行 → 标题单独成块
普通行        → append，超限立即 flush
```

四个缺陷（用户报告症状的直接原因）：

| 缺陷                                    | 代码位置                                                                                       | 后果                                 |
| ------------------------------------- | ------------------------------------------------------------------------------------------ | ---------------------------------- |
| `isMarkdownHeading` 只认 `## ` /`### `  | [chunk.go L105-L108](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L105-L108) | `# 大标题` 被当普通文本；后跟空行时孤立成块，后续正文块补不上链 |
| 空行无条件 flush                           | [chunk.go L50-L52](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L50-L52)     | "标题 + 空行" → 孤立标题块                  |
| 标题紧邻标题时旧标题落块                          | flush 无纯标题检测                                                                               | "## A\n## B" → A 成空块               |
| `updateHeadingChain` 只支持两级            | [chunk.go L67-L78](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L67-L78)     | `###` 块丢失父级 `##` 指引；`#` 完全脱离链      |

后果链：纯标题/目录索引块主题词密度高 → embedding 后与 query 余弦距离更小 → TopN 被索引块占满 → 正文块未被召回 → 用户问答"只召回标题和目录，其他没响应"。

引用范围已确认（grep）：`isMarkdownHeading` / `updateHeadingChain` / `startsWithHeading` / `splitWithHeading` 均只在 chunk.go 内部使用，可安全重写。调用方仅 [vector\_service.go L67](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L67) 的 `ChunkContent(note.Content, 500)`，函数签名不变则调用方零改动。

## Proposed Changes

### 文件 1：`internal/services/chunk.go`（重写核心逻辑）

保留：`ChunkContent` 签名、`runeLen`、`hardSplit`、`Float32ToBlob`、`BlobToFloat32`。
删除：`isMarkdownHeading`、`updateHeadingChain`、`startsWithHeading`（被新函数替代）。

**① 新增** **`headingLevel(line string) int`** — 识别 ATX 标题级别 1-6，非标题返回 0

```go
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
```

边界：`#tag` → 0；`##`（无后续空格）→ 0；`####### x`（7 个#）→ 0。

**② 新增** **`pushHeadingStack(stack []string, heading string) []string`** — 多级标题链栈（对齐 LlamaIndex header\_stack）

```go
// pushHeadingStack 将新标题压入标题链栈：保留所有级别 < 新标题的父级，同级/更深级被新标题取代
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
```

**③ 新增** **`prependChain(text string, stack []string) string`** — 块首标题链补全（父级补全，非简单整链）

```go
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
```

**④ 新增** **`isCodeFence(line string) bool`** — 围栏代码块识别

````go
// isCodeFence 判断是否为围栏代码块的开闭行（``` 或 ~~~，含语言标注如 ```go）
func isCodeFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}
````

**⑤ 主循环重写** — `ChunkContent` 内部

```go
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
		if !(len(cur) == 1 && headingLevel(strings.TrimSpace(cur[0])) > 0) {
			flush()
		}
	default:
		cur = append(cur, line)
		if runeLen(strings.Join(cur, "\n")) > maxRunes {
			flush()
		}
	}
}
flush()
return chunks
```

**⑥** **`splitWithHeading`** **简化** — 非首段统一用 `prependChain`

```go
// splitWithHeading 对超长块硬切，并为非首段补充所属标题链指引
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
```

**⑦ 更新** **`ChunkContent`** **doc 注释**，描述新规则（多级标题栈 1-6 级 / 标题块合并 / 空节丢弃 / 代码块保护 / 块首父级链补全）。

### 文件 2：`internal/services/chunk_test.go`

**更新** **`TestChunkHeadings`**（行为变化：`### 1.1` 块现在补全父级 `## 第一章`，首行变为父级）：

```go
// 断言 chunks[0] 以 "## 第一章" 开头（不变）
// 断言 chunks[1] 前缀为 "## 第一章\n### 1.1"（父级补全，原为 "### 1.1"）
// 断言 chunks[2] 以 "## 第二章" 开头（不变）
// 块数量仍为 3
```

**新增测试用例**：

| 测试                             | 输入要点                                                                      | 断言                                                                                                                 |
| ------------------------------ | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `TestChunkHeadingBlankMerge`   | `"## A\n\n正文"`                                                            | 1 块，内容 `"## A\n正文"`（标题+空行+正文合并）                                                                                    |
| `TestChunkH1NotIsolated`       | `"# 大标题\n\n## 子节\n正文"`                                                    | `# 大标题` 不孤立成块；正文块为 `"# 大标题\n## 子节\n正文"`（一级标题进链）                                                                    |
| `TestChunkEmptySectionDropped` | `"## A\n## B\n正文"`                                                        | 1 块 `"## B\n正文"`（空节 A 丢弃）                                                                                          |
| `TestChunkNestedParentChain`   | `"## 第一章\n### 1.1\n正文"`                                                   | 1 块 `"## 第一章\n### 1.1\n正文"`（子节补父级）                                                                                 |
| `TestChunkHeadingLevel4`       | `"#### 小节\n正文"`                                                           | 1 块 `"#### 小节\n正文"`（4 级标题支持）                                                                                       |
| `TestChunkCodeFenceProtected`  | `"## 示例\n```go\n// # 伪标题\n\nx := 1\n```\n\n### 原理\n正文"`                   | 代码块内空行/伪标题不切块，代码块完整，标题节正常                                                                                          |
| `TestChunkReportedScenario`    | 用户报告场景 `"# 大标题\n\n## 目录\n- [A](#a)\n- [B](#b)\n\n## A\n正文A\n\n## B\n正文B"` | 3 块；块0 = `"# 大标题\n## 目录\n- [A](#a)\n- [B](#b)"`（目录带父标题）；块1 = `"# 大标题\n## A\n正文A"`；块2 = `"# 大标题\n## B\n正文B"`；无孤立标题块 |

现有 `TestChunkLongChinese` / `TestChunkNoHeading` / `TestChunkHardSplit` / `TestChunkDefaultMaxRunes` / `TestFloat32BlobRoundTrip` 不涉及标题，保持通过。

## Assumptions & Decisions

1. **标题级别支持 1-6 级**（`#` 到 `######`），`#######`（7 个）与 `#tag`、孤行 `##` 不算标题
2. **空节标题丢弃**：无正文的孤立标题（`## A` 后直接 `## B`）不产生块，但保留在链栈中作为后续子节的父级指引
3. **围栏代码块保护**：\`\`\` / \~\~\~ 开启后，块内空行、伪标题（如 `# comment`）不触发切块；超长代码块留待最终 flush 硬切（`splitWithHeading` 补链兜底），不中途切断保证完整性
4. **目录（TOC）不专门识别**：目录链接行作为普通内容归属其标题节，与主流项目（LangChain/LlamaIndex）一致；方案 A 后目录块带父级标题，且召回侧 `adjacentBlocks=1` 相邻块补充兜底
5. **函数签名不变**：`ChunkContent(content string, maxRunes int) []string`，[vector\_service.go L67](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L67) 调用方零改动
6. **不做方案 C（召回侧纯索引块兜底）**：属可选后续，本次仅治本
7. ⚠️ **需重新量化**：分块规则变更后旧向量基于旧分块，须清空 `note_vectors` 并重新 IndexNotes，否则召回仍走旧块

## Verification

1. `go test ./internal/services/ -run TestChunk -v` — 全部切块测试通过
2. `go test ./internal/services/` — 全量 services 测试回归（含向量检索测试）
3. `go build ./...` — 全项目编译通过
4. `golangci-lint run ./...` — 0 issues（无未使用函数残留）
5. 可选（需用户本机）：`wails dev` 后清空量化数据重新索引，开启卡片召回提问验证正文可被召回

