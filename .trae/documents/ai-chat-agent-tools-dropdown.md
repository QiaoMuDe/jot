# AI 助手模块内嵌「Agent 工具选择」方案

## 一、概述

在 AI 聊天输入区的底部快捷工具栏（`.ai-chat-toolbar`）中，于「深度思考」开关右侧新增一个常显的「工具」图标（工具钳图标，不带工具数量文案）。点击展开一个浮层下拉，复用设置页「Agent 工具」管理面板的**同一份全局状态**（`agentToolsMeta` / `agentToolsDisabled` / 设置键 `ai_agent_tools_disabled`），在其中勾选/取消 Agent 工具，**每勾选一项立即保存（方案 A）**，关闭下拉时汇总提示本次改动。该图标仅当当前模式不是 `chat`（即 `agent` / `plan`）时显示。

## 二、现状分析（关键事实，均来自 Phase 1 探查）

1. **工具数据与保存全部位于 `main.js` 模块作用域**（不导出、不在 window 上）：
   - `agentToolsMeta`（来自 `App.GetAgentTools()`，含 `Name/Label/Enabled/PlanOnly/AlwaysOn`）
   - `agentToolsDisabled`（对应设置键 `ai_agent_tools_disabled`，`saveSettings()` 持久化）
   - `agentToolsChanges`（本次会话内启停变更暂存，用于关闭时汇总提示）
   - `createAgentToolRow(tool)`（`main.js` L10119）：渲染单个工具行，处理 `PlanOnly`/`AlwaysOn` 只读保护、勾选时改 `agentToolsDisabled` + 记入 `agentToolsChanges` + **立即 `saveSettings()`**，并调用 `updateAgentToolsButtonText()` / `updateSelectAllCheckboxState()`（两者均为 null-safe，可安全复用）
   - 关闭汇总逻辑内嵌于 `closeAgentToolsMgrList()`（L9903-9911，读取 `agentToolsChanges` 后弹 toast 并清空）
2. **AI 聊天模块 `ai-chat.js`**：`currentMode`（'chat'|'agent'|'plan'）是模块内变量，且 `syncModeToggle()`（L7321）在全部 4 处 `currentMode` 赋值点均被调用（L891 模式按钮点击、L1916 会话加载、L2515 恢复、L3937 plan→agent）。因此它可作模式→可见性的唯一同步点。
3. **跨模块通信**：`main.js` import `ai-chat.js`（单向），故 ai-chat 若要通知 main 需走 window 回调（既有先例：`window.__aiStreaming`、`window.showNotification?.()`）。采用 `window.__setAiChatAgentToolsVis?.(visible)`，结合可选链保证初始化顺序无虞。
4. **样式复用**：`.ai-agent-tools-item`、`.ai-agent-tools-name`、`.ai-agent-tools-desc`、`.plan-only-hint`、`.always-on-hint` 等类名是**全局**定义于 `settings-panel.css`（未被父级作用域限制），可在聊天下拉中直接复用。下拉面板定位可完全照搬 `.ai-chat-skills-dropdown`（`ai-chat.css` L1579-1636：`position:absolute; bottom:calc(100%+4px); right:0; ...` 含 open/closing 动效）。
5. **输入框内右侧已有「优化」按钮且锚定右侧**，工具图标不与其冲突（本方案放工具栏，不放入输入框）。

## 三、变更方案

### 1. `frontend/index.html` —— 新增按钮与下拉容器

在 `#aiChatSearchToggle` 结束的 `</div>`（当前 L1229）与 `ai-chat-skills-wrap`（L1231）之间插入：

```html
<!-- Agent 工具选择（仅 Agent/Plan 模式显示） -->
<div class="ai-chat-agent-tools-wrap" id="aiChatAgentToolsWrap">
    <button class="ai-chat-toolbar-btn" id="aiChatAgentToolsBtn"
            title="Agent 工具" aria-expanded="false" aria-controls="aiChatAgentToolsDropdown">
        <!-- 工具钳 SVG（wrench） -->
    </button>
    <div class="ai-chat-agent-tools-dropdown" id="aiChatAgentToolsDropdown"></div>
</div>
```

- 按钮无文本（不显示工具数量），仅图标；`title="Agent 工具"`。
- 默认不设 `hidden`（默认模式为 `agent` → 应显示），可见性由 JS 按模式控制。

### 2. `frontend/src/css/components/ai-chat.css` —— 下拉面板与按钮样式

在 `.ai-chat-skills-wrap` 块之后新增：

- `.ai-chat-agent-tools-wrap { position: relative; flex-shrink: 0; }`（对齐 skills-wrap，L1575-1577）
- `.ai-chat-agent-tools-dropdown`：**整体照搬** `.ai-chat-skills-dropdown`（L1579-1636）的定位/外观/open/closing 动效，仅改 `min-width: 220px; max-height: 320px`，内部复用现有工具行样式。
- 按钮沿用既有 `.ai-chat-toolbar-btn`（L2902 起），无需新样式；`.is-locked` / `.is-shaking`（L1471-1489）自动生效。

