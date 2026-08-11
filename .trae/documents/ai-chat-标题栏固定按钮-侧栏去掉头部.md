# AI 聊天页：标题栏固定三按钮 + 侧栏去掉"会话"头部

## Summary

简化 AI 聊天页头部结构：

1. **返回 / 展开折叠 / 新建** 三个按钮**固定**在内容区标题栏左侧，不再随侧栏折叠/展开在"侧栏头部 ↔ 标题栏"之间迁移（当前实现依赖 JS `moveHeaderActions` 移动 DOM）。
2. **删除**会话侧栏顶部的 `.ai-session-sidebar-header`（"会话"标题 + 按钮组那一层），让"搜索会话"搜索框直接顶到侧栏最上面。

最终布局：

```
标题栏:  [←返回][⇄][＋]   AI 助手(居中)   上下文大小
侧栏:    [搜索会话...]              ← 顶到最上面，无标题层
         会话列表
         [清空当前对话]
```

## Current State Analysis

* [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1104-L1139)：侧栏第一层是 `.ai-session-sidebar-header`（`#aiSessionTitle`"会话" + `.ai-session-header-actions` 按钮组 `#aiSidebarToggle`/`#aiSessionNewBtn`），其后才是搜索框；标题栏 `.view-header` 左侧是 `.view-header-left`（仅返回按钮）。

* [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L660-L713)：折叠逻辑 `window.toggleAISessionSidebar` 中调用 `moveHeaderActions(isCollapsed)`，把按钮组在 `.ai-session-sidebar-header` 与 `.view-header-left` 之间移动；另有 `sessionTitleEl`（`#aiSessionTitle`）双击脉冲绑定（L417-419）。

* [ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2575-L2697)：

  * `#viewAiChat .view-header`：grid 三列 `1fr auto 1fr`，标题居中（保留，不动）。

  * `.ai-session-sidebar-header` / `.ai-session-header-actions` / `.ai-session-sidebar-title`：头部相关样式（删除）。

  * `.ai-session-search-wrap`：`padding: 14px 10px 8px`（需要改为顶部贴合）。

## Proposed Changes

### 1. [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1107-L1122)

* **删除** `.ai-session-sidebar-header` 整个块（L1108-1119），`.ai-session-search-wrap` 成为侧栏第一个子元素，搜索框直接顶到最上面。

* **移动** `#aiSidebarToggle`、`#aiSessionNewBtn` 两个按钮到标题栏 `.view-header-left` 内（返回按钮之后），三个按钮固定排列。`.view-header-left` 内不再有 actions 容器，直接三个按钮：

```html
<div class="view-header">
    <div class="view-header-left">
        <button class="back-btn" id="aiChatBackBtn">… 返回</button>
        <button id="aiSidebarToggle" class="ai-tool-btn" title="折叠侧栏">…</button>
        <button id="aiSessionNewBtn" class="ai-tool-btn" title="新建会话">…</button>
    </div>
    <h2 class="view-title" id="aiChatTitle">AI 助手</h2>
    <div class="view-controls"><span id="aiChatContextSize" class="ai-context-size"></span></div>
</div>
```

### 2. [ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2608-L2697)

* **删除**：`.ai-session-sidebar-header`、`.ai-session-sidebar-header .ai-session-sidebar-title`、`.ai-session-header-actions` 及其 `:hover`/`:active`、`.ai-session-sidebar-title` 及其 `:hover`（L2608-2631、L2633-2656、L2684-2697）。

* **按钮样式迁移**：`.ai-tool-btn` 的 28×28 圆角/hover/active 样式改为挂在 `#viewAiChat .view-header-left .ai-tool-btn` 下（保留原样式值），确保三个按钮在标题栏内显示一致。

* **搜索框顶到最上面**：`.ai-session-search-wrap` padding 由 `14px 10px 8px` 改为 `10px 10px 8px`（或仅去掉顶部 14px 冗余，使搜索框贴近侧栏顶边）。

* `.view-header-left`（L2600-2606）保留：`justify-self: start; display: inline-flex; align-items: center; gap: 2px`。

### 3. [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L660-L713)

* **删除**：`sidebarHeader`、`viewHeaderLeft`、`headerActions` 三个引用（L663-665）和 `moveHeaderActions` 函数（L670-678）。

* **删除** `moveHeaderActions` 调用：L685（初始化折叠分支）、L690（初始化展开分支）、L701（toggle 函数内）。

* 保留：折叠 `classList.toggle('collapsed')`、图标 `panelOpenIcon`/`panelCloseIcon` 切换、`localStorage` 存取、Ctrl+J 全局函数、`loadSessionList()` 展开刷新逻辑。

* **删除** `sessionTitleEl` 相关：L20 变量声明、L236 获取、L417-419 双击脉冲绑定（`#aiSessionTitle` 已不存在）。

## Assumptions & Decisions

* 折叠侧栏后三个按钮始终显示在标题栏左侧（用户明确"不要隐藏了"），展开/折叠按钮图标随状态切换的逻辑不变。

* 侧栏去掉头部层后，搜索框直接顶到侧栏顶边；侧栏本身仍有右边框顶天立地（上一轮结构，不动）。

* 标题居中三列 grid 布局保持不变。

* Ctrl+J 快捷键绑定不变（仍调用 `window.toggleAISessionSidebar`）。

## Verification

1. `cd frontend && npm run build` 构建通过，无 lint 错误。
2. 浏览器/`wails dev` 验证：

   * 三个按钮始终固定在标题栏左侧，折叠/展开侧栏时不迁移、不消失。

   * 折叠后标题栏左侧为 返回+展开+新建，标题仍居中。

   * 侧栏顶部无"会话"标题层，搜索框紧贴侧栏顶边。

   * 折叠→展开后会话列表正常刷新。

