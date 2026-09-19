package services

// 本文件实现工作区管理器服务（WorkspaceService）：为用户提供对 AI 助手工作目录
// （~/.jot/workspace）的浏览与文件管理能力——递归文件树浏览、上传文件/目录
// （用户任意来源 → 工作区根，重名自动改名）、下载（工作区 → 桌面根同名相对
// 路径，重名自动改名）、删除（根目录与目录非递归删除受保护）。
// 所有进入工作区的路径一律经 config.WorkspaceFilePath 做 Clean + 符号链接解析 +
// 边界校验，防 ../ 逃逸与 symlink/junction 逃逸；复制复用 go-kit fs 的 CopyEx
// （目录递归复制、临时文件 + 原子重命名、失败自动回滚）。批次操作单项失败不中断，
// 错误写入对应 WorkspaceTransferResult.Error 字段，由 app.go 绑定层统一记录日志，
// 本层不依赖第三方日志库。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kitfs "gitee.com/MM-Q/go-kit/fs"

	"jot/internal/config"
)

// WorkspaceFileEntry 工作区文件树节点
type WorkspaceFileEntry struct {
	Name     string // 显示名
	RelPath  string // 相对工作区根路径（/ 分隔）
	IsDir    bool
	Size     int64                // 目录为 0
	ModTime  int64                // unix 秒
	Children []WorkspaceFileEntry // 目录才有
}

// WorkspaceTransferResult 单次上传/下载/删除结果
type WorkspaceTransferResult struct {
	Name   string // 原文件名/目录名
	Target string // 实际写入/删除的相对路径（含自动改名后的新名）
	Error  string // 失败原因，空表示成功
}

// WorkspaceService 工作区文件管理器服务：路径根支持测试注入，空值按默认解析
// （工作区 = config.WorkspaceDir()，桌面 = os.UserHomeDir()/Desktop）。
type WorkspaceService struct {
	workspaceRoot string // 测试注入用，空则取 config.WorkspaceDir()
	homeDir       string // 测试注入用，空则取 os.UserHomeDir()
	desktopRoot   string // 测试注入用，空则取 homeDir + "Desktop"
}

// NewWorkspaceService 创建一个新的 WorkspaceService 实例
func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{}
}

// ListWorkspaceFiles 递归收集工作区文件树：目录在前、名称升序；文件记录大小与
// 修改时间（unix 秒），目录大小置 0；相对路径统一用 / 分隔。空目录保留显示
// （以 Children 为空数组表示，新建空文件夹应立即可见）。工作区根不存在或为空时
// 返回空列表。
func (s *WorkspaceService) ListWorkspaceFiles() ([]WorkspaceFileEntry, error) {
	root, err := s.wsRoot()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceFileEntry{}, nil
		}
		return nil, fmt.Errorf("读取工作区目录失败: %w", err)
	}
	// 递归构建一级条目（空目录保留显示），随后整体排序
	tree, err := s.buildTree(root, entries, "")
	if err != nil {
		return nil, err
	}
	sortEntries(tree)
	return tree, nil
}

// UploadFiles 批量上传用户选中的文件/目录到工作区根：目标取源路径基名，
// 已存在时自动改名 name (1).ext、name (2).ext……（目录同样适用；无扩展名文件
// 如 README 改名为 README (1)；以 ~/. 开头的隐藏文件保持原名再追加序号）。
// 复制经 go-kit CopyEx（目录递归、原子性），目标路径经沙箱校验；单项失败
// 不中断批次，错误写入对应结果 Error 字段，Target 填实际写入的相对工作区根
// 的 / 分隔路径（含自动改名后的新名）。
func (s *WorkspaceService) UploadFiles(paths []string) ([]WorkspaceTransferResult, error) {
	root, err := s.wsRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建工作区目录失败: %w", err)
	}
	results := make([]WorkspaceTransferResult, 0, len(paths))
	for _, p := range paths {
		results = append(results, s.uploadOne(root, root, p))
	}
	return results, nil
}

