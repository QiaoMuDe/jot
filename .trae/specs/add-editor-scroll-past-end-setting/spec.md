# 编辑器「滚动超出内容」设置项 Spec

## Why
CM6 编辑器目前通过 `scrollPastEnd()` 强制启用「滚过最后一行」行为（文末追加约一屏空白），部分用户希望滚动止于内容末尾。需要一个设置项让用户自行控制该行为。

## What Changes
- 设置页「编辑器」分组新增布尔开关「滚动超出内容」，**默认关闭（false，用户明确要求默认不启用）**，KV 持久化（key: `editor_scroll_past_end`）
- Go 侧 `SettingsConfig` 新增 `EditorScrollPastEnd` 字段（读取用现有 `parseBoolSetting`，只认 `"true"`，缺 key 自然为 false，与默认关闭语义一致，无需新解析函数）
- `initCodeMirror` 新增 `cmScrollCompartment`（每实例 new，同 `cmReadOnlyCompartment` 先例），按开关状态挂载/不挂载 `scrollPastEnd()`
- jotTheme 的 `.cm-content` 底部留白由写死的 `40vh` 改为固定 `0`：滚动严格止于内容末尾（1.85 行高的半行距自带约 8px 视觉缓冲，与顶部 8px 留白对称）；「滚动超出内容」开启时由 scrollPastEnd 内联 padding 接管尾部空间（约一屏）
- 开关切换**即时生效**（Compartment reconfigure），无需切换笔记或重启
- 消除既有缺陷：原 `40vh` 底部留白被 scrollPastEnd 内联样式覆盖属死代码；关闭扩展后会复活导致"关了还能滚 40vh"，改为固定 `0` 同时解决

**行为变更说明**（非 API 破坏）：存量用户升级后默认不可超出滚动（当前行为是可滚约一屏），此为用户明确要求的默认值。

## Impact
- Affected specs: 无既有 spec 依赖此行为（`add-editor-word-wrap-setting` 为同构先例，仅模式复用）
- Affected code:
  - `internal/services/types.go`（SettingsConfig / GetAllSettings / SaveAllSettings）
  - `frontend/index.html`（设置页编辑器分组开关）
  - `frontend/src/main.js`（els 注册、loadSettings、collectSettings、change 监听、initCodeMirror、即时生效 dispatch）
  - `frontend/src/js/cm6-syntax-highlight.js`（jotTheme `.cm-content` padding）

## ADDED Requirements

### Requirement: 滚动超出内容设置项
系统 SHALL 在设置页「编辑器」分组提供「滚动超出内容」布尔开关，默认关闭，持久化至 KV 存储（key: `editor_scroll_past_end`）。

#### Scenario: 默认状态（存量升级 / 首次安装）
- **WHEN** KV 中不存在 `editor_scroll_past_end` 键
- **THEN** 设置项为关闭状态，编辑器滚动止于内容末尾（无额外尾部留白）

#### Scenario: 开启后行为
- **WHEN** 开关打开
- **THEN** 编辑器立即恢复「滚过最后一行」行为（文末约一屏空白，scrollPastEnd 生效），最后一行可滚至编辑区顶部

#### Scenario: 关闭后行为
- **WHEN** 开关关闭
- **THEN** 编辑器滚动止于内容末尾，滚动条长度与实际内容一致（无额外尾部留白），不再出现约一屏空白

#### Scenario: 即时生效
- **WHEN** 当前笔记处于编辑模式时切换开关
- **THEN** 编辑器滚动行为立即更新（无需切换笔记 / 重启），同时设置落库并弹出保存成功通知

#### Scenario: 持久化
- **WHEN** 切换开关后重启应用
- **THEN** 设置状态保持，编辑器按已保存状态初始化

#### Scenario: 不影响其他区域
- **WHEN** 预览面板、AI 消息代码块、MD 语法手册等其他 CM 实例或 DOM 区域滚动
- **THEN** 行为不受该设置影响（scrollPastEnd 仅挂载于笔记编辑器实例）

#### Scenario: 新建/切换笔记后状态一致
- **WHEN** 开关处于任一状态时新建或切换笔记（initCodeMirror 重建实例）
- **THEN** 新编辑器实例按当前设置初始化，与开关状态一致
