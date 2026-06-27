package services

import "testing"

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
