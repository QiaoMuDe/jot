package chunk

import "testing"

// firstRunes 取字符串前 n 个 rune（rune 安全，避免字节切片切断多字节字符）。
func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// TestChunkText 构造 2000+ 字且含多个 ## 标题的文本，验证切块数量与单块上限。
func TestChunkText(t *testing.T) {
	// 构造长段落（约 1300 字）
	longPara := "这是用于测试切块的长段落内容。向量检索系统需要先把长文档切成小块再生成向量。"
	for len([]rune(longPara)) < 1300 {
		longPara += "这里补充一些具体的技术细节：切块粒度影响召回质量，块太大语义混杂，块太小上下文不足。"
	}

	content := "## 引言\n\n" + longPara + "\n\n## 架构设计\n\n" + firstRunes(longPara, 600) + "\n\n" +
		"## 检索流程\n\n" + firstRunes(longPara, 500) + "\n\n## 总结\n\n" + "本文档用于验证切块逻辑的正确性与上限约束。"

	// 基本输入保证
	if len([]rune(content)) < 2000 {
		t.Fatalf("测试文本长度不足 2000 字，实际 %d 字", len([]rune(content)))
	}

	chunks := ChunkText(content, 500)
	if len(chunks) < 4 {
		t.Fatalf("期望至少切出 4 块，实际 %d 块", len(chunks))
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块超过 500 字上限，实际 %d 字", i, runeLen(c))
		}
		if c == "" {
			t.Errorf("第 %d 块为空块", i)
		}
	}
}

// TestChunkDefault 验证便捷封装使用默认 500 字上限。
func TestChunkDefault(t *testing.T) {
	content := "## 标题\n\n内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容"
	chunks := ChunkDefault(content)
	if len(chunks) == 0 {
		t.Fatal("ChunkDefault 不应返回空结果")
	}
	for i, c := range chunks {
		if runeLen(c) > 500 {
			t.Errorf("第 %d 块超过默认 500 字上限，实际 %d 字", i, runeLen(c))
		}
	}
}
