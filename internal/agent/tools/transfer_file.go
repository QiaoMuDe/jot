package tools

// 本文件实现 transfer_file 工具：在 AI 助手工作目录（~/.jot/workspace）与用户
// 桌面（~/Desktop）之间复制文件或目录，是唯一允许触碰工作目录之外的文件工具。
// direction=download 把工作目录内的 path 对应内容落到桌面根下同名位置；
// direction=upload 反向（桌面 → 工作目录）。两端均经 SandboxFilePath 做
// Clean + 符号链接解析 + 边界校验（防 ../ 逃逸与 symlink/junction 逃逸），
// 工具无法借道读写两端之外的任何位置。
// 复制复用 go-kit fs 的 CopyEx（原子性：临时文件 + os.Rename；覆盖时先备份
// 原文件，失败自动恢复；目录递归复制全部内容）。目标端为已存在目录时自动追加
// 源基名（fsFinalTarget，对齐 copy_file 智能路径语义）。
// 审批分级（TOOLS.md §6.1 判定标准：外部副作用 = critical）：
//   - download（写用户桌面 = 工作目录之外的外部副作用）一律审批，critical=true：
//     confirm_every/review 模式强制确认，auto 模式自动放行并留 tool_auto_approval
//     审计痕；覆盖已存在文件时摘要附「（覆盖）」标记。
//   - upload（写工作区）对齐 copy_file：纯新增免审批；覆盖已存在目标时审批
//     critical=false（常规审批可取消）。
// 跨端复制源/目标恒在不同子树，无自递归风险，不调用 checkSrcDestRelation。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	kitfs "gitee.com/MM-Q/go-kit/fs"
)

// transfer_file 两端沙箱边界标识（报错文案用）
const (
	transferLabelWorkspace = "~/.jot/workspace"
	transferLabelDesktop   = "~/Desktop"
)

