# 会话侧栏切换按钮重定位与圆形样式改造

## 概述

将 AI 助手页会话侧栏的展开/折叠按钮，从「侧栏与对话区交界处垂直居中、14×44px 纤细竖条」改造为「对话区顶部左上角、返回按钮正下方、34×34px 圆形图标按钮」，并保持折叠/展开切换、localStorage 状态记忆、Ctrl+J 快捷键等既有功能完全不变。

## 现状分析

- **按钮 DOM 位置**：[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1128-L1130) `#aiSidebarToggle` 位于 `.ai-chat-layout` 内、`.ai-session-sidebar` 与 `.ai-chat-content` 之间（兄弟节点顺序为 侧栏 → 按钮 → 对话区，现有 `~` 兄弟选择器依赖此顺序，**不可改变**）。
- **按钮样式**：[ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2910-L2950) `.ai-sidebar-toggle` 为绝对定位（`top:50%`）、14×44px、透明背景，展开时 `left:230px` 贴侧栏右边框、折叠时 `left:0`，带 `border-left` 延续分割线；与笔记本主侧栏 [sidebar.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/sidebar.css#L650-L688) 同款。
- **交互逻辑**：[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L660-L696) 负责初始化恢复 `localStorage.ai_sidebar_collapsed`、定义全局 `window.toggleAISessionSidebar`（切换 `.collapsed` 类、更新 chevron 与 title、保存状态、展开时刷新会话列表）；[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L6323-L6324) 的 Ctrl+J 调用该全局函数。这些逻辑与按钮**视觉位置无关**，核心切换逻辑无需改动。
- **相关布局**：`.ai-chat-layout`（[ai-chat.css L2603-L2608](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2603-L2608)）为 `position:relative; overflow:hidden` 的 flex 容器；`.ai-chat-messages-inner` 当前顶部 padding 为 16px（[ai-chat.css L25-L35](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L25-L35)）。按钮改为对话区顶部左上角浮动后，首条消息可能被遮挡，需预留空间。
- **主题变量**：`--card-bg` / `--border` / `--accent` / `--text-secondary` / `--text-muted` 均已存在，14 套主题自动适配；项目已有 `@media (prefers-reduced-motion: reduce)` 惯例（[ai-chat.css L509-L514](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L509-L514) 等）。

## 变更方案

### 1. index.html —— DOM 保持不变，仅更新注释

- 按钮保留在 `.ai-chat-layout` 内、侧栏之后（保证 `~` 兄弟选择器继续生效）。
- 更新 L1127 注释，说明新定位（对话区左上角浮动圆形按钮）。

### 2. ai-chat.css —— 重写 `.ai-sidebar-toggle` 样式（L2910-L2950 区域）

- **尺寸形状**：`width:34px; height:34px; border-radius:50%`。
- **定位**：`position:absolute; top:10px`；展开时 `left:calc(230px + 16px)`（对话区左上角、距侧栏边界 16px），折叠时 `left:16px`（与返回按钮水平对齐、贴近窗口左缘）。保留 `left 0.25s ease` 过渡实现侧栏开合时的平滑水平跟随（空间连续性）。
- **外观**：`background:var(--card-bg)`、`border:1px solid var(--border)`、`color:var(--text-secondary)`、`box-shadow:0 1px 3px rgba(0,0,0,0.08)`（克制细影，避免「漂浮感」）。
- **交互**：
  - hover：背景 `color-mix(in srgb, var(--accent) 10%, transparent)`，边框与图标转 `--accent`，阴影转为 accent 色扩散光晕；
  - active：`transform:scale(0.92)` 按压回弹（沿用 `.ai-session-new-btn:active` 语言）；
  - `cursor:pointer`、`z-index:20`。
- **图标**：`.ai-sidebar-toggle svg { width:16px; height:16px; }`（CSS 覆盖 JS 内联 14px）。
- **兄弟选择器**更新为：
  - `.ai-session-sidebar:not(.collapsed) ~ .ai-sidebar-toggle { left: calc(230px + 16px); }`
  - `.ai-session-sidebar.collapsed ~ .ai-sidebar-toggle { left: 16px; }`
- 新增 `@media (prefers-reduced-motion: reduce)`：禁用该按钮的 left/transform/阴影过渡。

### 3. ai-chat.css —— 消息列表顶部预留空间（L25-L35 区域）

- `.ai-chat-messages-inner` 顶部 padding 由 `16px` 增至 `52px`（`padding: 52px 15px 96px 18px`），确保首条消息不被浮动按钮遮挡；滚动行为不受影响。

### 4. ai-chat.js —— chevron 图标尺寸微调（L663-L664）

- 两处 chevron SVG 内联 `width/height` 由 `14` 改为 `16`，与 CSS 规则一致。

## 决策与假设

- 按钮视觉位置 = **对话区（聊天内容）左上角**，而非窗口最左上：侧栏展开时窗口最左侧属于侧栏区域，放按钮会压住「会话」标题与「新建会话」按钮，不可取；折叠时按钮贴窗口左缘并与返回按钮（x=16px）对齐，满足「靠近左侧窗口」的诉求。
- 采用「绝对定位 + `left` 过渡」实现侧栏开合时的水平跟随，完全复用现有 JS 切换逻辑，不改状态机。
- 按钮始终可见（折叠时也显示），维持随时可展开侧栏的入口；点击展开时刷新会话列表（既有行为）。
- 圆形按钮采用卡片式（`--card-bg` + `--border` + 细影），区别于幽灵样式 back-btn，体现「圆形图标按钮」选择。

## 验证步骤

1. `npm run build` 重建前端资源，`wails build` 重新编译（项目硬性约定）。
2. 运行应用进入 AI 助手页验证：
   - 默认展开：按钮位于对话区左上角、返回按钮正下方；点击折叠 → 侧栏收起、按钮平滑滑至窗口左缘、chevron 变向左；
   - 再点击展开 → 侧栏展开、按钮滑回、会话列表刷新；
   - Ctrl+J 快捷键切换仍生效；
   - 重启应用后折叠状态保持（localStorage）。
3. 视觉检查：
   - 14 套主题下按钮背景/边框/图标/hover 色均协调；
   - 首条消息不被按钮遮挡；
   - hover 有 accent 光晕、按压有回弹反馈；
   - 系统「减少动态效果」开启时无动画。
