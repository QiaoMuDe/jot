// Package config 提供 Jot 应用在用户家目录下的统一根目录（~/.jot）路径解析。
// 所有读写 ~/.jot 下文件的模块都应通过本包获取路径，避免硬编码散落各处。
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ~/.jot 下子目录名常量
const (
	DirData   = "data"   // 数据库目录
	DirBackup = "backup" // 备份目录
	DirImages = "images" // 图片目录
	DirLogs   = "logs"   // 日志目录
	DirMCP    = "mcp"    // MCP 配置目录
)

// JotHomeDir 返回应用根目录: ~/.jot
func JotHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户家目录失败: %w", err)
	}
	return filepath.Join(home, ".jot"), nil
}

// SubDir 返回应用根目录下的子目录路径，如 SubDir(DirMCP) -> ~/.jot/mcp
func SubDir(sub string) (string, error) {
	root, err := JotHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, sub), nil
}
