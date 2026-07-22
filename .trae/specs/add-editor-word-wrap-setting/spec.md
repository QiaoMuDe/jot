# 编辑器自动换行设置 Spec

## Why
当前 CM6 编辑器在查看/编辑/新建三种模式下均不支持自动换行（横向滚动），用户需要手动水平滚动才能查看长行内容。新增一个"自动换行"开关设置项，使编辑器在三种模式下均能根据配置决定是否换行。

## What Changes
1. **后端 SettingsConfig 结构体**新增 `EditorWordWrap bool` 字段，默认 `false`
2. **前端设置面板编辑器卡片**新增自动换行的 toggle 开关
3. **前端 loadSettings/saveSettings** 同步读写该字段
4. **CM6 初始化**根据该配置条件性注入 `EditorView.lineWrapping` 扩展
5. 三种模式（新建/编辑/查看）共用 `initCodeMirror()`，修改一处即可全覆盖

## Impact
- Affected specs: 编辑器设置项
- Affected code:
  - `internal/services/types.go` — SettingsConfig 结构体
  - `frontend/index.html` — 编辑器设置卡片 HTML
  - `frontend/src/main.js` — loadSettings/saveSettings/initCodeMirror
  - `frontend/src/js/cm6-syntax-highlight.js` — (无需修改，但需验证 lineWrapping 与主题不冲突)

## ADDED Requirements

### Requirement: 编辑器自动换行设置
系统 SHALL 提供一个"自动换行"开关设置项，控制 CM6 编辑器是否启用行内换行。

#### Scenario: 默认状态（未启用）
- **GIVEN** 用户首次打开应用
- **WHEN** 查看设置页编辑器设置卡片
- **THEN** "自动换行"开关默认为关闭状态
- **AND** 任何模式下的 CM6 编辑器均不换行（横向滚动）

#### Scenario: 启用自动换行
- **GIVEN** 用户在设置页开启"自动换行"
- **WHEN** 设置保存后
- **AND WHEN** 新建/打开笔记
- **THEN** CM6 编辑器中长行自动换行显示

#### Scenario: 切换后新建笔记
- **GIVEN** 用户开启"自动换行"并保存
- **WHEN** 新建笔记
- **THEN** 新建模式的 CM6 编辑器启用自动换行

#### Scenario: 切换后编辑已有笔记
- **GIVEN** 用户开启"自动换行"并保存
- **WHEN** 打开已有笔记进入编辑模式
- **THEN** 编辑模式的 CM6 编辑器启用自动换行

#### Scenario: 切换后查看笔记
- **GIVEN** 用户开启"自动换行"并保存
- **WHEN** 打开已有笔记进入查看模式
- **THEN** 查看模式的 CM6 编辑器启用自动换行

#### Scenario: 保存后状态持久化
- **GIVEN** 用户开启"自动换行"并保存
- **WHEN** 重启应用
- **THEN** 设置仍然为开启状态