// transferFileTool 工作目录与用户桌面之间传输文件/目录的工具。
type transferFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*transferFileTool)(nil)
var _ ActionTextProvider = (*transferFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *transferFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Direction string `json:"direction"`
		Path      string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "传输文件"
	}
	p := strings.TrimSpace(args.Path)
	switch strings.TrimSpace(args.Direction) {
	case "download":
		if p != "" {
			return "下载文件：" + TruncateRunes(p, 30)
		}
		return "下载文件"
	case "upload":
		if p != "" {
			return "上传文件：" + TruncateRunes(p, 30)
		}
		return "上传文件"
	}
	return "传输文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *transferFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "transfer_file",
		Desc: "在 AI 助手工作目录（~/.jot/workspace）与用户桌面（~/Desktop）之间复制文件或目录：direction=download 把工作目录内的内容下载到桌面，direction=upload 把桌面的内容上传到工作目录。path 为相对源端的路径（download 相对 ~/.jot/workspace，upload 相对 ~/Desktop；也支持以 ~ 开头的路径或源端内绝对路径）。目标端为已存在目录时自动追加源文件名；目标已存在时缺省拒绝，overwrite=true 才允许覆盖（触发审批）。下载到桌面属于工作目录之外的写入，始终需要审批确认。仅能操作这两个目录，无法访问其它位置。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"direction": {
				Type:     schema.String,
				Desc:     "传输方向：download=工作目录→桌面（下载），upload=桌面→工作目录（上传）",
				Enum:     []string{"download", "upload"},
				Required: true,
			},
			"path": {
				Type:     schema.String,
				Desc:     "传输对象路径；相对源端根目录的路径、~ 开头路径或源端内绝对路径均可（download 源端为 ~/.jot/workspace，upload 源端为 ~/Desktop）",
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

// InvokableRun 执行传输：参数校验 → 双端路径解析与边界校验 → 源存在性检查 →
// 覆盖判定与审批 → go-kit CopyEx 执行（目录递归、原子性）。
func (t *transferFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Direction string `json:"direction"`
		Path      string `json:"path"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 transfer_file 参数失败: %w", err)
	}
	direction := strings.TrimSpace(args.Direction)
	if direction != "download" && direction != "upload" {
		return "", errors.New("transfer_file 参数 direction 非法，仅支持 download（工作目录→桌面）或 upload（桌面→工作目录）")
	}
	p := strings.TrimSpace(args.Path)
	if p == "" {
		return "", errors.New("transfer_file 参数缺少 path")
	}
	if err := validateTextLen("path", p, maxToolShortText); err != nil {
		return "", err
	}

	// 双端根：download 工作目录→桌面，upload 桌面→工作目录
	wsRoot, err := t.wsRoot()
	if err != nil {
		return "", err
	}
	deskRoot, err := t.deskRoot()
	if err != nil {
		return "", err
	}
	srcRoot, dstRoot := wsRoot, deskRoot
	if direction == "upload" {
		srcRoot, dstRoot = deskRoot, wsRoot
	}

	// 源端解析与存在性检查：传输对象不存在是参数错误，无需审批
	srcFull, err := t.resolvePathIn(srcRoot, p, srcLabel(direction))
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(srcFull); err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("transfer_file 源文件/目录不存在：" + p)
		}
		return "", fmt.Errorf("检查源文件失败: %w", err)
	}

	// 目标端解析 + 覆盖判定：先把源端路径（可能为 ~ 形式或源端内绝对路径）归一化
	// 为相对源端根的路径，再在目标端按同名相对位置解析——若直接用原始 p 在目标端
	// 解析，~ 形式/绝对路径会展开到目标沙箱之外而必然越界报错（与工具描述不符）。
	rel, err := filepath.Rel(srcRoot, srcFull)
	if err != nil {
		return "", fmt.Errorf("计算目标端相对路径失败: %w", err)
	}
	dstFull, err := t.resolvePathIn(dstRoot, rel, dstLabel(direction))
	if err != nil {
		return "", err
	}
	destExists, finalDst, err := fsFinalTarget(dstFull, srcFull)
	if err != nil {
		return "", err
	}
	if destExists && !args.Overwrite {
		return "", errors.New("transfer_file 目标已存在，未设置 overwrite=true 时拒绝覆盖（覆盖请显式传 overwrite=true）")
	}

	// 审批检查点：download 写桌面（工作目录之外的外部副作用）一律审批 critical=true；
	// upload 写工作区对齐 copy_file（覆盖审批 critical=false，纯新增免审批）。
	// 审批在参数校验与覆盖判定之后、复制执行之前（先校验后审批）。
	switch {
	case direction == "download":
		summary := "下载到桌面：" + p + " → 桌面/" + relTo(dstRoot, finalDst)
		if destExists {
			summary += "（覆盖）"
		}
		if err := t.requestApproval(ctx, "transfer_file", summary, true); err != nil {
			return "", err
		}
	case destExists:
		if err := t.requestApproval(ctx, "transfer_file", "上传并覆盖："+p+" → "+relTo(dstRoot, finalDst), false); err != nil {
			return "", err
		}
	}

	if err := kitfs.CopyEx(srcFull, dstFull, args.Overwrite); err != nil {
		return "", fmt.Errorf("传输失败: %w", err)
	}

	// 结果文案：相对路径展示；源为目录时注明递归复制
	srcInfo, err := os.Stat(srcFull)
	isDir := err == nil && srcInfo.IsDir()
	if direction == "download" {
		out := "已下载：" + p + " → 桌面/" + relTo(dstRoot, finalDst)
		if isDir {
			out += "（目录，已递归复制）"
		}
		return out, nil
	}
	out := "已上传：桌面/" + p + " → " + relTo(dstRoot, finalDst)
	if isDir {
		out += "（目录，已递归复制）"
	}
	return out, nil
}

// srcLabel 返回源端边界标识（报错文案用）。
func srcLabel(direction string) string {
	if direction == "download" {
		return transferLabelWorkspace
	}
	return transferLabelDesktop
}

// dstLabel 返回目标端边界标识（报错文案用）。
func dstLabel(direction string) string {
	if direction == "download" {
		return transferLabelDesktop
	}
	return transferLabelWorkspace
}

// relTo 把沙箱根内绝对路径转为相对根的展示路径（工具反馈保持相对路径风格）；
// 失败时原样返回。
func relTo(root, fullPath string) string {
	if rel, err := filepath.Rel(root, fullPath); err == nil {
		return rel
	}
	return fullPath
}

// NewTransferFile 创建 transfer_file 工具。
func NewTransferFile(ctx *Context) tool.InvokableTool {
	return &transferFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}
