package tools

// 本文件实现 mkdir_dir 工具：在工作目录（~/.jot/workspace）内创建目录。
// 默认仅创建单级目录（父目录必须已存在，os.Root.Mkdir；对齐 ls_dir/glob/
// delete_item 的默认单层语义），recursive=true 时才递归创建完整层级
// （类似 mkdir -p，os.Root.MkdirAll 幂等：已存在目录直接成功）。经 fsToolBase
// 做路径边界校验（resolvePath 第一道防线 + os.Root 第二道防线，基于目录句柄，
// 杜绝 ../ 逃逸与符号链接逃逸）。
// 建目录是纯增量非破坏性操作，不触发审批（对齐 write_file 新建文件不审批）；
// 与 write_file/copy_item/move_item 的隐式自动建父目录互补，提供显式的目录
// 结构规划入口。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// mkdirDirTool 在工作目录内创建目录的工具。
type mkdirDirTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*mkdirDirTool)(nil)
var _ ActionTextProvider = (*mkdirDirTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *mkdirDirTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "创建目录"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "创建目录：" + TruncateRunes(p, 30)
	}
	return "创建目录"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *mkdirDirTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "mkdir_dir",
		Desc: "在工作目录内创建目录：默认仅创建单级目录（父目录必须已存在，已存在的目录直接成功，幂等不报错）；recursive=true 时递归创建完整层级（类似 mkdir -p，path 为 a/b/c 时一次建完）。纯建目录用本工具；写文件到新目录时 write_file 也会自动创建父目录，无需先建目录。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要创建的目录路径；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可",
				Required: true,
			},
			"recursive": {
				Type:     schema.Boolean,
				Desc:     "是否递归创建完整层级（类似 mkdir -p），缺省 false：默认仅创建单级目录，父目录必须已存在",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行建目录：参数校验 → 边界校验 → 幂等判定（已存在/非目录）→
// 分派（默认单级 os.Root.Mkdir / recursive os.Root.MkdirAll）。
func (t *mkdirDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 mkdir_dir 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("mkdir_dir 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}

	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	// 打开 os.Root 目录句柄（第二道防逃逸），创建走 Root 相对路径
	h, rel, err := t.openRootFor(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开工作目录失败: %w", err)
	}
	defer func() { _ = h.Close() }()

	// 幂等判定：目标已存在且是目录 → 直接成功；已存在但不是目录 → 参数错误
	stat, statErr := h.Lstat(rel)
	if statErr == nil {
		if stat.IsDir() {
			return "目录已存在：" + path, nil
		}
		return "", errors.New("mkdir_dir 目标已存在且不是目录")
	}
	if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("检查目标路径失败: %w", statErr)
	}

	// 默认仅创建单级目录（父目录必须已存在）；recursive=true 时递归创建完整层级
	if args.Recursive {
		if err := h.MkdirAll(rel, 0o755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}
		return "已递归创建目录：" + path, nil
	}
	if err := h.Mkdir(rel, 0o755); err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("mkdir_dir 父目录不存在：默认仅创建单级目录，请先创建父目录或传 recursive=true")
		}
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	return "已创建目录：" + path, nil
}

// NewMkdirDir 创建 mkdir_dir 工具。
func NewMkdirDir(ctx *Context) tool.InvokableTool {
	return &mkdirDirTool{fsToolBase: fsToolBase{ctx: ctx}}
}
