package tools

import (
	"strings"
	"testing"
)

// TestWhitespaceFold 验证空白折叠：\r\n→\n、连续空白→单空格、去首尾空白。
func TestWhitespaceFold(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空串", "", ""},
		{"纯空白", " \t\n ", ""},
		{"普通文本", "hello world", "hello world"},
		{"CRLF 统一", "a\r\nb", "a b"},
		{"连续空白折叠", "a   b\t\tc\n\nd", "a b c d"},
		{"首尾空白去除", "  hello  ", "hello"},
		{"多行缩进", "  # 标题\n    内容  ", "# 标题 内容"},
		{"中文", "你好  世界", "你好 世界"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := whitespaceFold(c.in); got != c.want {
				t.Errorf("whitespaceFold(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestFindNormalized 验证空白归一化匹配的原文偏移映射。
func TestFindNormalized(t *testing.T) {
	// 原文："line1\n  indented  text\nline3" —— 第二个片段带缩进与连续空格
	current := "line1\n  indented  text\nline3"
	cases := []struct {
		name    string
		find    string
		nth     int
		want    string // 期望命中的原文片段（current[start:end]）
		wantHit bool
	}{
		{"精确命中保持", "indented  text", 1, "indented  text", true},
		{"归一化折叠命中", "indented text", 1, "indented  text", true},
		{"换行差异命中", "line1\n  indented", 1, "line1\n  indented", true},
		{"CRLF 差异命中", "line1\r\nindented", 1, "line1\n  indented", true},
		{"未命中", "not-exist", 1, "", false},
		{"第 2 次", "e", 2, "e", true}, // line1 中的 e（第 2 个 e 在 "line" 里是第 2 个）
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, e := findNormalized(current, c.find, c.nth)
			if !c.wantHit {
				if s != -1 || e != -1 {
					t.Errorf("findNormalized(%q, nth=%d) = (%d,%d), want (-1,-1)", c.find, c.nth, s, e)
				}
				return
			}
			if s < 0 || e <= s {
				t.Fatalf("findNormalized(%q, nth=%d) = (%d,%d), want hit", c.find, c.nth, s, e)
			}
			got := current[s:e]
			if got != c.want {
				t.Errorf("findNormalized(%q, nth=%d) hit %q, want %q", c.find, c.nth, got, c.want)
			}
			// 映射必须落在原文边界内
			if s < 0 || e > len(current) {
				t.Errorf("偏移越界: (%d,%d) > len=%d", s, e, len(current))
			}
		})
	}

	// 第 n 次出现：验证归一化与精确语义一致（"a\nb a" 中找 "a"，第 2 次）
	t.Run("归一化第n次", func(t *testing.T) {
		cur := "alpha a\nbeta a"
		s, e := findNormalized(cur, "a", 2)
		if s < 0 {
			t.Fatal("应命中第 2 个 a")
		}
		if got := cur[s:e]; got != "a" {
			t.Errorf("命中 %q, want %q", got, "a")
		}
	})
}

// TestReplaceAllFragments 验证全部替换：精确 + 归一化兜底。
func TestReplaceAllFragments(t *testing.T) {
	cases := []struct {
		name    string
		current string
		find    string
		replace string
		want    string
		wantN   int
	}{
		{"精确全部替换", "a X a X a", "X", "Y", "a Y a Y a", 2},
		{"归一化全部替换", "a X\na X\na  X", "a X", "a Y", "a Y\na Y\na Y", 3},
		{"混合精确与归一化", "foo bar\nfoo  bar", "foo bar", "F", "F\nF", 2},
		{"未命中", "hello world", "nope", "X", "hello world", 0},
		{"find 为空", "abc", "", "X", "abc", 0},
		{"替换为空(删除)", "a\nb\na", "a", "", "\nb\n", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, n := replaceAllFragments(c.current, c.find, c.replace)
			if got != c.want || n != c.wantN {
				t.Errorf("replaceAllFragments(%q, %q, %q) = (%q, %d), want (%q, %d)",
					c.current, c.find, c.replace, got, n, c.want, c.wantN)
			}
		})
	}
}

// TestSplitNoteLines 验证按行拆分：末尾换行不产生幽灵空行。
func TestSplitNoteLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\nb", []string{"a", "b"}},
		{"a\nb\n", []string{"a", "b"}},
		{"a\n\nb", []string{"a", "", "b"}},
		{"a\r\nb\r\n", []string{"a\r", "b\r"}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := splitNoteLines(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("splitNoteLines(%q) = %#v, want %#v", c.in, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("splitNoteLines(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestReplaceLines 验证行级替换/删除与越界报错。
func TestReplaceLines(t *testing.T) {
	cases := []struct {
		name    string
		content string
		start   int
		end     int
		replace string
		want    string
		wantN   int
		wantErr bool
	}{
		{"替换单行", "a\nb\nc", 2, 2, "X", "a\nX\nc", 1, false},
		{"删除单行", "a\nb\nc", 2, 2, "", "a\nc", 1, false},
		{"替换多行", "a\nb\nc\nd", 2, 3, "X\nY", "a\nX\nY\nd", 2, false},
		{"删除多行", "a\nb\nc\nd", 2, 3, "", "a\nd", 2, false},
		{"替换首行", "a\nb", 1, 1, "X", "X\nb", 1, false},
		{"替换到末尾", "a\nb\nc", 2, 3, "X", "a\nX", 2, false},
		{"删除全部", "a\nb", 1, 2, "", "", 2, false},
		{"多行替换文本", "a\nb", 1, 1, "X\nY\nZ", "X\nY\nZ\nb", 1, false},
		{"CRLF 保留", "a\r\nb\r\nc", 2, 2, "X", "a\r\nX\nc", 1, false},
		{"start 越界", "a\nb", 3, 3, "X", "", 0, true},
		{"end 越界", "a\nb", 1, 3, "X", "", 0, true},
		{"start>end", "a\nb", 2, 1, "X", "", 0, true},
		{"start<1", "a\nb", 0, 1, "X", "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, n, total, err := replaceLines(c.content, c.start, c.end, c.replace)
			if c.wantErr {
				if err == nil {
					t.Errorf("replaceLines(%q, %d, %d, %q) 应报错，实际返回 %q", c.content, c.start, c.end, c.replace, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("replaceLines(%q, %d, %d, %q) 意外错误: %v", c.content, c.start, c.end, c.replace, err)
			}
			if got != c.want || n != c.wantN {
				t.Errorf("replaceLines(%q, %d, %d, %q) = (%q, %d, %d), want (%q, %d, %d)",
					c.content, c.start, c.end, c.replace, got, n, total, c.want, c.wantN, total)
			}
		})
	}
}

// TestNumberLines 验证行号前缀：1-based、可指定起始行号、\r 剔除。
func TestNumberLines(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		startLine int
		want      string
	}{
		{"空内容", "", 1, ""},
		{"基本行号", "a\nb\nc", 1, "行 1: a\n行 2: b\n行 3: c\n"},
		{"指定起始行号", "b\nc", 5, "行 5: b\n行 6: c\n"},
		{"CRLF 行号", "a\r\nb\r\n", 1, "行 1: a\n行 2: b\n"},
		{"空行保留", "a\n\nb", 1, "行 1: a\n行 2: \n行 3: b\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := numberLines(c.content, c.startLine); got != c.want {
				t.Errorf("numberLines(%q, %d) = %q, want %q", c.content, c.startLine, got, c.want)
			}
		})
	}
}

// TestIndexNth 回归：精确第 n 次出现语义保持。
func TestIndexNth(t *testing.T) {
	s := "a X a X a"
	cases := []struct {
		sub  string
		nth  int
		want int
	}{
		{"X", 1, 2},
		{"X", 2, 6},
		{"X", 3, -1},
		{"a", 1, 0},
		{"a", 2, 4},
		{"a", 3, 8},
	}
	for _, c := range cases {
		if got := indexNth(s, c.sub, c.nth); got != c.want {
			t.Errorf("indexNth(%q, %q, %d) = %d, want %d", s, c.sub, c.nth, got, c.want)
		}
	}
}

// TestEditNoteModeValidation 验证 editNote 的模式互斥拒绝分支（find+line 混用与
// 全空在触达 DB 前即返回，nil 服务安全）。
func TestEditNoteModeValidation(t *testing.T) {
	m := &manageNoteTool{}
	cases := []struct {
		name        string
		find        string
		lineStart   float64
		wantModeErr bool
	}{
		{"find+line 混用", "y", 2, true},
		{"全空", "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.editNote(1, c.find, "R", 0, false, c.lineStart, 0)
			if !c.wantModeErr {
				t.Fatal("该用例应报错")
			}
			if err == nil {
				t.Error("应返回模式互斥/参数错误")
			}
		})
	}
	// replace_all 与 count>1 冲突（片段模式专属校验）
	t.Run("replace_all 与 count>1 冲突", func(t *testing.T) {
		_, err := m.editNote(1, "x", "R", 2, true, 0, 0)
		if err == nil {
			t.Error("应返回 replace_all 与 count>1 互斥错误")
		}
	})
}

// TestExtractLastLineNum 验证从行号化文本提取最后一个行号。
func TestExtractLastLineNum(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"空串", "", 0},
		{"单行", "行 1: hello", 1},
		{"多行", "行 1: a\n行 2: b\n行 3: c", 3},
		{"无行号", "hello world", 0},
		{"大行号", "行 999: text", 999},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractLastLineNum(c.input)
			if got != c.want {
				t.Errorf("extractLastLineNum(%q) = %d, want %d", c.input, got, c.want)
			}
		})
	}
}

