package tools

// 本文件实现 delete_item 工具：删除 AI 助手工作目录（~/.jot/workspace）内的文件
// 或目录（类似 rm），复用 os.Root 的 Remove / RemoveAll（基于目录句柄，杜绝
// ../ 逃逸与符号链接逃逸）。经 fsToolBase 做路径边界校验（resolvePath 第一道
// 防线 + os.Root 第二道防线），并额外拒绝删除工作区根目录本身
// （os.RemoveAll(root) 会清空一切，属灾难性操作）。
// 删除不可恢复，因此始终经 Context.Approver 审批钩子接管，且 critical=true
// （不可绕过的强制确认，对齐 run_command 的 rm/del 高危判定，任何审批模式都必须
// 弹窗确认）。目录递归删除必须显式传 recursive=true，缺省只允许删除文件或空目录，
// 防止模型误删整个目录。纯文件操作不读内容，无需二进制检测。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// deleteItemTool 删除工作目录内文件/目录的工具。
type deleteItemTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*deleteItemTool)(nil)
var _ ActionTextProvider = (*deleteItemTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *deleteItemTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "删除项"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "删除项：" + TruncateRunes(p, 30)
	}
	return "删除项"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *deleteItemTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "delete_item",
		Desc: "删除工作目录内的文件或目录（类似 rm）。path 为文件时直接删除；path 为目录时必须显式传 recursive=true 才递归删除（缺省只允许删除文件或空目录）。删除不可恢复，执行前需强制审批确认（不可绕过）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要删除的文件/目录路径；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可",
				Required: true,
			},
			"recursive": {
				Type:     schema.Boolean,
				Desc:     "path 为目录时是否递归删除其全部内容，缺省 false（目录非空时报错）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行删除：参数校验 → 边界校验 → 拒绝删工作区根 → 存在性检查 →
// 目录递归判定 → 强制审批（critical）→ os.Remove/os.RemoveAll。
func (t *deleteItemTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 delete_item 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("delete_item 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}

	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	// 防误删：拒绝删除工作区根目录本身（清空一切属灾难性操作，参数错误无需审批）；
	// 大小写不敏感比较（Windows 文件系统大小写不敏感，resolvePath 会放行大小写变体）
	root, err := t.wsRoot()
	if err != nil {
		return "", err
	}
	if pathEqualsFold(fullPath, root) {
		return "", errors.New("delete_item 拒绝删除工作区根目录，如需清空请逐个删除内部文件/目录")
	}

	// 打开 os.Root 目录句柄（第二道防逃逸），后续存在性检查/删除均走 Root 相对路径
	h, rel, err := t.openRootFor(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开工作目录失败: %w", err)
	}
	defer func() { _ = h.Close() }()

	// 存在性检查：删除对象不存在是参数错误，无需审批
	stat, err := h.Lstat(rel)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("delete_item 目标文件/目录不存在")
		}
		return "", fmt.Errorf("检查目标文件失败: %w", err)
	}

	// 目录递归判定：非空目录必须显式 recursive=true 才允许递归删除；
	// 空目录无内容丢失风险，缺省直接允许删除（与"缺省只允许删文件/空目录"一致）
	if stat.IsDir() && !args.Recursive {
		// os.Root 无 ReadDir 方法，经 root.FS()（io/fs 适配器）列举。
		// 注意：适配器仅接受 "/" 分隔路径，filepath.Rel 在 Windows 上返回反斜杠
		// 分隔的多层 rel（如 a\b），直接传入会报 invalid argument，须 ToSlash
		// 统一分隔符（与 ls_dir 同根因，见 tools_log 50 次路径失败分析）。
		entries, err := fs.ReadDir(h.FS(), filepath.ToSlash(rel))
		if err != nil {
			return "", fmt.Errorf("检查目录内容失败: %w", err)
		}
		if len(entries) > 0 {
			return "", errors.New("delete_item 目标是非空目录，未设置 recursive=true 时拒绝删除（目录递归删除需显式确认）")
		}
	}

	// 强制审批检查点：删除不可恢复，critical=true（任何审批模式都必须确认）；
	// 未被拒绝（Approver 未注入/批准）则继续执行
	if err := t.requestApproval(ctx, "delete_item", "删除项："+path, true); err != nil {
		return "", err
	}

	if args.Recursive {
		err = h.RemoveAll(rel)
	} else {
		err = h.Remove(rel)
	}
	if err != nil {
		return "", fmt.Errorf("删除失败: %w", err)
	}
	return "已删除：" + path, nil
}

// NewDeleteItem 创建 delete_item 工具。
func NewDeleteItem(ctx *Context) tool.InvokableTool {
	return &deleteItemTool{fsToolBase: fsToolBase{ctx: ctx}}
}
