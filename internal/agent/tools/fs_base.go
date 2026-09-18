package tools

// 本文件定义工作目录文件工具（read_file / write_file / ls_dir / glob）共享的
// 基础：fsToolBase 统一提供工作目录根获取与路径边界校验，保证所有文件工具只能
// 操作 ~/.jot/workspace 内的路径（路径越权一律拒绝）。各工具按"一文件一工具"
// 拆分于 read_file.go / write_file.go / ls_dir.go / glob.go。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"jot/internal/config"
)

// fsToolBase 文件工具共享基础：持有执行上下文与可注入的工作目录根/家目录（测试用）。
type fsToolBase struct {
	ctx           *Context
	workspaceRoot string // 测试注入用，空则取 config.WorkspaceDir()
	homeDir       string // 测试注入用，空则取 os.UserHomeDir()（供 expandTilde 展开 ~ 前缀）
}

// fsFinalTarget 计算复制/移动的最终目标路径：dest 为已存在目录时自动追加源基名
// （对齐 go-kit 智能路径语义，如源 a.txt 复制到已存在的 dir/ 即 dir/a.txt）。
// 返回最终目标是否已存在及最终目标路径，供覆盖判定与审批使用。
func fsFinalTarget(dstFull, srcFull string) (bool, string, error) {
	finalDst := dstFull
	if st, err := os.Lstat(dstFull); err == nil {
		if st.IsDir() {
			finalDst = filepath.Join(dstFull, filepath.Base(srcFull))
		}
	} else if !os.IsNotExist(err) {
		return false, "", err
	}
	_, err := os.Lstat(finalDst)
	switch {
	case err == nil:
		return true, finalDst, nil
	case os.IsNotExist(err):
		return false, finalDst, nil
	default:
		return false, "", err
	}
}

// pathEqualsFold 大小写不敏感路径相等比较（先 Clean 再比较）：Windows 文件系统
// 大小写不敏感，且 resolvePath 的 EqualFold 边界校验会放行大小写变体路径，根保护
// 等判定需与之一致；Unix 上保持精确比较。
func pathEqualsFold(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// checkSrcDestRelation 校验复制/移动的源与最终目标关系：源与目标相同（自覆盖无
// 意义）、或目标位于源目录内部（目录复制到自身子目录会无限递归/磁盘暴涨）都必须
// 前置拦截（参数错误语义，不触发审批）。大小写不敏感比较覆盖 Windows 大小写变体
// 路径（go-kit CopyEx 对子目录自复制的兜底校验是大小写敏感的，存在绕过窗口）。
func checkSrcDestRelation(srcFull, finalDst string) error {
	sep := string(filepath.Separator)
	srcClean := filepath.Clean(srcFull)
	dstClean := filepath.Clean(finalDst)
	inside := false
	if runtime.GOOS == "windows" {
		inside = strings.HasPrefix(strings.ToLower(dstClean), strings.ToLower(srcClean+sep))
	} else {
		inside = strings.HasPrefix(dstClean, srcClean+sep)
	}
	switch {
	case pathEqualsFold(srcClean, dstClean):
		return errors.New("源与目标相同，操作无意义")
	case inside:
		return errors.New("目标位于源目录内部，复制/移动目录到自身子目录会无限递归")
	}
	return nil
}

// relDisplayPath 把工作区内绝对路径转为相对工作区根的展示路径（工具反馈与用户
// 输入风格保持一致）；失败时原样返回。
func (b *fsToolBase) relDisplayPath(fullPath string) string {
	root, err := b.wsRoot()
	if err != nil {
		return fullPath
	}
	if rel, err := filepath.Rel(root, fullPath); err == nil {
		return rel
	}
	return fullPath
}

// wsRoot 返回工作目录根路径：注入优先，否则取 config.WorkspaceDir()。
func (b *fsToolBase) wsRoot() (string, error) {
	if b.workspaceRoot != "" {
		return b.workspaceRoot, nil
	}
	return config.WorkspaceDir()
}

// openRootFor 打开工作区根的 os.Root 目录句柄，并把工作区内绝对路径转换为
// 相对工作区根的相对路径（供 Root 相对路径操作）。调用方负责 Close。
// os.Root 基于目录句柄（Unix openat / Windows 目录句柄）解析路径，拒绝 ../ 逃逸
// 与指向 root 外的符号链接，作为 resolvePath 之后的第二道防线（消除先校验后
// 打开的 TOCTOU 竞态窗口）。
func (b *fsToolBase) openRootFor(fullPath string) (*os.Root, string, error) {
	root, err := b.wsRoot()
	if err != nil {
		return nil, "", err
	}
	h, err := os.OpenRoot(root)
	if err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		_ = h.Close()
		return nil, "", err
	}
	return h, rel, nil
}

// resolvePath 把用户传入路径解析为工作目录内的绝对路径并做边界校验。
// 支持三种写法：相对工作区的路径、以 ~ 开头的路径（前导 ~ 展开为用户家目录，
// 如 ~/.jot/workspace/foo）、或工作区内绝对路径；展开后仍走 WorkspaceFilePath
// 的 Clean + 边界校验 + 符号链接解析，~/ 其它目录照样越界拒绝。
func (b *fsToolBase) resolvePath(p string) (string, error) {
	root, err := b.wsRoot()
	if err != nil {
		return "", err
	}
	expanded, err := b.expandTilde(p)
	if err != nil {
		return "", err
	}
	return config.WorkspaceFilePath(root, expanded)
}

// expandTilde 把路径的前导 ~ 展开为用户家目录：仅处理 p 为 "~" 或以 "~/"、
// "~\\" 开头三种形式；路径中间的 ~（如 notes/~foo.md）不展开、原样返回。
// 家目录优先取注入值（测试用），否则取 os.UserHomeDir()；获取失败 fail-fast
// 报错而非静默降级，避免 "~/..." 被当作相对路径解析出迷惑结果。
func (b *fsToolBase) expandTilde(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") && !strings.HasPrefix(p, `~\`) {
		return p, nil
	}
	home := b.homeDir
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户家目录，请改用相对工作区的路径: %w", err)
		}
		home = h
	}
	if p == "~" {
		return home, nil
	}
	return filepath.Join(home, p[2:]), nil
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