// TestLineEditPreview 验证行级替换的上下文预览。
func TestLineEditPreview(t *testing.T) {
	content := "第一行\n第二行\n第三行\n第四行\n第五行"
	t.Run("中间替换", func(t *testing.T) {
		preview := lineEditPreview(content, 3, 5) // 替换第 3 行
		if preview == "" {
			t.Error("预览不应为空")
		}
		if !strings.Contains(preview, "行 2:") || !strings.Contains(preview, "行 3:") || !strings.Contains(preview, "行 4:") {
			t.Errorf("预览应包含行 2-4 上下文，实际: %s", preview)
		}
	})
	t.Run("首行替换", func(t *testing.T) {
		preview := lineEditPreview(content, 1, 5)
		if preview == "" {
			t.Error("预览不应为空")
		}
		if !strings.Contains(preview, "行 1:") {
			t.Errorf("预览应包含行 1，实际: %s", preview)
		}
	})
	t.Run("空内容", func(t *testing.T) {
		preview := lineEditPreview("", 1, 0)
		if preview != "" {
			t.Errorf("空内容预览应为空，实际: %s", preview)
		}
	})
}

// TestFindMostSimilar 验证最相似片段查找。
func TestFindMostSimilar(t *testing.T) {
	content := "这是一段测试文字\n包含一些常见词汇\n第三行内容"
	t.Run("精确匹配附近", func(t *testing.T) {
		similar, line := findMostSimilar(content, "测试文字")
		if similar == "" {
			t.Fatal("应找到相似片段")
		}
		if !strings.Contains(similar, "测试文字") {
			t.Errorf("应包含'测试文字'，实际: %s", similar)
		}
		if line < 1 || line > 3 {
			t.Errorf("行号应在 1-3 范围内，实际: %d", line)
		}
	})
	t.Run("空输入", func(t *testing.T) {
		similar, _ := findMostSimilar("", "test")
		if similar != "" {
			t.Errorf("空内容应返回空，实际: %s", similar)
		}
	})
	t.Run("空查找串", func(t *testing.T) {
		similar, _ := findMostSimilar("content", "")
		if similar != "" {
			t.Errorf("空查找串应返回空，实际: %s", similar)
		}
	})
}

