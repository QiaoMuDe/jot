package tools

// 本文件定义工作目录文件工具（read_file / write_file / ls_dir / glob）共享的
// 基础：fsToolBase 统一提供工作目录根获取与路径边界校验，保证所有文件工具只能
// 操作 ~/.jot/workspace 内的路径（路径越权一律拒绝）。各工具按"一文件一工具"
// 拆分于 read_file.go / write_file.go / ls_dir.go / glob.go。

import (
	"context"
	"errors"

	"jot/internal/config"
)

// fsToolBase 文件工具共享基础：持有执行上下文与可注入的工作目录根（测试用）。
type fsToolBase struct {
	ctx           *Context
	workspaceRoot string // 测试注入用，空则取 config.WorkspaceDir()
}

// wsRoot 返回工作目录根路径：注入优先，否则取 config.WorkspaceDir()。
func (b *fsToolBase) wsRoot() (string, error) {
	if b.workspaceRoot != "" {
		return b.workspaceRoot, nil
	}
	return config.WorkspaceDir()
}

// resolvePath 把用户传入路径解析为工作目录内的绝对路径并做边界校验。
func (b *fsToolBase) resolvePath(p string) (string, error) {
	root, err := b.wsRoot()
	if err != nil {
		return "", err
	}
	return config.WorkspaceFilePath(root, p)
}

// requestApproval 通过 Context.Approver 请求用户审批；拒绝时返回拒绝错误文本，
// 调用方不执行写操作。critical 表示是否为不可绕过的危险操作（命中高危集合），
// 透传给审批实现。
//
// ctx 为空（测试/独立调用的裸工具、无审批机制）时视为放行；但 ctx 非空而
// Approver 未注入（已装配工具上下文却缺审批器）属于生产装配遗漏——审批会被
// 静默跳过，导致危险操作未确认即执行，因此此时直接报错而非放行。
func (b *fsToolBase) requestApproval(ctx context.Context, toolName, summary string, critical bool) error {
	if b.ctx == nil {
		return nil
	}
	if b.ctx.Approver == nil {
		return errors.New(toolName + " 需要审批确认，但当前未配置审批机制")
	}
	return b.ctx.Approver.RequestApproval(ctx, toolName, summary, critical)
}
