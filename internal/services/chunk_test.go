package services

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// emptyMetaPrefix 是空 ChunkMeta（Title="", Tags=nil, CreatedAt 零值）生成的前缀，
// 用于在断言中拼接期望的块内容（空 meta 时「分类标签」行省略，创建时间为零值 0001-01-01）
const emptyMetaPrefix = "笔记标题：\n创建时间：0001-01-01\n笔记核心内容："

// TestChunkLongChinese 验证 2000+ 字中文输入能分出至少 4 块，且每块不超过 maxRunes 个 rune
func TestChunkLongChinese(t *testing.T) {
	// 60 段 × 54 字 = 3240 字（段间用空行分隔）
	var b strings.Builder
	for i := 0; i < 60; i++ {
		b.WriteString(strings.Repeat("量化学习与向量检索", 6)) // 9 字 × 6 = 54 字
		b.WriteString("\n\n")
	}
	content := b.String()

	chunks := ChunkContent(content, 500, ChunkMeta{})
	if len(chunks) < 4 {
		t.Fatalf("2000+ 字中文期望至少 4 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkHeadings 验证按 ## / ### 标题分块，且子节 ### 块自动补全父级 ## 标题链
func TestChunkHeadings(t *testing.T) {
	content := "## 第一章 简介\n这是第一章的内容，介绍向量检索的基本概念。\n\n### 1.1 原理\n本节讲解向量化的原理。\n\n## 第二章 应用\n这里讲述向量检索在实际场景中的应用。"
	chunks := ChunkContent(content, 500, ChunkMeta{})

	if len(chunks) != 3 {
		t.Fatalf("期望按标题分出 3 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if !strings.HasPrefix(chunks[0], emptyMetaPrefix+"\n"+"## 第一章") {
		t.Errorf("第 0 块应以 \"## 第一章\" 开头（含元数据前缀），实际: %q", chunks[0])
	}
	// 子节块首补全父级标题链（原为 "### 1.1"）
	if !strings.HasPrefix(chunks[1], emptyMetaPrefix+"\n"+"## 第一章 简介\n### 1.1 原理") {
		t.Errorf("第 1 块应补全父级链 \"## 第一章 简介\\n### 1.1 原理\"（含元数据前缀），实际: %q", chunks[1])
	}
	if !strings.HasPrefix(chunks[2], emptyMetaPrefix+"\n"+"## 第二章") {
		t.Errorf("第 2 块应以 \"## 第二章\" 开头（含元数据前缀），实际: %q", chunks[2])
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkNoHeading 验证无标题短内容段落聚合为一块（空行作为段落分隔保留在块内）
func TestChunkNoHeading(t *testing.T) {
	content := "第一段内容，没有标题。\n\n第二段内容，也没有标题。\n\n第三段内容，同样没有标题。"
	chunks := ChunkContent(content, 500, ChunkMeta{})

	if len(chunks) != 1 {
		t.Fatalf("短段落聚合期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	want := emptyMetaPrefix + "\n" + "第一段内容，没有标题。\n\n第二段内容，也没有标题。\n\n第三段内容，同样没有标题。"
	if chunks[0] != want {
		t.Errorf("期望块内容 %q，实际: %q", want, chunks[0])
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkHardSplit 验证超长单段（无标题无空行）按 rune 硬切且不丢内容
func TestChunkHardSplit(t *testing.T) {
	// 单段 30 句 × 28 字 = 840 字，无标题无空行
	long := strings.Repeat("这是一段没有标题也没有空行的超长内容，用来验证硬切逻辑。", 30)
	chunks := ChunkContent(long, 500, ChunkMeta{})

	if len(chunks) < 2 {
		t.Fatalf("超长单段期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
	// 硬切不丢内容：每段都带元数据前缀，去除前缀后拼接应与原文一致
	prefixLine := emptyMetaPrefix + "\n"
	stripped := make([]string, len(chunks))
	for i, c := range chunks {
		stripped[i] = strings.TrimPrefix(c, prefixLine)
	}
	if joined := strings.Join(stripped, ""); joined != long {
		t.Error("硬切后（去除前缀）拼接结果与原文不一致")
	}
}

// TestChunkDefaultMaxRunes 验证 maxRunes<=0 时使用默认值 500
func TestChunkDefaultMaxRunes(t *testing.T) {
	content := strings.Repeat("中", 1200)
	chunks := ChunkContent(content, 0, ChunkMeta{}) // 应使用默认 500

	if len(chunks) != 3 {
		t.Fatalf("1200 字默认上限 500 期望 3 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过默认上限 500", i, runeLen(c))
		}
	}
}

// TestFloat32BlobRoundTrip 验证 float32 BLOB 序列化往返一致
func TestFloat32BlobRoundTrip(t *testing.T) {
	vec := []float32{0.1, -1.5, 2.0, 3.14159, 0}
	blob := Float32ToBlob(vec)
	if len(blob) != len(vec)*4 {
		t.Fatalf("BLOB 长度期望 %d，实际 %d", len(vec)*4, len(blob))
	}
	got, err := BlobToFloat32(blob)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if !reflect.DeepEqual(got, vec) {
		t.Errorf("往返结果不一致: got %v, want %v", got, vec)
	}

	// 非法长度应报错
	if _, err := BlobToFloat32([]byte{1, 2, 3}); err == nil {
		t.Error("长度为 3 的 BLOB 应返回错误")
	}
}

// TestChunkHeadingBlankMerge 验证标题行后跟空行时与后续正文合并为一块（段落聚合下空行保留在块内作分隔）
func TestChunkHeadingBlankMerge(t *testing.T) {
	chunks := ChunkContent("## A\n\n正文", 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("标题+空行+正文期望合并为 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## A\n\n正文" {
		t.Errorf("期望块内容 \"## A\\n\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkH1NotIsolated 验证一级标题 # 参与分块且不孤立成块，正文块带完整父级链
func TestChunkH1NotIsolated(t *testing.T) {
	chunks := ChunkContent("# 大标题\n\n## 子节\n正文", 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("一级标题场景期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"# 大标题\n## 子节\n正文" {
		t.Errorf("期望块内容 \"# 大标题\\n## 子节\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkEmptySectionDropped 验证空节（无正文的孤立标题）被丢弃，不产生噪音块
func TestChunkEmptySectionDropped(t *testing.T) {
	chunks := ChunkContent("## A\n## B\n正文", 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("空节 A 应丢弃，期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## B\n正文" {
		t.Errorf("期望块内容 \"## B\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkNestedParentChain 验证嵌套子节 ### 块自动补全父级 ## 标题链
func TestChunkNestedParentChain(t *testing.T) {
	chunks := ChunkContent("## 第一章\n### 1.1\n正文", 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("嵌套子节期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## 第一章\n### 1.1\n正文" {
		t.Errorf("期望块内容 \"## 第一章\\n### 1.1\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkHeadingLevel4 验证四级标题 #### 同样参与分块与链栈
func TestChunkHeadingLevel4(t *testing.T) {
	chunks := ChunkContent("#### 小节\n正文", 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("四级标题期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"#### 小节\n正文" {
		t.Errorf("期望块内容 \"#### 小节\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkCodeFenceProtected 验证围栏代码块内空行、伪标题行不触发切块，代码块完整保留
func TestChunkCodeFenceProtected(t *testing.T) {
	content := "## 示例\n```go\n// # 伪标题\n\nx := 1\n```\n\n### 原理\n正文"
	chunks := ChunkContent(content, 500, ChunkMeta{})
	if len(chunks) != 2 {
		t.Fatalf("代码块场景期望 2 块，实际 %d 块: %v", len(chunks), chunks)
	}
	// 块 0：代码块完整，内部空行与伪标题未触发切块
	if !strings.Contains(chunks[0], "```go") || !strings.Contains(chunks[0], "// # 伪标题") ||
		!strings.Contains(chunks[0], "x := 1") || !strings.Contains(chunks[0], "```") {
		t.Errorf("第 0 块应包含完整代码块，实际: %q", chunks[0])
	}
	if strings.HasPrefix(chunks[0], "## 示例\n###") {
		t.Errorf("第 0 块不应在代码块内切出标题，实际: %q", chunks[0])
	}
	// 块 1：子节补父级标题链（含元数据前缀）
	if chunks[1] != emptyMetaPrefix+"\n"+"## 示例\n### 原理\n正文" {
		t.Errorf("期望块内容 \"## 示例\\n### 原理\\n正文\"（含元数据前缀），实际: %q", chunks[1])
	}
}

// TestChunkReportedScenario 回归用户报告场景：大标题+目录+分节正文，无孤立标题块，目录带父标题
func TestChunkReportedScenario(t *testing.T) {
	content := "# 大标题\n\n## 目录\n- [A](#a)\n- [B](#b)\n\n## A\n正文A\n\n## B\n正文B"
	chunks := ChunkContent(content, 500, ChunkMeta{})
	want := []string{
		emptyMetaPrefix + "\n" + "# 大标题\n## 目录\n- [A](#a)\n- [B](#b)",
		emptyMetaPrefix + "\n" + "# 大标题\n## A\n正文A",
		emptyMetaPrefix + "\n" + "# 大标题\n## B\n正文B",
	}
	if len(chunks) != len(want) {
		t.Fatalf("期望 %d 块，实际 %d 块: %v", len(want), len(chunks), chunks)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Errorf("第 %d 块期望 %q，实际 %q", i, want[i], chunks[i])
		}
	}
}

// TestChunkMetaPrefixWithTags 验证有标签时分块元数据前缀格式正确（标签用中文顿号分隔）
func TestChunkMetaPrefixWithTags(t *testing.T) {
	meta := ChunkMeta{
		Title:     "数据库设计",
		Tags:      []string{"架构", "后端"},
		CreatedAt: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}
	chunks := ChunkContent("## 设计\n正文内容", 600, meta)
	if len(chunks) == 0 {
		t.Fatal("期望至少 1 块，实际 0 块")
	}
	want := "笔记标题：数据库设计\n分类标签：架构、后端\n创建时间：2026-08-07\n笔记核心内容："
	for i, c := range chunks {
		if !strings.HasPrefix(c, want) {
			t.Errorf("第 %d 块前缀期望以 %q 开头，实际: %q", i, want, c)
		}
	}
}

// TestChunkMetaPrefixNoTags 验证无标签时「分类标签」行整行省略
func TestChunkMetaPrefixNoTags(t *testing.T) {
	meta := ChunkMeta{
		Title:     "日记",
		Tags:      []string{},
		CreatedAt: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}
	chunks := ChunkContent("## 设计\n正文内容", 600, meta)
	if len(chunks) == 0 {
		t.Fatal("期望至少 1 块，实际 0 块")
	}
	want := "笔记标题：日记\n创建时间：2026-08-07\n笔记核心内容："
	for i, c := range chunks {
		if !strings.HasPrefix(c, want) {
			t.Errorf("第 %d 块前缀期望以 %q 开头，实际: %q", i, want, c)
		}
		if strings.Contains(c, "分类标签：") {
			t.Errorf("第 %d 块不应包含分类标签行，实际: %q", i, c)
		}
	}
}

// TestChunkParagraphAggregation 验证段落聚合：多个短段落累积到接近 maxRunes 才切块，而非每个空行切一块
func TestChunkParagraphAggregation(t *testing.T) {
	// 10 段 × 每段约 30 字，段间空行分隔；总长约 300+ 字
	// maxRunes=500（含前缀 ~40）→ 正文预算约 460，10 段聚合后仍 < 460，应合成 1 块
	var b strings.Builder
	for i := 0; i < 10; i++ {
		b.WriteString("这是第" + string(rune('一'+i)) + "段短内容用于验证段落聚合。")
		b.WriteString("\n\n")
	}
	content := strings.TrimSpace(b.String())
	chunks := ChunkContent(content, 500, ChunkMeta{})

	if len(chunks) != 1 {
		t.Fatalf("10 段短内容聚合期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	// 块内应包含全部 10 段，空行作为段落分隔保留
	for i := 0; i < 10; i++ {
		marker := "这是第" + string(rune('一'+i)) + "段"
		if !strings.Contains(chunks[0], marker) {
			t.Errorf("聚合块应包含 %q，实际: %q", marker, chunks[0])
		}
	}
	if runeLen(chunks[0]) > 500 {
		t.Errorf("聚合块长度 %d 超过上限 500", runeLen(chunks[0]))
	}
}

// TestChunkParagraphAggregationSplit 验证段落聚合后超限时正确切块：多段累积超过 maxRunes 时按 rune 硬切
func TestChunkParagraphAggregationSplit(t *testing.T) {
	// 30 段 × 每段约 24 字 + 空行 = 总长约 720 字，超过 maxRunes=500（含前缀 ~40）→ 应切 2 块
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString("聚合切段测试段落内容编号" + string(rune('A'+i%26)) + "用于填充长度确保超限。")
		b.WriteString("\n\n")
	}
	content := strings.TrimSpace(b.String())
	chunks := ChunkContent(content, 500, ChunkMeta{})

	if len(chunks) < 2 {
		t.Fatalf("超长聚合内容期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestStripMetaPrefix 验证剥离分块元数据前缀：有标签/无标签前缀正确剥离，无前缀旧数据兜底返回原文
func TestStripMetaPrefix(t *testing.T) {
	// 场景1：有标签前缀
	withTags := formatMetaPrefix(ChunkMeta{
		Title:     "数据库设计",
		Tags:      []string{"架构", "后端"},
		CreatedAt: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}) + "\n## 表结构\n正文内容"
	got := stripMetaPrefix(withTags)
	want := "## 表结构\n正文内容"
	if got != want {
		t.Errorf("有标签前缀剥离失败：期望 %q，实际 %q", want, got)
	}

	// 场景2：无标签前缀（分类标签行省略）
	noTags := formatMetaPrefix(ChunkMeta{
		Title:     "日记",
		Tags:      []string{},
		CreatedAt: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}) + "\n正文"
	got = stripMetaPrefix(noTags)
	if got != "正文" {
		t.Errorf("无标签前缀剥离失败：期望 %q，实际 %q", "正文", got)
	}

	// 场景3：无前缀旧数据兜底返回原文
	legacy := "## 旧数据\n没有元数据前缀"
	got = stripMetaPrefix(legacy)
	if got != legacy {
		t.Errorf("无前缀旧数据应原样返回：期望 %q，实际 %q", legacy, got)
	}
}
