# 新增文本转换与文本清理操作 Spec

## Why
编辑器操作菜单目前只有"格式化"分组（各种代码格式的格式化/压缩），缺少日常文本处理功能。添加"文本转换"和"文本清理"两个分组，方便用户快速进行文本大小写转换、格式清理等操作。

## What Changes
- 在 `EDITOR_ACTIONS` 数组中新增两个分组：
  - **「文本转换」**：7 个操作项，全部零依赖
  - **「文本清理」**：5 个操作项，全部零依赖
- 所有新操作项使用 `errorLabel: '文本'`（文本操作不涉及解析，一般不会抛出异常，但保持接口一致）

## Impact
- Affected specs: [extend-editor-format-operations](file:///d:/峡谷/Dev/本地项目/jot/.trae/specs/extend-editor-format-operations/) — 共享同一操作菜单
- Affected code:
  - `frontend/src/js/editor-actions.js` — 在 `EDITOR_ACTIONS` 数组末尾追加 12 个新操作条目
  - 无需新增文件、无需安装依赖、无需修改 CSS

## ADDED Requirements

### Requirement: 文本转换分组
系统 SHALL 在操作菜单中提供"文本转换"分组，包含以下 7 个操作项（子菜单）。

#### Scenario: 大写
- **WHEN** 用户对文本 `hello world` 执行「大写」
- **THEN** 结果输出 `HELLO WORLD`

#### Scenario: 小写
- **WHEN** 用户对文本 `HELLO WORLD` 执行「小写」
- **THEN** 结果输出 `hello world`

#### Scenario: 首字母大写
- **WHEN** 用户对文本 `hello world` 执行「首字母大写」
- **THEN** 结果输出 `Hello World`

#### Scenario: 驼峰式
- **WHEN** 用户对文本 `hello world foo` 执行「驼峰式」
- **THEN** 结果输出 `helloWorldFoo`

#### Scenario: 蛇形式
- **WHEN** 用户对文本 `helloWorldFoo` 执行「蛇形式」
- **THEN** 结果输出 `hello_world_foo`

#### Scenario: 行反转
- **WHEN** 用户对文本 `a\nb\nc` 执行「行反转」
- **THEN** 结果输出 `c\nb\na`

#### Scenario: 字符反转
- **WHEN** 用户对文本 `abc` 执行「字符反转」
- **THEN** 结果输出 `cba`

### Requirement: 文本清理分组
系统 SHALL 在操作菜单中提供"文本清理"分组，包含以下 5 个操作项（子菜单）。

#### Scenario: 去除多余空格
- **WHEN** 用户对文本 ` a   b   c ` 执行「去除多余空格」
- **THEN** 结果输出 `a b c`

#### Scenario: 去除空行
- **WHEN** 用户对文本 `a\n\n\nb` 执行「去除空行」
- **THEN** 结果输出 `a\nb`

#### Scenario: 行尾空格清理
- **WHEN** 用户对文本 `a  \nb  \nc` 执行「行尾空格清理」
- **THEN** 结果输出 `a\nb\nc`

#### Scenario: Tab 转空格
- **WHEN** 用户对文本 `\ta\tb` 执行「Tab 转空格」
- **THEN** 结果输出 `  a  b`（Tab 替换为 2 空格）

#### Scenario: 空格转 Tab
- **WHEN** 用户对文本 `  a  b` 执行「空格转 Tab」
- **THEN** 结果输出 `\ta\tb`（2 连续空格替换为 Tab）

## MODIFIED Requirements
无。

## REMOVED Requirements
无。