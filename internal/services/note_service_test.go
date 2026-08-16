package services

import (
	"strings"
	"testing"
)

// TestBuildSortOrder 验证搜索/列表的 ORDER BY 子句构造逻辑
// 覆盖 3 个 sortBy 值 + 默认值 + 非法值
func TestBuildSortOrder(t *testing.T) {
	cases := []struct {
		name     string
		sortBy   string
		expected string
	}{
		{"updated_at 默认", "updated_at", "pinned DESC, updated_at DESC"},
		{"created_at", "created_at", "pinned DESC, created_at DESC"},
		{"title 升序", "title", "pinned DESC, title ASC"},
		{"空字符串回退默认", "", "pinned DESC, updated_at DESC"},
		{"非法值回退默认", "xxx_invalid", "pinned DESC, updated_at DESC"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildSortOrder(c.sortBy)
			if got != c.expected {
				t.Errorf("buildSortOrder(%q) = %q, want %q", c.sortBy, got, c.expected)
			}
		})
	}
}

// TestNextDuplicateTitle 验证副本标题生成：后缀、冲突递增序号、标题截断、空标题回退。
func TestNextDuplicateTitle(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		existing map[string]bool
		want     string
	}{
		{"首副本", "会议记录", nil, "会议记录 副本"},
		{"副本冲突递增到2", "会议记录", map[string]bool{"会议记录 副本": true}, "会议记录 副本 2"},
		{"副本2也冲突递增到3", "会议记录", map[string]bool{"会议记录 副本": true, "会议记录 副本 2": true}, "会议记录 副本 3"},
		{"首尾空格去除", "  会议记录  ", nil, "会议记录 副本"},
		{"空标题回退未命名", "   ", nil, "未命名 副本"},
		{"空标题冲突递增", "", map[string]bool{"未命名 副本": true}, "未命名 副本 2"},
		{"超长标题截断", strings.Repeat("长", 200), nil, strings.Repeat("长", 197) + " 副本"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exists := func(title string) bool { return c.existing[title] }
			got := nextDuplicateTitle(c.base, exists)
			if got != c.want {
				t.Errorf("nextDuplicateTitle(%q) = %q, want %q", c.base, got, c.want)
			}
			if len([]rune(got)) > noteTitleMaxRunes {
				t.Errorf("标题超长: %d runes > %d", len([]rune(got)), noteTitleMaxRunes)
			}
		})
	}
}

// TestTruncateTitleRunes 验证标题 rune 截断边界。
func TestTruncateTitleRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"未超限原样返回", "hello", 10, "hello"},
		{"恰好等于上限", "hello", 5, "hello"},
		{"多字节截断", "你好世界", 3, "你好世"},
		{"空串", "", 5, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := truncateTitleRunes(c.in, c.max); got != c.want {
				t.Errorf("truncateTitleRunes(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
			}
		})
	}
}
