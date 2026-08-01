# 编辑器操作模块化拆分 Spec

## Why

`editor-actions.js` 已增长到约 690 行，同时承载操作项定义、渲染逻辑、执行引擎和辅助函数。与已独立成文件的 MD 语法操作（`md-syntax.js`）模式不一致。将各分组操作逻辑拆分为独立文件，让主文件只保留渲染与执行引擎，职责单一、便于维护和扩展。

## What Changes

- 新建 `frontend/src/js/editor-actions/format.js`：迁移「格式化」分组（18 项）及 `compactSQL`、`convertSQLCase` 辅助函数
- 新建 `frontend/src/js/editor-actions/text-transform.js`：迁移「文本转换」分组（7 项）
- 新建 `frontend/src/js/editor-actions/text-clean.js`：迁移「文本清理」分组（5 项）
- 新建 `frontend/src/js/editor-actions/encode-decode.js`：迁移「编码解码」分组（6 项）
- 修改 `frontend/src/js/editor-actions.js`：删除已迁移的操作项与辅助函数，改为导入并展开四个新模块
- 无新增依赖，无功能行为变化

## Impact

- Affected files:
  - `frontend/src/js/editor-actions/format.js`（新建，约 260 行）
  - `frontend/src/js/editor-actions/text-transform.js`（新建，约 75 行）
  - `frontend/src/js/editor-actions/text-clean.js`（新建，约 50 行）
  - `frontend/src/js/editor-actions/encode-decode.js`（新建，约 60 行）
  - `frontend/src/js/editor-actions.js`（从约 690 行缩减到约 230 行）

## ADDED Requirements

### Requirement: 各分组操作项模块化

每个分组拆分为独立文件，默认导出操作项数组：
- `format.js` 默认导出 `FORMAT_ACTIONS`
- `text-transform.js` 默认导出 `TEXT_TRANSFORM_ACTIONS`
- `text-clean.js` 默认导出 `TEXT_CLEAN_ACTIONS`
- `encode-decode.js` 默认导出 `ENCODE_DECODE_ACTIONS`

#### Scenario: format.js
- **WHEN** 打开 `frontend/src/js/editor-actions/format.js`
- **THEN** 其中包含 18 个格式化操作项（JSON/XML/HTML/CSS/JS/SQL/CSV/YAML/TOML），包含 `compactSQL` 与 `convertSQLCase` 辅助函数，引用 formatters 目录路径为 `../formatters/...`

#### Scenario: 其他模块
- **WHEN** 打开对应文件
- **THEN** 其中包含原分组全部操作项，行为与原实现完全一致

### Requirement: editor-actions.js 精简

`editor-actions.js` 只保留：第三方依赖导入、四个新模块导入、`EDITOR_ACTIONS` 聚合数组、`initEditorActionsMenu` 渲染与执行引擎、`window` 暴露。

#### Scenario: 聚合操作项
- **WHEN** 查看 `EDITOR_ACTIONS` 数组
- **THEN** 该数组通过展开四个模块（`...FORMAT_ACTIONS` 等）组成，删除原内联的操作项定义与辅助函数

#### Scenario: 行为不变
- **WHEN** 用户点击任意菜单操作项
- **THEN** 执行结果与拆分前完全一致，无任何功能差异

## MODIFIED Requirements

无（纯重构，不改变任何行为）

## REMOVED Requirements

### Requirement: editor-actions.js 内联操作项定义
**Reason**: 迁移到独立模块文件，主文件只保留渲染与执行引擎
**Migration**: 操作项定义移至各分组对应文件，`EDITOR_ACTIONS` 通过展开导入合并