# 系统字体枚举跨平台适配（Linux/macOS）Spec

## Why

`internal/fontutil/fonts_windows.go` 通过 GDI API（`EnumFontFamiliesW`）枚举系统字体，文件名 `_windows.go` 后缀使其仅在 Windows 编译。但项目缺少 Linux/macOS 实现，导致 `app.go:GetSystemFonts()`（L3153 引用 `fontutil.GetFonts()`）在非 Windows 平台编译时报 undefined symbol，成为跨平台构建的硬阻塞。

## What Changes

- 新增 `internal/fontutil/fonts_fc.go`（`//go:build linux || darwin`）：共享的 fc-list 解析与字体目录扫描辅助函数。
- 新增 `internal/fontutil/fonts_linux.go`：Linux 实现 `GetFonts()`，优先 `fc-list`，失败回退目录扫描。
- 新增 `internal/fontutil/fonts_darwin.go`：macOS 实现 `GetFonts()`，依次尝试 `fc-list` → `system_profiler SPFontsDataType -json` → 目录扫描。
- **不改动** `fonts_windows.go`（Windows 行为保持原样，避免回归）。
- **不改动** `app.go` 与前端（`GetSystemFonts()` 签名不变；前端 [main.js L1461-1469](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L1461-L1469) 已有空列表降级逻辑）。

## Impact

- Affected specs: 无（新增能力，不改既有 spec）
- Affected code: `internal/fontutil/`（新增 3 个文件）；`app.go:GetSystemFonts()` 间接受益，无需改动
- 行为变化：Windows 无变化；Linux/macOS 从"无法编译"变为"可编译且返回系统字体列表"

## ADDED Requirements

### Requirement: Linux 系统字体枚举
系统 SHALL 在 Linux 平台提供 `GetFonts()` 实现，返回已安装字体族列表。

#### Scenario: fc-list 可用（默认路径）
- **WHEN** 在 Linux 上调用 `fontutil.GetFonts()`
- **THEN** 执行 `fc-list --format="%{family}\n"`，解析每行逗号分隔的多个 family，去重后按字典序排序返回
- **THEN** 返回列表非空（系统装有字体时）

#### Scenario: fc-list 不可用或执行失败
- **WHEN** `fc-list` 命令不存在或执行报错
- **THEN** 回退扫描字体目录（`/usr/share/fonts`、`/usr/local/share/fonts`、`~/.fonts`），以字体文件基名（去扩展名）作为 family 名，去重排序返回
- **THEN** 若目录扫描也无可枚举字体，返回空列表（前端降级兜底）

### Requirement: macOS 系统字体枚举
系统 SHALL 在 macOS 平台提供 `GetFonts()` 实现，返回已安装字体族列表。

#### Scenario: fc-list 可用
- **WHEN** 在 macOS 上调用 `fontutil.GetFonts()` 且系统安装了 fontconfig
- **THEN** 使用与 Linux 相同的 fc-list 解析逻辑返回字体族列表

#### Scenario: fc-list 不可用
- **WHEN** `fc-list` 命令不存在
- **THEN** 执行 `system_profiler SPFontsDataType -json` 并解析 JSON 中 `family` 字段，去重排序返回
- **THEN** 若 system_profiler 也失败，回退扫描字体目录（`/System/Library/Fonts`、`/Library/Fonts`、`~/Library/Fonts`），以文件基名作为 family 名
- **THEN** 全部失败时返回空列表（前端降级兜底）

### Requirement: 与 Windows 行为一致
系统 SHALL 保证三个平台 `GetFonts()` 返回同一契约：去重、排序的 `[]string`；枚举失败返回空列表而非 panic。

#### Scenario: 契约一致性
- **WHEN** 任一平台调用 `fontutil.GetFonts()`
- **THEN** 返回类型为 `[]string`，无重复项，已排序
- **THEN** 内部实现并发安全（每次调用独立收集，不依赖跨调用的包级共享状态；Windows 现有实现保持不变）

### Requirement: 跨平台可编译
系统 SHALL 保证 `internal/fontutil` 包在 Windows/Linux/macOS 三个平台均可编译。

#### Scenario: 三平台编译
- **WHEN** 在 Windows 本机执行 `go build ./...`
- **THEN** 编译通过（Windows 回归）
- **WHEN** 执行 `GOOS=linux go build ./...` 与 `GOOS=darwin go build ./...`
- **THEN** 交叉编译通过，`fontutil.GetFonts()` 无 undefined symbol
