# 新增「最大文件限制数」设置 Spec

## Why

目前项目中有 3 处硬编码的 10MB 文件大小限制（文件上传门禁、文件导入门禁、引用总上下文限制），用户无法自定义。需要将其变为可配置的设置项，默认值为 1MB，让用户可以在设置页中调整。

## What Changes

- **新增** `max_file_size` DB 设置键，默认值 `"1048576"`（1MB）
- **新增** `SettingsConfig.MaxFileSize` 字段，类型 `int`，JSON tag `"max_file_size"`
- **替换** 3 处硬编码 `10 * 1024 * 1024` 为动态读取新设置
- **新增** 前端设置页 UI：输入框 + 单位后缀 "MB"，位于「引用截断」下方
- **保留** `ai_ref_max_chars` 原样不动

## Impact

- Affected specs: 设置页、文件上传、文件导入、笔记引用
- Affected code: `internal/database/db.go`、`internal/services/types.go`、`app.go`、`internal/services/note_service.go`、`frontend/index.html`、`frontend/src/main.js`、`frontend/wailsjs/go/models.ts`

## ADDED Requirements

### Requirement: 新增设置项「最大文件限制数」

系统 SHALL 在设置页中新增一个「最大文件限制数」配置项，控制文件上传/导入/引用上下文的大小上限。

#### Scenario: 设置页显示及保存
- **WHEN** 用户打开设置页
- **THEN** 显示「最大文件限制数」输入框，默认值 `1`，单位 `MB`，范围 `1-100`
- **WHEN** 用户修改值并触发 change 事件
- **THEN** 自动保存到后端 `max_file_size` 设置项

#### Scenario: 文件上传使用新设置
- **WHEN** 用户在 AI Chat 中拖拽/选择文件上传
- **THEN** 使用 `max_file_size` 值（字节）检查文件大小，超过则拒绝并提示「文件过大（超过 X MB），请选择小于 X MB 的文件」

#### Scenario: 文件导入使用新设置
- **WHEN** 用户拖拽文件到笔记列表导入
- **THEN** 使用 `max_file_size` 值（字节）检查文件大小，超过则拒绝并提示「文件过大（超过 X MB），无法导入」

#### Scenario: 笔记引用上下文使用新设置
- **WHEN** 构建笔记引用上下文
- **THEN** 使用 `max_file_size` 值（字节）作为总上下文最大字节数，超过则截断后续笔记

### Requirement: 范围校验

系统 SHALL 对 `max_file_size` 进行范围校验。

#### Scenario: 保存时校验
- **WHEN** 保存设置
- **THEN** `max_file_size` 小于 1MB 时重置为 1MB，大于 100MB 时限制为 100MB
- **THEN** 前端输入框 `min=1`, `max=100`

### Requirement: 硬编码替换

系统 SHALL 将以下 3 处硬编码 10MB 替换为动态读取 `max_file_size`。

#### Scenario: 文件上传 `readAIChatFiles`
- **WHEN** `readAIChatFiles` 执行文件大小检查
- **THEN** 读取 `max_file_size` 设置值作为上限
- **THEN** 错误提示中的数字跟随设置值动态显示

#### Scenario: 文件导入 `ImportFiles`
- **WHEN** `ImportFiles` 执行文件大小检查
- **THEN** 读取 `max_file_size` 设置值作为上限
- **THEN** 错误提示中的数字跟随设置值动态显示

#### Scenario: 笔记引用上下文 `BuildNoteRefContext`
- **WHEN** `BuildNoteRefContext` 执行总上下文长度检查
- **THEN** 读取 `max_file_size` 设置值作为上限
- **THEN** 截断逻辑不变，仅上限值改为动态

## MODIFIED Requirements

### Requirement: 设置页布局（已存在）

在 AI 设置区域，**「引用截断」下方**新增一行「最大文件限制数」：
- 标签：`最大文件限制数`
- 输入框：`id="maxFileSize"`，类型 `number`，`min=1`，`max=100`，单位 `MB`
- 初始值 `1`（对应 1MB = 1048576 字节）

## REMOVED Requirements

无。原有 `ai_ref_max_chars` 及其他设置均保留不变。
