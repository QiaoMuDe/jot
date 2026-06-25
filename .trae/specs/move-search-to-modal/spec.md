# 笔记搜索迁移到居中弹窗 Spec

## Why
topbar 内的笔记搜索框占用了大量顶部空间（`#searchInput` + `.topbar-search` 容器，max-width 320px），是用户反复要求降低 topbar 高度的根源（56→48→40）。将搜索功能迁移到居中弹窗（modal）后，topbar 可以做到极致紧凑，弹窗也提供更集中的搜索体验（关键字+多过滤器+结果列表），符合 Windows 桌面应用习惯（Outlook/Everything/Listary）。

## What Changes
- **BREAKING**: 移除 topbar 内的 `<input id="searchInput">` 元素和 `.topbar-search` 容器
- **BREAKING**: topbar 高度从 `48px` 降低到 `40px`（按钮 34px，上下各 3px 留白）
- **BREAKING**: 侧栏 header 高度从 `48px` 同步降低到 `40px`
- **BREAKING**: 全屏相关计算（`top: 40px`、`calc(100vh - 40px)`）同步
- **BREAKING**: Ctrl+F 在编辑器关闭时不再聚焦 topbar 搜索框，改为打开搜索弹窗
- 新增居中弹窗 `#searchModal`，包含：搜索输入框、过滤器（笔记本/标签/日期）、实时结果列表
- 弹窗支持：实时搜索（防抖 200ms）、关键字高亮、键盘导航（↑↓/Enter/Esc）、遮罩点击关闭
- 复用 `add-search-filters` spec 中的过滤器实现（笔记本/标签/日期）

## Impact
- Affected specs:
  - `add-find-replace`：Ctrl+F 行为变化（编辑器关闭时从聚焦 input 改为打开 modal）
  - `move-menu-to-left`：topbar 布局变化（搜索框移除后 topbar-left 区域更紧凑）
  - `add-search-filters`：过滤功能在弹窗中复用
- Affected code:
  - `frontend/index.html`：删除 `#searchInput` 和 `.topbar-search` 容器，新增 `#searchModal` 弹窗结构
  - `frontend/src/style.css`：topbar/sidebar 高度改 40px，新增 `.search-modal` 弹窗样式（居中、遮罩、动画）
  - `frontend/src/main.js`：搜索逻辑从 `searchInput` 事件迁移到弹窗 input 事件；Ctrl+F 触发逻辑修改；焦点管理

## ADDED Requirements

### Requirement: 居中搜索弹窗
系统 SHALL 提供一个居中显示的搜索弹窗（modal），用于替代 topbar 内的搜索框。

#### Scenario: Ctrl+F 打开弹窗
- **WHEN** 编辑器关闭时（`.viewEditor` 不处于 active 状态）用户按下 Ctrl+F
- **THEN** 弹窗 `#searchModal` 立即出现并居中显示
- **AND** 搜索输入框自动获得焦点并全选已有内容
- **AND** 弹窗打开动画执行（缩放 + 淡入，0.2s）

#### Scenario: Esc 关闭弹窗
- **WHEN** 弹窗已打开且用户按下 Escape 键
- **THEN** 弹窗关闭（淡出动画 0.15s）
- **AND** 焦点恢复到触发 Ctrl+F 时的元素（如果该元素仍在 DOM 中），否则不处理

#### Scenario: 点击遮罩关闭弹窗
- **WHEN** 弹窗已打开且用户点击弹窗遮罩区域（弹窗内容之外）
- **THEN** 弹窗关闭（同 Esc 行为）

#### Scenario: 关闭弹窗时清空状态
- **WHEN** 弹窗关闭
- **THEN** 清空输入框、过滤器选择、结果列表
- **AND** 下次打开弹窗时显示空状态（提示输入关键字）

### Requirement: 搜索功能完整迁移
弹窗 SHALL 完整接管原 topbar 搜索框的所有功能。

#### Scenario: 实时搜索（防抖 + 分页）
- **WHEN** 用户在搜索输入框中输入关键字
- **THEN** 防抖 200ms 后调用 `SearchNotes` API(复用现有后端,带 notebookID 过滤)
- **AND** 搜索范围：笔记标题 + 内容 + 标签名 + 笔记本名
- **AND** 第一页请求 `pageSize` 条(取自全局 `pageSize` 变量,默认 18,用户在设置中可调为 9/18/27/36/45/54)
- **AND** 结果列表显示第一页,支持滚动加载更多(滚动到底部附近 200px 时加载下一页,追加到列表)

#### Scenario: 关键字高亮
- **WHEN** 搜索结果列表显示
- **THEN** 笔记标题和摘要中的匹配关键字使用 `<mark>` 标签包裹
- **AND** 高亮样式与原 topbar 搜索一致（黄色背景）