// TestRuneOverlap 验证字符重叠率计算。
func TestRuneOverlap(t *testing.T) {
	cases := []struct {
		name string
		a, b []rune
		min  float64
		max  float64
	}{
		{"完全相同", []rune("abc"), []rune("abc"), 1.0, 1.0},
		{"无重叠", []rune("abc"), []rune("xyz"), 0.0, 0.0},
		{"部分重叠", []rune("abcd"), []rune("bcde"), 0.7, 0.8},
		{"空切片", []rune(""), []rune("abc"), 0.0, 0.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			score := runeOverlap(c.a, c.b)
			if score < c.min || score > c.max {
				t.Errorf("runeOverlap(%q, %q) = %f, want [%f, %f]", c.a, c.b, score, c.min, c.max)
			}
		})
	}
}

// TestBuildNotFoundHint 验证未找到片段的提示信息格式。
func TestBuildNotFoundHint(t *testing.T) {
	t.Run("有相似片段", func(t *testing.T) {
		_, err := buildNotFoundHint(42, "测试文字xyz", "这是一段测试文字内容")
		if err == nil {
			t.Fatal("应返回错误")
		}
		msg := err.Error()
		if !strings.Contains(msg, "笔记 #42") {
			t.Error("应包含笔记编号")
		}
		if !strings.Contains(msg, "测试文字xyz") {
			t.Error("应包含查找的片段")
		}
		if !strings.Contains(msg, "最接近") {
			t.Error("应包含最接近的片段提示")
		}
	})
	t.Run("无相似片段", func(t *testing.T) {
		_, err := buildNotFoundHint(1, "xyz", "hello world")
		if err == nil {
			t.Fatal("应返回错误")
		}
		msg := err.Error()
		if !strings.Contains(msg, "笔记 #1") {
			t.Error("应包含笔记编号")
		}
		// 无重叠时提示中不含「最接近」但仍应包含重试建议
		if !strings.Contains(msg, "调用 view") {
			t.Error("应包含重试建议")
		}
	})
}
