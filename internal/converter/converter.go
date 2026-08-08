// Package converter 封装 markitdown 库，用于将办公文件转换为 Markdown 文本。
package converter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	markitdownlib "github.com/conductor-oss/markitdown"
)

var (
	// ErrUnsupportedFormat 表示 markitdown 无法处理该文件格式。
	ErrUnsupportedFormat = errors.New("不支持的文件格式")

	// ErrConversionTimeout 表示文件转换超过 60 秒超时限制。
	ErrConversionTimeout = errors.New("文件转换超时")
)

// markitdown 支持的二进制办公文件扩展名。
// 纯文本文件（.txt/.md/.html/.csv/.json 等）不走此转换器，通过二进制检测兜底。
var officeExtensions = map[string]bool{
	".docx": true,
	".xlsx": true,
	".xls":  true,
	".pptx": true,
	".pdf":  true,
	".epub": true,
	".zip":  true,
}

// markitdown 实例（复用）
var md = markitdownlib.New()

// IsOfficeFile 检查文件扩展名是否为支持的办公文件类型。
func IsOfficeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return officeExtensions[ext]
}

// ConvertToMarkdown 将办公文件转换为 Markdown 文本，带 60 秒超时保护。
func ConvertToMarkdown(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	type result struct {
		markdown string
		err      error
	}

	ch := make(chan result, 1)

	go func() {
		// 第三方解析库对畸形文件可能 panic，若不拦截会直接拖垮整个进程。
		// 这里把 panic 转成错误返回，保证导入失败只是提示错误而不是闪退。
		defer func() {
			if r := recover(); r != nil {
				ch <- result{err: fmt.Errorf("文件内容解析失败，该文件可能已损坏或不是有效的办公文档（%v）", r)}
			}
		}()

		f, err := os.Open(path)
		if err != nil {
			ch <- result{err: fmt.Errorf("打开文件失败: %w", err)}
			return
		}
		defer func() { _ = f.Close() }()

		ext := strings.ToLower(filepath.Ext(path))
		info := markitdownlib.StreamInfo{
			Extension: ext,
			Filename:  filepath.Base(path),
			LocalPath: path,
		}

		r, err := md.ConvertReader(f, info)
		if err != nil {
			if markitdownlib.IsUnsupportedFormat(err) {
				ch <- result{err: ErrUnsupportedFormat}
				return
			}
			ch <- result{err: fmt.Errorf("转换失败: %w", err)}
			return
		}
		ch <- result{markdown: r.Markdown}
	}()

	select {
	case r := <-ch:
		return r.markdown, r.err
	case <-ctx.Done():
		return "", ErrConversionTimeout
	}
}