说明：下拉内部 header 与工具行复用 `settings-panel.css` 中全局类（`.agent-tools-mgr-header`、`.agent-tools-mgr-title`、`.agent-tools-mgr-select-all`、`.ai-agent-tools-item` 等）；如行内边距在浮层内观感偏挤，仅微调 `.ai-chat-agent-tools-dropdown` 内对应类，不动 settings 原样式。

### 3. `frontend/src/main.js` —— 下拉逻辑（数据与保存归属处）

在既有「Agent 工具管理面板」绑定块（L2641-2673）附近新增 `initChatAgentTools()`，并在初始化流程调用：

1. **取元素**：`aiChatAgentToolsBtn`、`aiChatAgentToolsDropdown`、`aiChatAgentToolsWrap`。
2. **暴露模式可见性回调**（供 ai-chat.js 调用）：
   ```js
   window.__setAiChatAgentToolsVis = (visible) => {
       if (aiChatAgentToolsWrap) aiChatAgentToolsWrap.hidden = !visible;
   };
   window.__setAiChatAgentToolsVis(true); // 默认 agent 模式可见
   ```
3. **按钮点击展开/收起下拉**：仿设置页 `aiAgentToolsBtn`（L2644-2659）——点击 `stopPropagation` 后切换展开；展开时渲染列表，同步 `aria-expanded` 与按钮 `open` 态（`.ai-chat-toolbar-btn.open` 旋转箭头，若需要）。
4. **渲染下拉**：仿 `renderAgentToolsMgrList()`（L10050 起）：header（全选 checkbox `toggleSelectAllTools` + 标题「Agent 工具」+ 「关闭」按钮）+ `agentToolsMeta.forEach(createAgentToolRow)`。复用同一模块作用域中的 `agentToolsMeta`/`agentToolsDisabled`/`agentToolsChanges`/`toggleSelectAllTools`/`updateSelectAllCheckboxState`/`createAgentToolRow`。
5. **关闭汇总**：新增 `closeChatAgentToolsList()`，收起动画（可复用 L9931-9950 的 opacity/scaleY/max-height 思路）后，把 `closeAgentToolsMgrList` 内那段「汇总 toast」逻辑抽成公共函数 `reportAgentToolsChanges()` 并复用（读取 `agentToolsChanges` 拼「工具配置已保存：禁用 X 个，启用 Y 个」，随后清空）。
6. **外点关闭 + ESC 关闭**：仿 L2661-2672（document click 排除目标、Escape 关闭）。
7. **流式锁态**：把工具按钮加入 `ai-chat.js` 的 `setToggleLocked`（见下）与下拉收起。

### 4. `frontend/src/js/ai-chat.js` —— 模式可见性 + 流式锁态

1. **`syncModeToggle()`（L7321-7326）末尾追加**：
   ```js
   window.__setAiChatAgentToolsVis?.(currentMode !== 'chat');
   ```
   因 `syncModeToggle` 在全部 4 处 mode 变更点被调用，chat→隐藏、agent/plan→显示自动同步。
2. **`setToggleLocked()`（L7264-7269）追加**：`agentToolsBtn?.classList.toggle('is-locked', locked);`（新查 `aiChatAgentToolsBtn`），并在 `locked` 分支（L7300 起「锁定同时收起已展开的下拉菜单」区块）追加收起 `aiChatAgentToolsDropdown`。

## 四、假设与决策记录

- **保存时机 = 方案 A（即时保存）**：勾选即 `saveSettings()`，关闭时仅汇总 toast（与设置页行为完全一致）。用户已确认。
- **图标常显，仅在 chat 模式隐藏**：不显示工具数量（用户已确认，省去 `已启用 N/M` 维护）。
- **复用全局状态**：聊天下拉与设置页操作同一份 `agentToolsMeta`/`agentToolsDisabled`、同一存储键 `ai_agent_tools_disabled`，改任一入口全局生效。
- **跨模块通信走 window 回调**（沿用 `window.__aiStreaming`/`showNotification` 先例），避免引入新事件机制。
- **下拉浮层防裁剪**：下拉沿 `.ai-chat-skills-dropdown` 先例锚定在 wrap 内（底部上弹出、right:0），位于 composer 区不随 `#mainContent` 滚动，故不受内部滚动容器裁剪影响。
- **不在输入框内放图标**：用户最终改为放工具栏「深度思考」右侧，避开与右侧「优化」按钮/光标动画的定位竞争。

## 五、验证步骤

1. **静态检查**：`frontend` 目录运行 `.\node_modules\.bin\eslint.cmd src/main.js src/js/ai-chat.js`，0 error。
2. **wails dev 实测**：
   - Agent 模式：工具图标可见；展开下拉能列出工具，勾选/取消即时生效，`PlanOnly`/`AlwaysOn` 项呈只读抖动提示，关闭时弹「工具配置已保存」汇总。
   - 切到 Chat 模式：工具图标隐藏；切回 Agent/Plan：图标重新显示。
   - 打开聊天下拉勾选后，进设置页「Agent 工具」面板，状态一致（双向同步）。
   - 流式回复中：工具按钮 `is-locked` 置灰、点击抖动提示，下拉被收起。
   - 外点 / ESC 均能关闭下拉。
   - 提示词、占位符、光标不受影响。
3. **回归**：设置页既有「Agent 工具」面板开关、汇总提示行为不变。