#### Scenario: 结果键盘导航
- **WHEN** 弹窗已打开且结果列表有项目
- **THEN** 按 ↓ 键选中下一项，按 ↑ 键选中上一项
- **AND** 选中项使用 `--hover-bg` 背景高亮
- **AND** 按 Enter 键打开当前选中项的笔记（弹窗关闭）
- **AND** 如果没有选中项，Enter 打开第一项

#### Scenario: 鼠标点击结果
- **WHEN** 用户点击结果列表中的某条笔记
- **THEN** 打开该笔记（调用 `openEditor`）
- **AND** 弹窗自动关闭

#### Scenario: 全部加载完毕
- **WHEN** 用户滚动到底部且已加载所有匹配结果
- **THEN** 弹窗结果区底部显示「共 X 条结果」提示,不再触发加载
- **AND** 防止重复请求:维护 `state.searchModalLoading` 和 `state.searchModalHasMore` 标志位

### Requirement: 多维过滤器
弹窗 SHALL 提供笔记本/标签/日期三种过滤器，可组合使用。

#### Scenario: 笔记本过滤
- **WHEN** 用户在笔记本下拉中选择「工作笔记」
- **THEN** 搜索结果仅包含该笔记本下的笔记
- **AND** 下拉支持清除选择（恢复全笔记本范围）

#### Scenario: 标签过滤
- **WHEN** 用户在标签下拉中选择「#重要」
- **THEN** 搜索结果仅包含带该标签的笔记
- **AND** 多选支持（点击多个标签，搜索包含任一标签的笔记）

#### Scenario: 日期范围过滤
- **WHEN** 用户选择日期范围「最近 7 天」
- **THEN** 搜索结果仅包含该时间范围内创建/修改的笔记
- **AND** 预设选项：今天、最近 7 天、最近 30 天、自定义范围

### Requirement: Ctrl+F 智能切换
Ctrl+F 行为 SHALL 根据编辑器状态智能切换。

#### Scenario: 编辑器内 Ctrl+F
- **WHEN** 编辑器打开时（`.viewEditor.active`）用户按下 Ctrl+F
- **THEN** 触发 CodeMirror 查找面板（`openSearchPanel`，与现有 `add-find-replace` spec 保持一致）
- **AND** 不打开搜索弹窗

#### Scenario: 编辑器外 Ctrl+F
- **WHEN** 编辑器关闭时用户按下 Ctrl+F
- **THEN** 打开搜索弹窗
- **AND** 阻止浏览器默认行为（preventDefault）

### Requirement: 移除 topbar 搜索框
topbar 内的笔记搜索框 SHALL 完全移除。

#### Scenario: 物理移除
- **WHEN** 部署此 spec
- **THEN** `<input id="searchInput">` 元素从 DOM 中删除
- **AND** `.topbar-search` 容器从 DOM 中删除
- **AND** 所有引用 `els.searchInput` 的 JS 代码全部迁移或删除

#### Scenario: topbar 高度降低
- **WHEN** 搜索框移除后
- **THEN** `#topbar { height: 40px; }`
- **AND** `.sidebar-header { height: 40px; }`
- **AND** `.editor-overlay.fullscreening { top: 40px; }`
- **AND** `.editor-panel.fullscreen { height: calc(100vh - 40px); }`
- **AND** 注释中「高度与 topbar 对齐（48px）」更新为「40px」

#### Scenario: 弹窗打开时 topbar 不变
- **WHEN** 搜索弹窗打开时
- **THEN** topbar 仍保持 40px 高度（弹窗是叠加层，不影响 topbar）
- **AND** 弹窗遮罩覆盖整个视口（含 topbar），点击 topbar 区域不关闭弹窗（避免误操作）

## MODIFIED Requirements

### Requirement: Ctrl+F 全局快捷键（原 add-find-replace）
- **旧行为**：
  - 编辑器打开时：打开 CodeMirror 查找面板
  - 编辑器关闭时：`els.searchInput.focus()`（聚焦 topbar 搜索框）
- **新行为**：
  - 编辑器打开时：打开 CodeMirror 查找面板（不变）
  - 编辑器关闭时：打开搜索弹窗 `#searchModal`（替换原聚焦 input 行为）

## REMOVED Requirements

### Requirement: topbar 永久搜索框（原 add-search-filters / redesign-ui 早期实现）
**Reason**：topbar 永久显示搜索框占用大量顶部空间，与用户追求极致紧凑 topbar 的需求冲突。搜索功能完全迁移到弹窗，弹窗是更可控、更集中的搜索交互形式。
**Migration**：
- `<input id="searchInput">` 和 `.topbar-search` 容器从 HTML 中删除
- 所有 `els.searchInput` 相关的事件监听（input/keydown）从 `main.js` 中移除或迁移到弹窗
- 搜索 API 调用（`Search` / `SearchByNotebook`）保持不变，结果展示从「过滤笔记列表」改为「弹窗结果列表」
- 关键字高亮逻辑（`highlightMatch` 等）从 topbar 搜索路径迁移到弹窗结果渲染
- 防抖参数（300ms）改为 200ms（弹窗交互更紧凑，需要更及时的反馈）
