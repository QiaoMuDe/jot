package tools

// 本文件实现 ls_dir 工具：像 ls 一样只列出工作目录（~/.jot/workspace）内某
// 目录的**直接子项**（单层、不含递归）。要查看更深的内容，模型以该子目录作为
// 新的 path 再次调用即可，逐步钻取。经 fsToolBase 做路径边界校验并经 os.Root
// 目录句柄列举（第二道防逃逸），输出按 rune 上限有界缓冲（目录下条目极多时即
// 在写入前提前截断），避免刷屏或内存放大。detail=true 时每行额外附大小与修改
// 时间（默认仅列名称，保持简洁并节省有界输出的 token 预算）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// maxLsDirRunes ls_dir 累计输出 rune 上限：写入逼近该值（剩余不足
// lsDirHeadroomRunes）即停止收集，避免单个超大目录产生过量输出。
const maxLsDirRunes = 20000

// lsDirHeadroomRunes ls_dir 提前停止收集的剩余阈值。
const lsDirHeadroomRunes = 2000

// lsDirTimeFormat 详情模式下修改时间的显示格式（本地时间）。
const lsDirTimeFormat = "2006-01-02 15:04:05"

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
		Desc: "像 ls 一样列出一个目录的直接内容（单层，目录与文件各一行）；不带 path 时列出工作目录根，要钻取更深请以目标子目录作为 path 再次调用。detail=true 时每行额外附大小与修改时间。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要列出的目录路径，缺省为工作目录根；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可",
				Required: false,
			},
			"detail": {
				Type:     schema.Boolean,
				Desc:     "是否输出详情（每行附大小与修改时间），缺省 false 仅列名称",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行目录列举：边界校验 → os.ReadDir 读取直接子项 → 按
// "目录: 名称" / "文件: 名称" 逐行输出，detail=true 时每行附"大小: x / 修改:
// 时间"（目录大小无意义显示 -，Info 失败降级为 ?）；超长截断。只列当前层。
func (t *lsDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path   string `json:"path"`
		Detail bool   `json:"detail"`
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

	// os.Root 无 ReadDir 方法，经 root.FS()（io/fs 适配器）列举，返回 []fs.DirEntry。
	// 注意：适配器仅接受 "/" 分隔路径，filepath.Rel 在 Windows 上返回反斜杠分隔的
	// 多层 rel（如 a\b），直接传入会报 invalid argument，须 ToSlash 统一分隔符。
	entries, err := fs.ReadDir(root.FS(), filepath.ToSlash(rel))
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("列出目录失败: %w", err)
	}

	var b strings.Builder
	cumRunes := 0
	// 首行输出当前所在目录的相对路径（如 "当前目录: a/b"），为模型提供路径锚点，
	// 便于正确拼接后续相对路径（替代 pwd 诉求）；根目录显示 "."（filepath.Rel 惯例）。
	if rel := t.relDisplayPath(fullPath); rel != "" {
		header := "当前目录: " + rel
		b.WriteString(header)
		cumRunes = len([]rune(header))
	}
	for _, d := range entries { // os.ReadDir 已按文件名升序
		typeName := "文件"
		if d.IsDir() {
			typeName = "目录"
		}
		line := typeName + ": " + d.Name()
		if args.Detail {
			// 大小/修改时间取自 d.Info()（纯标准库）；目录大小无意义显示 "-"，
			// 个别条目 Info 失败（如权限）降级为 "?"，不中断整个列举。
			sizeStr, modStr := "-", "?"
			if info, err := d.Info(); err == nil {
				if !d.IsDir() {
					sizeStr = formatSize(info.Size())
				}
				modStr = info.ModTime().Format(lsDirTimeFormat)
			}
			line += "  大小: " + sizeStr + "  修改: " + modStr
		}
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
