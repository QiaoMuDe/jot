package tools

// 本文件实现 move_item 工具：移动 AI 助手工作目录（~/.jot/workspace）内的文件
// 或目录到目标位置（类似 mv），复用 go-kit fs 的 MoveEx（优先 os.Rename 原子
// 操作，失败降级为复制+删除，支持跨文件系统）。经 fsToolBase 做路径边界校验
// （source 与 dest 都必须在工作目录内，越权一律拒绝）。
// 移动必然移除源，属结构性变更，因此始终经 Context.Approver 审批钩子接管
// （critical=false，常规审批可取消），无论是否覆盖；覆盖已存在目标时缺省拒绝
// （报错引导设 overwrite=true），overwrite=true 才允许。
// dest 为已存在目录时自动追加源文件名（go-kit 智能路径语义）；移动目录时递归
// 移动全部内容。纯文件操作不读内容，无需二进制检测。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	kitfs "gitee.com/MM-Q/go-kit/fs"
)

// moveItemTool 移动工作目录内文件/目录的工具。
type moveItemTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*moveItemTool)(nil)
var _ ActionTextProvider = (*moveItemTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *moveItemTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "移动项"
	}
	if s := strings.TrimSpace(args.Source); s != "" {
		return "移动项：" + TruncateRunes(s, 30)
	}
	return "移动项"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *moveItemTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "move_item",
		Desc: "在工作目录内移动文件或目录到目标位置（类似 mv）。source 为源路径、dest 为目标路径；dest 为已存在目录时自动追加源文件名（如源 a.txt 移动到已存在的 dir/ 即变为 dir/a.txt）。移动会移除源文件/目录，执行前需审批（常规审批可取消）。目标已存在时缺省拒绝，overwrite=true 才允许覆盖。移动目录时递归移动全部内容。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"source": {
				Type:     schema.String,
				Desc:     "源文件/目录路径；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可",
				Required: true,
			},
			"dest": {
				Type:     schema.String,
				Desc:     "目标路径；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可；为已存在目录时自动追加源文件名",
				Required: true,
			},
			"overwrite": {
				Type:     schema.Boolean,
				Desc:     "是否覆盖已存在目标，缺省 false（目标已存在时报错）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行移动：参数校验 → 边界校验 → 源存在性检查 → 覆盖判定 →
// 审批检查 → go-kit MoveEx 执行（智能路径：dest 为已存在目录时自动追加源文件名）。
func (t *moveItemTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Source    string `json:"source"`
		Dest      string `json:"dest"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 move_item 参数失败: %w", err)
	}
	src := strings.TrimSpace(args.Source)
	if src == "" {
		return "", errors.New("move_item 参数缺少 source")
	}
	dst := strings.TrimSpace(args.Dest)
	if dst == "" {
		return "", errors.New("move_item 参数缺少 dest")
	}
	if err := validateTextLen("source", src, maxToolShortText); err != nil {
		return "", err
	}
	if err := validateTextLen("dest", dst, maxToolShortText); err != nil {
		return "", err
	}

	srcFull, err := t.resolvePath(src)
	if err != nil {
		return "", err
	}
	dstFull, err := t.resolvePath(dst)
	if err != nil {
		return "", err
	}

	// 源存在性检查：移动对象不存在是参数错误，无需审批
	if _, err := os.Lstat(srcFull); err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("move_item 源文件/目录不存在")
		}
		return "", fmt.Errorf("检查源文件失败: %w", err)
	}

	// 覆盖判定：dest 为已存在目录时按 go-kit 智能路径追加源文件名后再判目标存在性；
	// 目标已存在且未显式 overwrite=true 时缺省拒绝（安全默认）
	destExists, finalDst, err := fsFinalTarget(dstFull, srcFull)
	if err != nil {
		return "", err
	}
	// 源-目标关系校验：源与目标相同、或目录移动到自身子目录（MoveEx 虽有兜底但
	// 大小写敏感，Windows 大小写变体路径可绕过），前置拦截（参数错误语义，不触发审批）
	if err := checkSrcDestRelation(srcFull, finalDst); err != nil {
		return "", fmt.Errorf("move_item %w", err)
	}
	if destExists && !args.Overwrite {
		return "", errors.New("move_item 目标已存在，未设置 overwrite=true 时拒绝覆盖（覆盖请显式传 overwrite=true）")
	}

	// 审批检查点：移动必然移除源文件/目录，属结构性变更，始终需取得用户批准；
	// 未被拒绝（Approver 未注入/批准）则继续执行
	if err := t.requestApproval(ctx, "move_item", "移动项："+src+" → "+dst, false); err != nil {
		return "", err
	}

	if err := kitfs.MoveEx(srcFull, dstFull, args.Overwrite); err != nil {
		return "", fmt.Errorf("移动失败: %w", err)
	}
	return "已移动：" + src + " → " + t.relDisplayPath(finalDst), nil
}

// NewMoveItem 创建 move_item 工具。
func NewMoveItem(ctx *Context) tool.InvokableTool {
	return &moveItemTool{fsToolBase: fsToolBase{ctx: ctx}}
}
