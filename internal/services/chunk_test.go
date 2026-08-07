package services

import (
	"reflect"
	"strings"
	"testing"
)

// TestChunkLongChinese 验证 2000+ 字中文输入能分出至少 4 块，且每块不超过 maxRunes 个 rune
func TestChunkLongChinese(t *testing.T) {
	// 60 段 × 54 字 = 3240 字（段间用空行分隔）
	var b strings.Builder
	for i := 0; i < 60; i++ {
		b.WriteString(strings.Repeat("量化学习与向量检索", 6)) // 9 字 × 6 = 54 字
		b.WriteString("\n\n")
	}
	content := b.String()

	chunks := ChunkContent(content, 500)
	if len(chunks) < 4 {
		t.Fatalf("2000+ 字中文期望至少 4 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkHeadings 验证按 ## / ### 标题分块
func TestChunkHeadings(t *testing.T) {
	content := "## 第一章 简介\n这是第一章的内容，介绍向量检索的基本概念。\n\n### 1.1 原理\n本节讲解向量化的原理。\n\n## 第二章 应用\n这里讲述向量检索在实际场景中的应用。"
	chunks := ChunkContent(content, 500)

	if len(chunks) != 3 {
		t.Fatalf("期望按标题分出 3 块，实际 %d 块: %v", len(chunks), chunks)
	}
	if !strings.HasPrefix(chunks[0], "## 第一章") {
		t.Errorf("第 0 块应以 \"## 第一章\" 开头，实际: %q", chunks[0])
	}
	if !strings.HasPrefix(chunks[1], "### 1.1") {
		t.Errorf("第 1 块应以 \"### 1.1\" 开头，实际: %q", chunks[1])
	}
	if !strings.HasPrefix(chunks[2], "## 第二章") {
		t.Errorf("第 2 块应以 \"## 第二章\" 开头，实际: %q", chunks[2])
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
}

// TestChunkNoHeading 验证无标题时按空行分段
func TestChunkNoHeading(t *testing.T) {
	content := "第一段内容，没有标题。\n\n第二段内容，也没有标题。\n\n第三段内容，同样没有标题。"
	chunks := ChunkContent(content, 500)

	if len(chunks) != 3 {
		t.Fatalf("期望按空行分出 3 块，实际 %d 块: %v", len(chunks), chunks)
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
	chunks := ChunkContent(long, 500)

	if len(chunks) < 2 {
		t.Fatalf("超长单段期望至少 2 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块长度 %d 超过上限 500", i, runeLen(c))
		}
	}
	// 硬切不丢内容：拼接后应与原文一致
	if joined := strings.Join(chunks, ""); joined != long {
		t.Error("硬切后拼接结果与原文不一致")
	}
}

// TestChunkDefaultMaxRunes 验证 maxRunes<=0 时使用默认值 500
func TestChunkDefaultMaxRunes(t *testing.T) {
	content := strings.Repeat("中", 1200)
	chunks := ChunkContent(content, 0) // 应使用默认 500

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