// UploadPathsToWorkspace 拖拽上传文件/目录到工作区指定相对目录：paths 为拖拽传入
// 的绝对路径（文件/目录混合均可，源为用户任意位置无需沙箱校验），targetRel 为
// 工作区相对路径（/ 分隔），空串或 "/" 表示根目录。targetRel 非空时先经
// config.WorkspaceFilePath 做沙箱校验（防 ../ 逃逸与 symlink 逃逸），并确认目标
// 目录存在且为目录，不合法直接整体返回错误（不逐条处理）。之后逐条复用
// uploadOne 全部既有逻辑（重名自动改名 / 单项失败不中断批次 / CopyEx 原子复制）。
func (s *WorkspaceService) UploadPathsToWorkspace(paths []string, targetRel string) ([]WorkspaceTransferResult, error) {
	root, err := s.wsRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建工作区目录失败: %w", err)
	}
	// 解析目标目录：空串或 "/" 视为根目录，否则沙箱校验后确认是已存在目录
	targetDir := root
	if targetRel != "" && targetRel != "/" {
		targetDir, err = config.WorkspaceFilePath(root, filepath.FromSlash(targetRel))
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(targetDir)
		if statErr != nil || !info.IsDir() {
			if statErr != nil {
				return nil, fmt.Errorf("拖拽目标目录不存在：%s", targetRel)
			}
			return nil, fmt.Errorf("拖拽目标不是目录：%s", targetRel)
		}
	}
	results := make([]WorkspaceTransferResult, 0, len(paths))
	for _, p := range paths {
		results = append(results, s.uploadOne(root, targetDir, p))
	}
	return results, nil
}

// CreateDirectory 在工作区指定相对目录下新建子目录：targetRel 为父目录（/ 分隔，
// 空串或 "/" 表示根目录）。先经 config.WorkspaceFilePath 沙箱校验父目录（防 ../ 与
// symlink 逃逸）并确认其存在且为目录；name 仅允许单层目录名（不得含路径分隔符、
// 不能为空或 . / ..）。重名直接报错（创建场景需要明确反馈，不复用自动改名）。
// 成功返回新目录的相对工作区根的 / 分隔路径，失败返回错误（一次性操作无需批次结构）。
func (s *WorkspaceService) CreateDirectory(targetRel, name string) (string, error) {
	root, err := s.wsRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("创建工作区目录失败: %w", err)
	}
	// 目录名校验：不能为空、不能是 . 或 ..、不得包含路径分隔符（只允许单层目录名）
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("目录名无效：%q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("目录名不能包含路径分隔符：%q", name)
	}
	// Windows 文件系统非法字符/尾部点与空格（mkdir 会失败，提前给出中文提示）
	if strings.ContainsAny(name, `<>:"|?*`) {
		return "", fmt.Errorf("目录名包含非法字符（< > : \" | ? *）：%q", name)
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return "", fmt.Errorf("目录名不能以点或空格结尾：%q", name)
	}
	// 解析父目录：空串或 "/" 视为根，否则沙箱校验后确认是已存在目录
	parent := root
	if targetRel != "" && targetRel != "/" {
		parent, err = config.WorkspaceFilePath(root, filepath.FromSlash(targetRel))
		if err != nil {
			return "", err
		}
		info, statErr := os.Stat(parent)
		if statErr != nil {
			return "", fmt.Errorf("目标目录不存在：%s", targetRel)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("目标不是目录：%s", targetRel)
		}
	}
	// 重名直接报错（创建重点明确反馈，不复用自动改名）
	newFull := filepath.Join(parent, name)
	if _, err := os.Lstat(newFull); err == nil {
		return "", fmt.Errorf("同名文件或目录已存在：%s", name)
	}
	if err := os.Mkdir(newFull, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	return toSlashRel(root, newFull), nil
}

// DownloadFiles 批量下载工作区文件/目录到桌面根下同名相对路径（目录自动创建
// 父目录）：源为工作区相对路径，经沙箱校验；桌面目标已存在时自动改名（逻辑同
// 上传，针对最后一段基名）。源不存在时该条返回错误；单项失败不中断批次，
// Target 填实际写入的相对桌面根的 / 分隔路径（含自动改名后的新名）。
func (s *WorkspaceService) DownloadFiles(relPaths []string) ([]WorkspaceTransferResult, error) {
	root, err := s.wsRoot()
	if err != nil {
		return nil, err
	}
	desk, err := s.deskRoot()
	if err != nil {
		return nil, err
	}
	results := make([]WorkspaceTransferResult, 0, len(relPaths))
	for _, rel := range relPaths {
		results = append(results, s.downloadOne(root, desk, rel))
	}
	return results, nil
}

