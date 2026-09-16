# os.Root 试点：read\_file / ls\_dir / delete\_file 切换为 Root 目录句柄模式

## Summary

将文件工具家族中 **read\_file、ls\_dir、delete\_file** 三个工具的 I/O 从"绝对路径字符串 + `os.*` 函数"切换为 **`os.Root`** **目录句柄 + 相对路径**模式（Go 1.24+，项目 go1.26 支持）。`os.Root` 基于目录句柄（Unix `openat` / Windows 目录句柄）做路径解析，从根上杜绝 `../` 逃逸与符号链接逃逸（消除"先校验后打开"的 TOCTOU 竞态窗口）。本次为试点：保留 `resolvePath`（`config.WorkspaceFilePath`）作为第一道防线，`os.Root` 作为第二道防线，**对外行为与输出格式完全不变**，现有测试全量回归即可验证。

## Current State Analysis

### fsToolBase（fs\_base.go）

* `fsToolBase` 持有 `ctx *Context` 与可注入的 `workspaceRoot string`（测试用）。

* `wsRoot()` 返回工作区根（注入优先，否则 `config.WorkspaceDir()`）。

* `resolvePath(p)` 调用 `config.WorkspaceFilePath(root, p)` 做 Abs+Clean+EvalSymlinks+前缀边界校验，返回**绝对路径**。

* `requestApproval(ctx, toolName, summary, critical)`：ctx 为空放行；ctx 非空但 Approver 未注入则报错；否则转 Approver。

### 三个工具的 I/O 调用点

| 工具                 | 当前 I/O（绝对路径）                                                                              | os.Root 对应                                                       |
| ------------------ | ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| read\_file.go L109 | `os.Open(fullPath)`（L116 defer Close；L120 `kitfs.IsBinaryFile(f)` 接受 `*os.File`）          | `root.Open(rel)` 返回 `*os.File`，二进制检测兼容                           |
| ls\_dir.go L82     | `os.ReadDir(fullPath)`（返回 `[]fs.DirEntry`）                                                | `root.ReadDir(rel)` 签名一致                                         |
| delete\_file.go    | `os.Lstat(fullPath)`（IsDir 判定）、`os.ReadDir(fullPath)`（空目录判定）、`os.Remove` / `os.RemoveAll` | `root.Lstat` / `root.ReadDir` / `root.Remove` / `root.RemoveAll` |

### 关键兼容点

* `os.Root.Lstat` 返回 `fs.FileInfo`（`IsDir()` 可用）；`Root.ReadDir` 返回 `[]fs.DirEntry`，与 `os.ReadDir` 一致。

* `Root.Open` 返回 `*os.File`，`kitfs.IsBinaryFile(f)` 直接接受。

* `os.Root` 拒绝 `..` 逃逸与指向 root 外的符号链接（第二道防线）；`resolvePath` 先行的绝对路径校验不变，双防线叠加不改变任何既有行为。

* ls\_dir 的 `target` 缺省为空：`resolvePath("")` 返回根绝对路径，相对化后为 `"."`，`root.ReadDir(".")` 语义正确。

## Proposed Changes

### 1. fs\_base.go：新增两个辅助方法（保留现有 resolvePath/wsRoot/requestApproval 不动）

```go
// openRootFor 打开工作区根的 os.Root 目录句柄，并把工作区内绝对路径转换为
// 相对工作区根的相对路径（供 Root 相对路径操作）。调用方负责 Close。
// os.Root 基于目录句柄解析路径，拒绝 ../ 逃逸与指向 root 外的符号链接，
// 作为 resolvePath 之后的第二道防线。
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
```

* `fs_base.go` 已 import `os` / `path/filepath`，无需新增 import。

* 生命周期决策：**每次工具调用新建 Root、用完即 Close**（目录打开成本可忽略；测试注入临时目录时避免缓存串扰；工具本身是一次性执行语义）。

### 2. read\_file.go：`os.Open` → `root.Open`

InvokableRun 中 L104-116 段替换：

```go
fullPath, err := t.resolvePath(path)
if err != nil {
	return "", err
}
root, rel, err := t.openRootFor(fullPath)
if err != nil {
	return "", fmt.Errorf("打开工作目录失败: %w", err)
}
defer func() { _ = root.Close() }()

f, err := root.Open(rel)
if err != nil {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "", fmt.Errorf("读取文件失败: %w", err)
}
defer func() { _ = f.Close() }()
```

* 其后 `kitfs.IsBinaryFile(f)`、`readRunesBounded(f, ...)` 完全不变。

* import 变更：删除 `os`（L16，仅 L109 使用）；`filepath` 未用到（无需加）。

* 文件头注释补一句"经 os.Root 目录句柄做第二道防逃逸"。

### 3. ls\_dir.go：`os.ReadDir` → `root.ReadDir`

