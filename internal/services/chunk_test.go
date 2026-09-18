package services

import (
	"reflect"
	"strconv"
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

	chunks := ChunkContent(content, 500, 500, ChunkMeta{})
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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})

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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})

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
	chunks := ChunkContent(long, 500, 500, ChunkMeta{})

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

// TestChunkClampDefensive 验证防御性钳制行为：
//   - targetRunes > maxRunes 时降级为 maxRunes（target=1200,max=600 → 等价 max=600 切块）
//   - maxRunes < 1 时置 1（单段被逐字硬切）
//   - targetRunes < 1 时置 1
func TestChunkClampDefensive(t *testing.T) {
	// target>max 降级：1200 字单段（无标题无空行），target=1200,max=600
	// 与 target=600,max=600 应产生完全一致的块（硬切预算均为 max）
	content := strings.Repeat("中", 1200)
	clamped := ChunkContent(content, 1200, 600, ChunkMeta{})
	normal := ChunkContent(content, 600, 600, ChunkMeta{})
	if !reflect.DeepEqual(clamped, normal) {
		t.Fatalf("target>max 应降级为 max 与 target=max 等价，\nclamped=%q\nnormal=%q", clamped, normal)
	}
	for i, c := range clamped {
		if runeLen(c) > 600 {
			t.Errorf("第 %d 块长度 %d 超过上限 600", i, runeLen(c))
		}
	}
	// max<1 → 置 1：单段被逐字硬切，块数 = rune 数
	tiny := ChunkContent(content, 0, 0, ChunkMeta{})
	if len(tiny) != runeLen(content) {
		t.Fatalf("max=0 置 1 后应逐字切块，期望 %d 块，实际 %d 块", runeLen(content), len(tiny))
	}
	// target<1 → 置 1：max=600 下 target=0 与 target=1 等价（均极小落刀点，同 max 硬切兜底）
	onlyMax := ChunkContent("## A\n"+"正文内容", 0, 600, ChunkMeta{})
	targetOne := ChunkContent("## A\n"+"正文内容", 1, 600, ChunkMeta{})
	if !reflect.DeepEqual(onlyMax, targetOne) {
		t.Fatalf("target=0 应置 1 与 target=1 等价，\nonlyMax=%q\ntargetOne=%q", onlyMax, targetOne)
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
	chunks := ChunkContent("## A\n\n正文", 500, 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("标题+空行+正文期望合并为 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## A\n\n正文" {
		t.Errorf("期望块内容 \"## A\\n\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkH1NotIsolated 验证一级标题 # 参与分块且不孤立成块，正文块带完整父级链
func TestChunkH1NotIsolated(t *testing.T) {
	chunks := ChunkContent("# 大标题\n\n## 子节\n正文", 500, 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("一级标题场景期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"# 大标题\n## 子节\n正文" {
		t.Errorf("期望块内容 \"# 大标题\\n## 子节\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkEmptySectionDropped 验证空节（无正文的孤立标题）被丢弃，不产生噪音块
func TestChunkEmptySectionDropped(t *testing.T) {
	chunks := ChunkContent("## A\n## B\n正文", 500, 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("空节 A 应丢弃，期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## B\n正文" {
		t.Errorf("期望块内容 \"## B\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkNestedParentChain 验证嵌套子节 ### 块自动补全父级 ## 标题链
func TestChunkNestedParentChain(t *testing.T) {
	chunks := ChunkContent("## 第一章\n### 1.1\n正文", 500, 500, ChunkMeta{})
	if len(chunks) != 1 {
		t.Fatalf("嵌套子节期望 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if chunks[0] != emptyMetaPrefix+"\n"+"## 第一章\n### 1.1\n正文" {
		t.Errorf("期望块内容 \"## 第一章\\n### 1.1\\n正文\"（含元数据前缀），实际: %q", chunks[0])
	}
}

// TestChunkHeadingLevel4 验证四级标题 #### 同样参与分块与链栈
func TestChunkHeadingLevel4(t *testing.T) {
	chunks := ChunkContent("#### 小节\n正文", 500, 500, ChunkMeta{})
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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})
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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})
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
	chunks := ChunkContent("## 设计\n正文内容", 600, 600, meta)
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
	chunks := ChunkContent("## 设计\n正文内容", 600, 600, meta)
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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})

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
	chunks := ChunkContent(content, 500, 500, ChunkMeta{})

	if len(chunks) < 2 {
		t.Fatalf("超长聚合内容期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkMixedLongInput 验证混合大输入（超长段落+代码围栏+多空行段落聚合）走
// curRunes 计数器路径无 panic、每块不超限、关键内容不丢失：
// 覆盖 addLine 的 4 个 append 点（围栏开启/代码块内/空行/正文）与多次超限 flush，
// 标题 append 点由现有 TestChunkHeadings 等覆盖，防止计数器与 Join 计算结果漂移导致输出变化或越界
func TestChunkMixedLongInput(t *testing.T) {
	var b strings.Builder
	// 顶层超长段落（远超单块上限，硬切时标题链栈为空 → 块长严格受限；无标题以免叠加硬切补链固有超限）
	b.WriteString(strings.Repeat("这是一段超过单块上限的超长正文内容，用于验证计数器路径的多次超限 flush。", 40)) // ≈1400 字
	b.WriteString("\n\n")
	// 代码围栏（代码块内空行/伪标题不切块，超限留待 flush 硬切）
	b.WriteString("```go\n")
	for i := 0; i < 50; i++ {
		b.WriteString("    x := " + strconv.Itoa(i) + " // 保留缩进\n")
	}
	b.WriteString("```\n\n")
	// 多空行段落聚合（空行分支 + 正文分支的累积超限切块），总量 1000+ 行
	for i := 0; i < 500; i++ {
		b.WriteString("聚合段落" + strconv.Itoa(i) + "：这是用于段落聚合与超限切块的多行文本。\n\n")
	}
	content := b.String()

	chunks := ChunkContent(content, 600, 600, ChunkMeta{Title: "混合输入", Tags: []string{"测试"}, CreatedAt: time.Now()})
	if len(chunks) < 10 {
		t.Fatalf("大型混合输入期望至少 10 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 600 {
			t.Errorf("第 %d 块长度 %d 超过上限 600", i, runeLen(c))
		}
		if c == "" {
			t.Errorf("第 %d 块为空块", i)
		}
	}
	// 关键内容不丢失：围栏代码末行与最后一段文本应出现在某块中
	joined := strings.Join(chunks, "")
	for _, marker := range []string{"x := 49", "聚合段落499"} {
		if !strings.Contains(joined, marker) {
			t.Errorf("混合输入丢失关键内容 %q", marker)
		}
	}
}

// TestChunkTargetMaxBigSection 验证「结构清晰、大节整段」：位于 target..max 之间的一段无空行正文
// 应整段成为一块，不被切成 target 那么碎（target=600,max=1500，一段约 1000 rune）
func TestChunkTargetMaxBigSection(t *testing.T) {
	// 单段约 1000 字（无标题无空行），介于 target(600) 与 max(1500) 之间
	content := strings.Repeat("整段大节正文内容用于验证大节不被切成 target 那么碎，保持段落完整性。", 20)
	chunks := ChunkContent(content, 600, 1500, ChunkMeta{})

	if len(chunks) != 1 {
		t.Fatalf("介于 target..max 的大段期望整段 1 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if runeLen(chunks[0]) > 1500 {
		t.Errorf("块长度 %d 超过硬上限 1500", runeLen(chunks[0]))
	}
	// 整段保留（去除前缀后与原文拼接一致）
	prefixLine := emptyMetaPrefix + "\n"
	if strings.TrimPrefix(chunks[0], prefixLine) != content {
		t.Error("大段内容未完整保留")
	}
}

// TestChunkTargetMaxParagraphCut 验证「纯文本、多段落聚合」：多个段落聚合到触及 target 时在段落边界落刀，
// 不切断段落（target=600,max=1500，5 段各约 250 字 → 约 2-3 块）
func TestChunkTargetMaxParagraphCut(t *testing.T) {
	var paras []string
	for i := 0; i < 5; i++ {
		paras = append(paras, "段落"+strconv.Itoa(i)+"："+strings.Repeat("这是用于段落聚合落刀测试的第"+strconv.Itoa(i)+"段填充内容，", 20))
	}
	content := strings.Join(paras, "\n\n")
	chunks := ChunkContent(content, 600, 1500, ChunkMeta{})

	if len(chunks) < 2 || len(chunks) > 3 {
		t.Fatalf("5 段各约 250 字聚合到 target=600 期望 2-3 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 1500 {
			t.Errorf("第 %d 块长度 %d 超过硬上限 1500", i, runeLen(c))
		}
	}
	// 段落不被切开：每段 marker 应完整出现在恰好一个块中（落刀仅在段落边界）
	prefixLine := emptyMetaPrefix + "\n"
	for i := range paras {
		marker := "第" + strconv.Itoa(i) + "段"
		count := 0
		for _, c := range chunks {
			body := strings.TrimPrefix(c, prefixLine)
			if strings.Contains(body, marker) {
				count++
			}
		}
		if count != 1 {
			t.Errorf("段落 %d 的 marker %q 出现在 %d 个块中，期望恰好 1（段落未被切开）", i, marker, count)
		}
	}
}

// TestChunkTargetMaxHardSplit 验证仅单个不可分语义单元真超 max 才硬切：一段约 1500 字无空行正文，
// target=600,max=1000 → 应硬切为 ≥2 块且每块 ≤ 1000（纯文本无标题，避免补链叠加超限）
func TestChunkTargetMaxHardSplit(t *testing.T) {
	content := strings.Repeat("这是一段超过硬上限且无空行的超长正文，用于验证单段真超 max 时的硬切兜底。", 40) // 约 1400 字
	chunks := ChunkContent(content, 600, 1000, ChunkMeta{})

	if len(chunks) < 2 {
		t.Fatalf("单段约 1400 字、max=1000 期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 1000 {
			t.Errorf("第 %d 块长度 %d 超过硬上限 1000", i, runeLen(c))
		}
	}
	// 硬切不丢内容：去除前缀拼接后与原文一致
	prefixLine := emptyMetaPrefix + "\n"
	stripped := make([]string, len(chunks))
	for i, c := range chunks {
		stripped[i] = strings.TrimPrefix(c, prefixLine)
	}
	if joined := strings.Join(stripped, ""); joined != content {
		t.Error("硬切后（去除前缀）拼接结果与原文不一致")
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

// TestNormalizeChunkSource 验证空白压缩：非代码围栏行的连续空格折叠、行尾空格去除，
// 代码围栏内内容（缩进/对齐空格）原样保留
func TestNormalizeChunkSource(t *testing.T) {
	input := "这是  普通  文本   有连续空格。\n" +
		"| 小时数据 |      2061      | 内容 |\n" +
		"```go\n" +
		"func main() {\n" +
		"    x := 1   // 保留缩进与空格\n" +
		"}\n" +
		"```\n" +
		"行尾空格  \n" +
		"下一段 内容"
	want := "这是 普通 文本 有连续空格。\n" +
		"| 小时数据 | 2061 | 内容 |\n" +
		"```go\n" +
		"func main() {\n" +
		"    x := 1   // 保留缩进与空格\n" +
		"}\n" +
		"```\n" +
		"行尾空格\n" +
		"下一段 内容"
	got := normalizeChunkSource(input)
	if got != want {
		t.Errorf("normalizeChunkSource 结果不符：\n期望 %q\n实际 %q", want, got)
	}
}

// TestChunkTableHeaderCarry 验证表格行块自动携带表头上下文：
// 表头与数据行在同一块（整块落袋，不进硬切路径）时，含数据行的块在块首补上表头（列名语义进入嵌入）；
// 无表格数据行的普通段落块不补表头
func TestChunkTableHeaderCarry(t *testing.T) {
	header := "| 数据类型 | 命令编码 | 上传内容 |"
	longRow := "| 小时数据 | 2061 | " + strings.Repeat("污染物浓度（标干、折算浓度）均值，排放量，流量，温度、压力、流速、氧含量、湿度等均值。", 2) + " |"
	content := "## 数据上传编码\n\n" +
		header + "\n" +
		"| --- | --- | --- |\n" +
		"| 分钟数据 | 2051 | 颗粒物浓度均值 |\n" +
		longRow + "\n" +
		"| 日数据 | 2031 | 日均值 |\n" +
		"\n## 后续说明\n\n" +
		"表格到此结束，这是普通段落，与表格无关。"

	// 较大 max 让表格段落整块落袋（不进硬切路径），块首自然携带表头；
	// "## 后续说明" 标题把普通段落切为独立块，验证其不补表头
	chunks := ChunkContent(content, 2000, 2000, ChunkMeta{})
	if len(chunks) < 2 {
		t.Fatalf("期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if strings.Contains(c, "2051") || strings.Contains(c, "2061") || strings.Contains(c, "2031") {
			if !strings.Contains(c, "命令编码") {
				t.Errorf("第 %d 块含表格数据行但缺少表头上下文：\n%s", i, c)
			}
		}
	}
	// 普通段落块不应被补表头
	last := chunks[len(chunks)-1]
	if strings.Contains(last, "命令编码") {
		t.Errorf("普通段落块不应携带表头：\n%s", last)
	}
}

// TestChunkHardSplitPreservesLines 验证硬切在行边界落刀：多行代码块超限时，
// 每行内部不被字符级切断（方案①核心：结构语义完整），Marker 整行完整保留
func TestChunkHardSplitPreservesLines(t *testing.T) {
	var b strings.Builder
	b.WriteString("```go\n")
	for i := 0; i < 30; i++ {
		b.WriteString("    customLine" + strconv.Itoa(i) + " := computeValue(); // " + strconv.Itoa(i) + "\n")
	}
	b.WriteString("```\n")
	// 小 max 强制整块走硬切；每行约 40 rune，总远超上限
	chunks := ChunkContent(b.String(), 200, 200, ChunkMeta{})
	if len(chunks) < 2 {
		t.Fatalf("期望至少 2 块，实际 %d 块", len(chunks))
	}
	// 校验每行 Marker 整行完整存在（未被字符级切断）
	joined := strings.Join(chunks, "\n")
	for i := 0; i < 30; i++ {
		marker := "customLine" + strconv.Itoa(i) + " := computeValue();"
		if !strings.Contains(joined, marker) {
			t.Errorf("硬切切断行 %d，Marker 不完整：%s", i, marker)
		}
	}
}
