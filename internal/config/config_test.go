package config

import (
	"path/filepath"
	"testing"
)

// TestSubDir 验证 SubDir 返回 ~/.jot/<sub> 路径。
func TestSubDir(t *testing.T) {
	root, err := JotHomeDir()
	if err != nil {
		t.Fatalf("JotHomeDir() 意外错误: %v", err)
	}
	got, err := SubDir(DirMCP)
	if err != nil {
		t.Fatalf("SubDir(%q) 意外错误: %v", DirMCP, err)
	}
	want := filepath.Join(root, DirMCP)
	if got != want {
		t.Errorf("SubDir(%q) = %q, want %q", DirMCP, got, want)
	}
}
