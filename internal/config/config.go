// Package config 提供 Jot 应用在用户家目录下的统一根目录（~/.jot）路径解析。
// 所有读写 ~/.jot 下文件的模块都应通过本包获取路径，避免硬编码散落各处。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ~/.jot 下子目录名常量
const (
	DirData      = "data"      // 数据库目录
	DirBackup    = "backup"    // 备份目录
	DirImages    = "images"    // 图片目录
	DirLogs      = "logs"      // 日志目录
	DirWorkspace = "workspace" // AI 助手工作目录（文件/命令工具唯一可写根目录）
)

// JotHomeDir 返回应用根目录: ~/.jot
func JotHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户家目录失败: %w", err)
	}
	return filepath.Join(home, ".jot"), nil
}

// SubDir 返回应用根目录下的子目录路径，如 SubDir(DirData) -> ~/.jot/data
func SubDir(sub string) (string, error) {
	root, err := JotHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, sub), nil
}

// WorkspaceDir 返回 AI 助手工作目录路径: ~/.jot/workspace
func WorkspaceDir() (string, error) {
	return SubDir(DirWorkspace)
}

// EnsureWorkspaceDir 确保工作目录存在（幂等），不存在则创建。
func EnsureWorkspaceDir() error {
	dir, err := WorkspaceDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// WorkspaceFilePath 将相对/绝对路径解析为 workspace 内的绝对路径并校验边界：
// 落出 workspace 返回错误；合法返回清理后的绝对路径。workspaceRoot 须为绝对路径
// （如 WorkspaceDir() 返回值）。p 为相对路径时相对 workspaceRoot 解析；
// p 为工作目录根目录本身同样视为合法（目录操作，如 ls_dir）。
func WorkspaceFilePath(workspaceRoot, p string) (string, error) {
	return SandboxFilePath(workspaceRoot, p, "~/.jot/workspace")
}

// SandboxFilePath 通用沙箱路径底座（由 WorkspaceFilePath 泛化而来）：把相对/绝对
// 路径解析为指定沙箱根（workspace / 用户桌面等）内的绝对路径并校验边界。边界校验
// 会解析符号链接：对「目标路径中最深的已存在祖先」做 EvalSymlinks，使指向沙箱外的
// symlink/junction（含新建文件落在符号链接目录内的场景）暴露真实位置，从而被拒绝
// 放行（防止符号链接逃逸边界）。label 用于报错文案中标识沙箱边界（如 "~/.jot/workspace"）。
func SandboxFilePath(sandboxRoot, p, label string) (string, error) {
	// 规范化根目录：Abs + Clean；根目录须为绝对路径，否则视为配置错误
	rootClean, err := filepath.Abs(filepath.Clean(sandboxRoot))
	if err != nil {
		return "", fmt.Errorf("规范化沙箱根路径失败: %w", err)
	}

	// 外来/畸形绝对路径前置拦截：以 / 或 \ 开头但本机不识别为绝对路径的输入
	// （如 Windows 上传入 Unix 风格 /home/user/...，filepath.IsAbs 为 false），
	// 若放行会被当作相对路径拼接进沙箱根，产生幽灵路径并报出无诊断性的
	// "文件不存在"，模型无法识别格式错误而反复重试。直接报出格式错误。
	// 类 Unix 平台上 /home/... 是合法绝对路径（IsAbs=true），不受影响。
	if (strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`)) && !filepath.IsAbs(p) {
		return "", fmt.Errorf("路径格式无效（不是可识别的绝对路径）;请使用相对路径（如 notes/a.md）或以 %s/ 开头的路径", label)
	}

	// 解析目标路径：相对路径相对根目录拼接，绝对路径直接采用，均做 Clean
	target := p
	if !filepath.IsAbs(p) {
		target = filepath.Join(rootClean, p)
	}
	cleaned := filepath.Clean(target)

	// 符号链接兜底：对父目录 Dir(cleaned) 再做一次解析（覆盖目标文件本身经
	// symlink 指向外部、而 EvalSymlinks 对不存在的目标会失败的场景）。
	dirReal := resolveRealTarget(filepath.Dir(cleaned))
	// 主解析：先尝试解析整个目标真实路径，失败则用父目录解析兜底，最终取
	// 两者中较"深"（路径更长）的一个作为真实路径参与边界校验。
	targetReal := resolveRealTarget(cleaned)
	resolved := targetReal
	if len([]rune(dirReal)) > len([]rune(targetReal)) {
		resolved = dirReal
	}

	// 边界校验：resolved（真实路径）必须等于根目录，或以根目录 + 分隔符为前缀。
	// 大小写比较用 EqualFold（Windows 大小写不敏感；类 Unix 平台两者等价），
	// 前缀比较对二者统一 ToLower 后与分隔符一起判前缀，保证 Windows 大小写不敏感
	// 且不会把兄弟目录（如 root2 或 root_2）误判为在 root 内。
	if !strings.EqualFold(rootClean, resolved) &&
		!strings.HasPrefix(strings.ToLower(resolved), strings.ToLower(rootClean)+string(filepath.Separator)) {
		return "", errors.New("超出 " + label + " 边界，仅允许操作 " + label + " 内的文件；请使用相对路径（如 notes/a.md）或以 " + label + "/ 开头的路径")
	}
	return cleaned, nil
}

// resolveRealTarget 解析 path 的符号链接，返回"最深已存在祖先"的 EvalSymlinks
// 真实路径，并拼接其后尚未创建（不存在）的路径段。path 或其祖先全不存在时
// 返回原 cleaned 路径（无符号链接，原样参与前缀校验）。
func resolveRealTarget(path string) string {
	cleaned := filepath.Clean(path)
	ancestor := cleaned
	var missing []string
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			// 已到文件系统根，无法再向上（路径全部不存在且无符号链接）
			return cleaned
		}
		missing = append([]string{filepath.Base(ancestor)}, missing...)
		ancestor = parent
	}
	real, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return cleaned
	}
	if len(missing) == 0 {
		return real
	}
	// 在真实祖先后拼接尚未创建的路径段，得到完整真实路径
	parts := append([]string{real}, missing...)
	return filepath.Join(parts...)
}
