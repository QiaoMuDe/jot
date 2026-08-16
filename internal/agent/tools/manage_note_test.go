package tools

import (
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

// TestEditNoteModeValidation 验证 editNote 的模式互斥拒绝分支（modes>1 与 modes==0
// 在触达 DB 前即返回，nil 服务安全）。
func TestEditNoteModeValidation(t *testing.T) {
	m := &manageNoteTool{}
	cases := []struct {
		name        string
		content     string
		find        string
		appendC     string
		lineStart   float64
		wantModeErr bool
	}{
		{"content+find 混用", "x", "y", "", 0, true},
		{"find+line 混用", "", "y", "", 2, true},
		{"content+append 混用", "x", "", "z", 0, true},
		{"append+line 混用", "", "", "z", 2, true},
		{"全空", "", "", "", 0, true},
		{"replace_all 与 count>1 冲突", "", "x", "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var err error
			if c.name == "replace_all 与 count>1 冲突" {
				_, err = m.editNote(1, c.content, c.find, "R", c.appendC, 2, true, c.lineStart, 0)
			} else {
				_, err = m.editNote(1, c.content, c.find, "R", c.appendC, 0, false, c.lineStart, 0)
			}
			if !c.wantModeErr {
				t.Fatal("该用例应报错")
			}
			if err == nil {
				t.Error("应返回模式互斥/参数错误")
			}
		})
	}
}