// DeleteFiles 批量删除工作区文件/目录：relPaths 为工作区相对路径，经沙箱校验。
// 拒绝删除工作区根目录本身（relPath 为空、. 或 /）；目录删除要求 recursive=true
// （为 false 时该条返回错误不删除），文件直接删除。单项失败不中断批次，
// Target 填被删除的相对工作区根的 / 分隔路径。
func (s *WorkspaceService) DeleteFiles(relPaths []string, recursive bool) ([]WorkspaceTransferResult, error) {
	root, err := s.wsRoot()
	if err != nil {
		return nil, err
	}
	results := make([]WorkspaceTransferResult, 0, len(relPaths))
	for _, rel := range relPaths {
		results = append(results, s.deleteOne(root, rel, recursive))
	}
	return results, nil
}

// uploadOne 上传单个源路径到工作区 targetDir（根目录场景下调用方传 targetDir=root）；
// 错误写入结果 Error 字段而非中断批次。
func (s *WorkspaceService) uploadOne(root, targetDir, srcPath string) WorkspaceTransferResult {
	name := filepath.Base(srcPath)
	res := WorkspaceTransferResult{Name: name}
	if name == "." || name == string(filepath.Separator) {
		res.Error = "源路径无效：" + srcPath
		return res
	}
	// 源存在性检查（源为用户任意位置，无需沙箱校验）
	if _, err := os.Lstat(srcPath); err != nil {
		if os.IsNotExist(err) {
			res.Error = "源文件/目录不存在：" + srcPath
		} else {
			res.Error = fmt.Sprintf("检查源文件失败: %v", err)
		}
		return res
	}
	// 防止选择工作区自身或其子目录作为上传源（递归复制会无限膨胀沙箱），给出中文提示
	if srcAbs, err := filepath.Abs(srcPath); err == nil {
		if rel, err := filepath.Rel(root, srcAbs); err == nil {
			// rel=="." 表示源即工作区根；rel 不以 ".." 开头表示位于工作区内
			if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
				res.Error = "不能上传工作区自身或其子目录"
				return res
			}
		}
	}
	// 目标取源基名落在 targetDir 下，重名自动改名，并经沙箱校验
	finalDst, err := config.WorkspaceFilePath(targetDir, uniqueTarget(targetDir, name))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if err := kitfs.CopyEx(srcPath, finalDst, false); err != nil {
		res.Error = "上传失败: " + err.Error()
		return res
	}
	res.Target = toSlashRel(root, finalDst)
	return res
}

// downloadOne 下载单个工作区相对路径到桌面根同名相对位置；错误写入结果字段。
func (s *WorkspaceService) downloadOne(root, desk, rel string) WorkspaceTransferResult {
	name := filepath.Base(rel)
	res := WorkspaceTransferResult{Name: name}
	if name == "." || name == string(filepath.Separator) {
		res.Error = "源路径无效：" + rel
		return res
	}
	// 源解析 + 沙箱校验（防 ../ 逃逸与符号链接逃逸）
	srcFull, err := config.WorkspaceFilePath(root, rel)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if _, err := os.Lstat(srcFull); err != nil {
		if os.IsNotExist(err) {
			res.Error = "源文件/目录不存在：" + rel
		} else {
			res.Error = fmt.Sprintf("检查源文件失败: %v", err)
		}
		return res
	}
	// 目标 = 桌面根下同名相对路径；显式沙箱校验双保险：源端 WorkspaceFilePath 已保证
	// rel 无逃逸，此处再以桌面为沙箱根校验一次，防止未来 rel 来源变化破坏传递性保证
	dstFull, err := config.SandboxFilePath(desk, rel, "~/Desktop")
	if err != nil {
		res.Error = err.Error()
		return res
	}
	finalDst := uniqueTarget(filepath.Dir(dstFull), filepath.Base(dstFull))
	if err := os.MkdirAll(filepath.Dir(finalDst), 0o755); err != nil {
		res.Error = fmt.Sprintf("创建桌面目录失败: %v", err)
		return res
	}
	if err := kitfs.CopyEx(srcFull, finalDst, false); err != nil {
		res.Error = "下载失败: " + err.Error()
		return res
	}
	res.Target = toSlashRel(desk, finalDst)
	return res
}

// deleteOne 删除单个工作区相对路径；错误写入结果字段而非中断批次。
func (s *WorkspaceService) deleteOne(root, rel string, recursive bool) WorkspaceTransferResult {
	res := WorkspaceTransferResult{Name: filepath.Base(rel)}
	// 根目录保护：relPath 为空、. 或 /（跨平台分隔符）均视为工作区根本身
	cleaned := filepath.Clean(rel)
	if rel == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		res.Error = "不允许删除工作区根目录"
		return res
	}
	full, err := config.WorkspaceFilePath(root, rel)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			res.Error = "源文件/目录不存在：" + rel
		} else {
			res.Error = fmt.Sprintf("检查源文件失败: %v", err)
		}
		return res
	}
	if info.IsDir() && !recursive {
		res.Error = "目录删除需要 recursive=true"
		return res
	}
	if info.IsDir() {
		err = os.RemoveAll(full)
	} else {
		err = os.Remove(full)
	}
	if err != nil {
		res.Error = "删除失败: " + err.Error()
		return res
	}
	res.Target = toSlash(rel)
	return res
}

