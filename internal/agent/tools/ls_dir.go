package tools

// 本文件实现 ls_dir 工具：像 ls 一样只列出工作目录（~/.jot/workspace）内某
// 目录的**直接子项**（单层、不含递归）。要查看更深的内容，模型以该子目录作为
// 新的 path 再次调用即可，逐步钻取。经 fsToolBase 做路径边界校验并经 os.Root
// 目录句柄列举（第二道防逃逸），输出按 rune 上限有界缓冲（目录下条目极多时即
// 在写入前提前截断），避免刷屏或内存放大。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// maxLsDirRunes ls_dir 累计输出 rune 上限：写入逼近该值（剩余不足
// lsDirHeadroomRunes）即停止收集，避免单个超大目录产生过量输出。
const maxLsDirRunes = 20000

// lsDirHeadroomRunes ls_dir 提前停止收集的剩余阈值。
const lsDirHeadroomRunes = 2000

// lsDirTool 列出工作目录内某目录直接子项的工具。
type lsDirTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*lsDirTool)(nil)
var _ ActionTextProvider = (*lsDirTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *lsDirTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "列出目录"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "列出目录：" + TruncateRunes(p, 30)
	}
	return "列出目录"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *lsDirTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ls_dir",
		Desc: "像 ls 一样列出一个目录的直接内容（单层，目录与文件各一行）；不带 path 时列出工作目录根，要钻取更深请以目标子目录作为 path 再次调用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要列出的目录路径，缺省为工作目录根；相对 ~/.jot/workspace 或为其内绝对路径",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行目录列举：边界校验 → os.ReadDir 读取直接子项 → 按
// "目录: 名称" / "文件: 名称" 逐行输出，超长截断。只列当前层，不递归。
func (t *lsDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 ls_dir 参数失败: %w", err)
	}
	target := strings.TrimSpace(args.Path)
	fullPath, err := t.resolvePath(target)
	if err != nil {
		return "", err
	}
	root, rel, err := t.openRootFor(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开工作目录失败: %w", err)
	}
	defer func() { _ = root.Close() }()

	// 目标类型判定：只允许列目录；文件/其他类型给出明确提示（fs.ReadDir 对文件
	// 报 ENOTDIR，统一文案含糊）
	st, err := root.Stat(rel)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("列出目录失败: %w", err)
	}
	if !st.IsDir() {
		return "", errors.New("ls_dir 目标不是目录（不支持列出文件），请传入目录路径")
	}

	// os.Root 无 ReadDir 方法，经 root.FS()（io/fs 适配器）列举，返回 []fs.DirEntry
	entries, err := fs.ReadDir(root.FS(), rel)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("列出目录失败: %w", err)
	}

	var b strings.Builder
	cumRunes := 0
	for _, d := range entries { // os.ReadDir 已按文件名升序
		typeName := "文件"
		if d.IsDir() {
			typeName = "目录"
		}
		line := typeName + ": " + d.Name()
		if remaining := maxLsDirRunes - cumRunes; remaining < lsDirHeadroomRunes {
			return b.String() + "\n[目录内容过多，已提前停止列举]", nil
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
		cumRunes += len([]rune(line))
	}
	return b.String(), nil
}

// NewLsDir 创建列出目录工具。
func NewLsDir(ctx *Context) tool.InvokableTool {
	return &lsDirTool{fsToolBase: fsToolBase{ctx: ctx}}
}