InvokableRun 中 L77-88 段替换：

```go
fullPath, err := t.resolvePath(target)
if err != nil {
	return "", err
}
root, rel, err := t.openRootFor(fullPath)
if err != nil {
	return "", fmt.Errorf("打开工作目录失败: %w", err)
}
defer func() { _ = root.Close() }()

entries, err := root.ReadDir(rel)
if err != nil {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "", fmt.Errorf("列出目录失败: %w", err)
}
```

* 后续遍历逻辑（`d.IsDir()` / `d.Name()`）不变。

* import 变更：删除 `os`（仅 L82 使用）。

* 文件头注释补 os.Root 说明。

### 4. delete\_file.go：四个 `os.*` 调用全部换 `root.*`

InvokableRun 改造：

* 在"拒绝删工作区根"检查（`fullPath == root`）之后、`os.Lstat` 之前打开 root：

```go
root, rel, err := t.openRootFor(fullPath)
if err != nil {
	return "", fmt.Errorf("打开工作目录失败: %w", err)
}
defer func() { _ = root.Close() }()

// 存在性检查
stat, err := root.Lstat(rel)
if err != nil {
	if os.IsNotExist(err) {
		return "", errors.New("delete_file 目标文件/目录不存在")
	}
	return "", fmt.Errorf("检查目标文件失败: %w", err)
}

// 目录递归判定（空目录缺省可删）
if stat.IsDir() && !args.Recursive {
	entries, err := root.ReadDir(rel)
	if err != nil {
		return "", fmt.Errorf("检查目录内容失败: %w", err)
	}
	if len(entries) > 0 {
		return "", errors.New("delete_file 目标是非空目录，未设置 recursive=true 时拒绝删除（目录递归删除需显式确认）")
	}
}

// 审批检查点（critical=true）不变

if args.Recursive {
	err = root.RemoveAll(rel)
} else {
	err = root.Remove(rel)
}
```

* `os.IsNotExist(err)` 判定对 `Root.Lstat` 返回的 error 同样适用（底层仍是 `PathError`/`*SyscallError` 包装）。

* "拒绝删工作区根"检查保留 `fullPath == root`（先用 `wsRoot()` 取根比较），逻辑不动。

* import 变更：`os` 仅剩 `os.IsNotExist` 一处（保留 os import），删除 `os.Lstat/os.ReadDir/os.Remove/os.RemoveAll` 调用。

* 文件头注释补 os.Root 说明。

### 5. 测试

* **不改现有测试**：三个工具的既有用例（fs\_tools\_test.go 的 TestReadFileBasics/TestReadFilePagination/TestReadFileLineNumbers、TestLsDir；fs\_operate\_test.go 的 TestDeleteFile 全 9 个子测试）行为不变，全量回归。

* **不新增符号链接逃逸测试**：Windows 创建符号链接需管理员/开发者模式权限，测试有平台依赖；`resolvePath` 已是第一道防线且覆盖逃逸场景，Root 第二道防线不改变既有测试的可观测行为。

* 可顺手确认 fs\_tools\_test.go 中 `newTestReadFile`/`newTestLsDir` 注入 `workspaceRoot` 的路径在 `openRootFor` 下仍正确（`os.OpenRoot(workspaceRoot)` 打开临时目录）。

## Assumptions & Decisions

1. **保留双防线**：`resolvePath`（绝对路径校验）仍是主防线，`os.Root` 是第二道。改造最小化、行为零变化，试点验证 Root 的可用性后再评估是否前移为唯一防线。
2. **Root 句柄每次调用新建**：不常驻缓存（避免测试注入目录串扰、生命周期简单、工具一次性执行语义匹配）。
3. **不涉及 copy\_file / move\_file / write\_file / edit\_file / glob**：copy/move 依赖 go-kit（不认 Root），write/edit 缺 Root.WriteFile，glob 需换遍历引擎——这些留在试点验证后再议。
4. **错误文案不变**：仍使用"读取文件失败"/"列出目录失败"/"删除失败"等既有文案与错误前缀。
5. **Go 版本**：项目 go.mod 1.26.0、本地 1.26.4，`os.Root` 可用，无依赖升级。

## Verification

1. `go build ./...` 编译通过（注意三个文件的 import 清理）。
2. `go vet ./internal/agent/...` 通过。
3. `go test -timeout 180s ./internal/agent/tools/` 全量通过（三个工具的既有测试回归，无行为变化）。
4. `go test -timeout 180s ./internal/agent/` 通过（注：`TestRequestApprovalRejectsParallel` 有已知偶发挂起，若触发则单独重跑该测试）。
5. 人工抽查：read\_file 分页续读、ls\_dir 单层列举、delete\_file 非空目录 recursive 拒绝，输出与改造前一致。

