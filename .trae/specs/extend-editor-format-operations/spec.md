# 扩展编辑器格式化操作菜单 Spec

## Why
编辑器操作菜单目前只有 JSON 格式化/压缩两个操作，用户希望补充更多格式化/压缩操作，覆盖 XML、HTML、CSS、JavaScript、SQL、CSV、YAML、TOML 等常见格式，使编辑器具备更全面的文本处理能力。

## What Changes
- **安装 4 个 npm 依赖**: `js-yaml` (YAML 解析/序列化)、`smol-toml` (TOML 解析/序列化)、`sql-formatter` (SQL 格式化)、`js-beautify` (CSS/JS 格式化)
- **重构操作分组方式**: 将当前单一「格式化」分组，拆分为**按格式类型分组**（JSON/XML/HTML/CSS/JavaScript/SQL/CSV/YAML/TOML），每个分组下包含「格式化」和「压缩」两个操作项（CSV 仅格式化）
- **更新 `editor-actions.js`**:
  - 新增所有格式的操作条目到 `EDITOR_ACTIONS` 注册表
  - XML/HTML/CSV 格式化/压缩为纯函数，零外部依赖（浏览器内置 API）
  - 错误提示信息改为动态（根据操作类型显示对应格式名称，而非硬编码的「不是合法的 JSON」）
  - 各 handler 在解析失败时抛出 `Error`，错误信息包含格式名称，`executeAction` 的 catch 块使用 `e.message` 作为通知文案
- **CSS/JS 格式化/压缩 使用 `js-beautify`**，CSS 调用 `css_beautify`，JS 调用 `js_beautify`；压缩通过 `{ indent_size: 0, preserve_newlines: false }` 配置实现基本压缩
- **新增 `frontend/src/js/formatters/` 目录**（可选，视实现方便程度）：将 XML/HTML/CSV 等零依赖格式化函数抽离为独立模块，保持 `editor-actions.js` 聚焦于注册表与执行引擎

## Impact
- Affected specs: [add-editor-action-menu](file:///d:/峡谷/Dev/本地项目/jot/.trae/specs/add-editor-action-menu/) — 扩展操作注册表，不影响既有功能
- Affected code:
  - `frontend/package.json` — 新增 4 个依赖
  - `frontend/src/js/editor-actions.js` — 新增操作条目，改进错误提示，重构分组
  - `frontend/src/css/components/editor.css` — 无变动（菜单样式已足够容纳更多子菜单项）
  - `frontend/index.html` — 无变动
  - `frontend/src/main.js` — 无变动

## ADDED Requirements

### Requirement: 按格式类型分组
系统 SHALL 将操作菜单从单一「格式化」分组重构为按格式类型分组，每个格式类型作为一个子菜单。

#### Scenario: 菜单结构
- **WHEN** 用户点击操作菜单按钮
- **THEN** 菜单显示以下子菜单分组（按此顺序）：JSON、XML、HTML、CSS、JavaScript、SQL、CSV、YAML、TOML
- **AND** 每个分组（除 CSV 外）包含「格式化」和「压缩」两个操作项
- **AND** CSV 分组仅包含「CSV 格式化」一个操作项

### Requirement: 新增 8 种格式操作
系统 SHALL 支持以下格式的格式化与压缩操作（共 17 个操作项）。

#### Scenario: XML 格式化/压缩
- **WHEN** 用户对合法 XML 文本执行「XML 格式化」
- **THEN** 使用 `DOMParser` 解析 → 递归遍历节点 → 带 2 空格缩进重组输出
- **AND** 执行「XML 压缩」时输出去掉多余空白文本节点的紧凑形式

#### Scenario: HTML 格式化/压缩
- **WHEN** 用户对合法 HTML 文本执行「HTML 格式化」
- **THEN** 使用 `DOMParser` 解析 → 自定义序列化器（保留 HTML 标签名大小写，不转为 XHTML）→ 带缩进重组
- **AND** 执行「HTML 压缩」时去掉多余空白文本节点

#### Scenario: CSS 格式化/压缩
- **WHEN** 用户对 CSS 文本执行「CSS 格式化」
- **THEN** 调用 `js_beautify.css_beautify(source, { indent_size: 2 })` 输出美化结果
- **AND** 执行「CSS 压缩」时调用 `js_beautify.css_beautify(source, { indent_size: 0, preserve_newlines: false })` 输出紧凑形式

#### Scenario: JavaScript 格式化/压缩
- **WHEN** 用户对 JS 文本执行「JS 格式化」
- **THEN** 调用 `js_beautify.js_beautify(source, { indent_size: 2 })` 输出美化结果
- **AND** 执行「JS 压缩」时调用 `js_beautify.js_beautify(source, { indent_size: 0, preserve_newlines: false })` 输出紧凑形式

#### Scenario: SQL 格式化/压缩
- **WHEN** 用户对 SQL 文本执行「SQL 格式化」
- **THEN** 调用 `sqlFormatter.format(source, { indent: '  ' })` 输出美化结果
- **AND** 执行「SQL 压缩」时调用 `sqlFormatter.format(source, { indent: '  ' })` 并额外去除多余换行

#### Scenario: CSV 格式化
- **WHEN** 用户对 CSV 文本执行「CSV 格式化」
- **THEN** 按行按列拆分，计算每列最大宽度，使用 `padEnd` 对齐各列，输出表格状对齐文本
- **AND** CSV 不提供压缩操作

#### Scenario: YAML 格式化/压缩
- **WHEN** 用户对合法 YAML 文本执行「YAML 格式化」
- **THEN** 调用 `yaml.load(text)` 解析 → `yaml.dump(parsed, { indent: 2, lineWidth: 120 })` 输出美化结果
- **AND** 执行「YAML 压缩」时调用 `yaml.dump(parsed, { indent: 2, lineWidth: -1, flowLevel: 0 })` 输出紧凑形式

#### Scenario: TOML 格式化/压缩
- **WHEN** 用户对合法 TOML 文本执行「TOML 格式化」
- **THEN** 调用 `smolToml.parse(text)` 解析 → 自定义序列化输出美化结果
- **AND** 执行「TOML 压缩」时去掉多余空白

### Requirement: 动态错误提示
系统 SHALL 在操作失败时显示包含格式名称的提示信息，而非硬编码的「不是合法的 JSON」。

#### Scenario: 各格式解析失败
- **WHEN** 用户对非法 XML 执行「XML 格式化」
- **THEN** 编辑器内容不变，通知显示「不是合法的 XML」
- **AND** 对非法 YAML 执行操作时通知显示「不是合法的 YAML」
- **AND** 以此类推，每种格式显示对应的格式名称

## MODIFIED Requirements
### Requirement: 操作注册表 errorLabel 字段
`EDITOR_ACTIONS` 中的每个操作项新增 `errorLabel` 字段（如 `'XML'`、`'YAML'`），`executeAction` 的 catch 块使用 `e.message` 优先，回落为 `errorLabel` 拼接的提示文案。

## REMOVED Requirements
无。