// buildTree 递归构建 entries（某目录下的一级条目）对应的文件树。空目录保留显示
// （以 Children 为空数组表示），便于用户看到新建的空文件夹。
// relBase 为当前目录相对工作区根的路径（根为 ""）。
func (s *WorkspaceService) buildTree(root string, entries []os.DirEntry, relBase string) ([]WorkspaceFileEntry, error) {
	children := make([]WorkspaceFileEntry, 0, len(entries))
	for _, e := range entries {
		childRel := filepath.Join(relBase, e.Name())
		if e.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(root, childRel))
			if err != nil {
				return nil, fmt.Errorf("读取目录 %s 失败: %w", toSlash(childRel), err)
			}
			sub, err := s.buildTree(root, subEntries, childRel)
			if err != nil {
				return nil, err
			}
			children = append(children, WorkspaceFileEntry{
				Name:     e.Name(),
				RelPath:  toSlash(childRel),
				IsDir:    true,
				ModTime:  dirEntryModTime(e),
				Children: sub, // 空目录以 Children:[] 保留显示（新建空文件夹必须可见）
			})
			continue
		}
		children = append(children, WorkspaceFileEntry{
			Name:    e.Name(),
			RelPath: toSlash(childRel),
			Size:    dirEntrySize(e),
			ModTime: dirEntryModTime(e),
		})
	}
	sortEntries(children)
	return children, nil
}

// sortEntries 对同级条目排序：目录在前，同类型按名称升序。
func sortEntries(entries []WorkspaceFileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir // 目录在前
		}
		return entries[i].Name < entries[j].Name
	})
}

// dirEntrySize 取目录条目大小（失败返回 0）。
func dirEntrySize(e os.DirEntry) int64 {
	info, err := e.Info()
	if err != nil {
		return 0
	}
	return info.Size()
}

// dirEntryModTime 取目录条目修改时间（unix 秒，失败返回 0）。
func dirEntryModTime(e os.DirEntry) int64 {
	info, err := e.Info()
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

// uniqueTarget 在 dir 下为 name 生成不冲突的目标路径：目标不存在时原样返回；
// 已存在时自动追加序号 name (1).ext、name (2).ext……（无扩展名文件如 README
// 改名为 README (1)；以 ~/. 开头的隐藏文件视为无扩展名整体追加序号，如
// .gitignore → .gitignore (1)，避免 filepath.Ext(".gitignore") 返回整个文件名
// 导致 stem 为空）。
func uniqueTarget(dir, name string) string {
	full := filepath.Join(dir, name)
	if _, err := os.Lstat(full); os.IsNotExist(err) {
		return full
	}
	// 隐藏文件（以 . 或 ~ 开头）保持原名再追加序号，不拆分扩展名
	ext := ""
	if !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "~") {
		ext = filepath.Ext(name)
	}
	stem := strings.TrimSuffix(name, ext)
	for seq := 1; ; seq++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, seq, ext))
		if _, err := os.Lstat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

// toSlashRel 把 root 内绝对路径转为相对 root 的 / 分隔展示路径；失败时原样返回。
func toSlashRel(root, fullPath string) string {
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return fullPath
	}
	return toSlash(rel)
}

// toSlash 把路径分隔符统一转为 /（跨平台相对路径展示）。
func toSlash(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// wsRoot 返回工作区根路径：注入优先，否则取 config.WorkspaceDir()。
func (s *WorkspaceService) wsRoot() (string, error) {
	if s.workspaceRoot != "" {
		return s.workspaceRoot, nil
	}
	return config.WorkspaceDir()
}

// deskRoot 返回用户桌面根路径：注入优先，否则取 homeDir + "Desktop"
// （os.UserHomeDir() 读 USERPROFILE/HOME 环境变量；不做 OneDrive 重定向等特殊解析）。
func (s *WorkspaceService) deskRoot() (string, error) {
	if s.desktopRoot != "" {
		return s.desktopRoot, nil
	}
	home := s.homeDir
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户家目录: %w", err)
		}
		home = h
	}
	return filepath.Join(home, "Desktop"), nil
}
