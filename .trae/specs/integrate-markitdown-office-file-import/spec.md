# 集成 markitdown 支持办公文件导入 Spec

## Why

当前 jot 通过 `fs.IsBinaryPath()`（前 8000 字节空字符检测）判断文件类型，导致 `.docx`、`.pdf`、`.xlsx`、`.pptx`、`.epub` 等常见办公文件被误判为二进制文件而直接拒绝，用户无法将这些文件导入为笔记。项目 `tmp/` 目录下已存在 markitdown 库源码（`github.com/conductor-oss/markitdown`），该库支持 13 种文件格式到 Markdown 的转换，但尚未被项目引用。

## What Changes

- **go.mod**: 通过 `replace` 指令引入 `tmp/markitdown-0.0.1` 作为正式依赖
- **新建 `internal/markitdown/converter.go`**: 封装 markitdown 的校验和转换逻辑，提供统一入口
- **改造 `ImportFiles`**: 批处理改为 goroutine 并发；文件类型校验从 `fs.IsBinaryPath()` 改为 markitdown 三段式判定（纯文本直接读 / 办公文件转换 / 都不支持则拒绝）
- **改造 `readAIChatFiles`**: 同上改造逻辑，支持办公文件上传到 AI 聊天

## Impact

- Affected specs: 文件导入、AI 聊天文件上传
- Affected code: `app.go`（ImportFiles, readAIChatFiles 两个方法）
- New file: `internal/markitdown/converter.go`
- Modified: `go.mod`

## ADDED Requirements

### Requirement: markitdown 封装层

The system SHALL provide a unified entry point for markitdown conversion.

#### Scenario: 纯文本文件判定
- **GIVEN** markitdown 封装层
- **WHEN** 传入 `.txt`/`.md`/`.json` 等文件路径
- **THEN** 返回 `IsPlainText=true`，不执行转换

#### Scenario: 办公文件转换
- **GIVEN** markitdown 封装层
- **WHEN** 传入 `.docx`/`.pdf`/`.xlsx`/`.pptx`/`.epub`/`.html`/`.csv`/`.ipynb`/`.rss` 等文件路径
- **THEN** 返回转换后的 Markdown 字符串

#### Scenario: 不支持的格式
- **GIVEN** markitdown 封装层
- **WHEN** 传入图片/视频/可执行文件等 markitdown 无法处理的文件
- **THEN** 返回 `ErrUnsupportedFormat` 错误

#### Scenario: 转换超时保护
- **GIVEN** markitdown 封装层
- **WHEN** 单个文件转换超过 60 秒
- **THEN** 返回 `ErrConversionTimeout` 错误

### Requirement: ImportFiles 并发导入

The system SHALL support并发导入多个文件，每个文件在独立 goroutine 中处理。

#### Scenario: 混合文件批量导入
- **GIVEN** 用户拖拽多个文件到笔记区
- **WHEN** 文件包含 `.txt`、`.docx`、`.pdf` 和 `.png`
- **THEN** `.txt` 直接读取创建笔记；`.docx`/`.pdf` 转换后创建笔记；`.png` 返回错误（不支持）
- **AND** 各文件互不阻塞，一个文件转换慢不影响其他文件

### Requirement: readAIChatFiles 办公文件转换

The system SHALL support上传办公文件到 AI 聊天。

#### Scenario: 拖拽办公文件到 AI 聊天区
- **GIVEN** AI 聊天对话框
- **WHEN** 用户拖入 `.docx`/`.pdf`/`.xlsx` 等文件
- **THEN** 文件内容被转换为 Markdown 文本后作为 AI 上下文传入

## MODIFIED Requirements

### Requirement: 文件类型校验逻辑

**Before**: 所有入口统一使用 `fs.IsBinaryPath()`（空字符检测）拒绝二进制文件

**After**: 不同入口采用不同策略：
- `ImportFiles`: 用 markitdown 判定，纯文本直接读 / 办公文件转换 / 否则拒绝
- `readAIChatFiles`: 同上
- `ReadTextFile`: **保持不变**，继续使用 `fs.IsBinaryPath()` 拒绝二进制文件

## REMOVED Requirements

### Requirement: fs.IsBinaryPath 在导入中的使用
**Reason**: 被 markitdown 的更精确的 Accepts 判定取代
**Migration**: `ImportFiles` 和 `readAIChatFiles` 中不再调用 `fs.IsBinaryPath`，改用 markitdown 判定；`ReadTextFile` 保持不变。
