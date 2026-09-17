# 计划：Agent 工具列表分组分割线圆角修正 + 工具按钮右侧加箭头

## 一、Summary（目标）

1. 将「设置页」Agent 工具管理面板与「AI 助手输入框」Agent 工具浮层中，分组标签的**底部分割线（border-bottom）由圆角改为正常直线全宽**。
2. 给 AI 助手输入框的「工具」按钮（`#aiChatAgentToolsBtn`）右侧**增加一个下拉箭头**，与输入框其他带下拉的按钮（更多技能/执行审批）风格一致。

## 二、Current State Analysis（现状）

* 两处分组标签都设置了 `border-bottom + border-radius: 6px`，导致底边线两端被圆角切圆内收，视觉上像「弯曲的两端」而非正常直线分隔：

  * AI 助手浮层：`.ai-chat-agent-tools-group` — [ai-chat.css](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\ai-chat.css#L1852-L1860)

  * 设置页：`.agent-tools-mgr-group` — [settings-panel.css](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\settings-panel.css#L1672-L1684)

* AI 助手「工具」按钮当前结构仅「图标 + 文字」，缺下拉箭头；而同栏的「更多技能」「执行审批」按钮均为「图标 + 文字 + chevron」结构：

  * 工具按钮：`#aiChatAgentToolsBtn` — [index.html](d:\资源池\下水道\Dev\本地项目\jot\frontend\index.html#L1257-L1264)

  * 参考按钮（更多技能）：`#aiChatMoreSkillsBtn` — index.html L1301-L1306

## 三、Proposed Changes（改动方案）

### 改动 1：分组分割线改直线（两处 CSS）

仅修改各分组标签的 `border-radius` 为 `6px 6px 0 0`（上圆下直），并同步更新注释。这样：

* 底部分割线随 bottom 圆角归零变为**全宽直线**，不再两端内收圆角。

* 顶部两角保留圆角，hover/active 背景与 focus outline 视觉过渡不受影响。

| 文件                                               | 行                                        | 现状                         | 改为                                                   |
| ------------------------------------------------ | ---------------------------------------- | -------------------------- | ---------------------------------------------------- |
| `frontend/src/css/components/ai-chat.css`        | L1852-1860（`.ai-chat-agent-tools-group`） | `border-radius: 6px;` + 注释 | `border-radius: 6px 6px 0 0;` + 注释说明「上圆下方：底边分割线全宽直线」 |
| `frontend/src/css/components/settings-panel.css` | L1672-1684（`.agent-tools-mgr-group`）     | `border-radius: 6px;` + 注释 | 同上                                                   |

### 改动 2：工具按钮右侧加箭头（index.html）

在 `#aiChatAgentToolsBtn` 的 `<span>工具</span>` 之后追加一个与「更多技能/审批」一致的 chevron（10×10，`polyline points="6 9 12 15 18 9"`）：

```html
<button class="ai-chat-toolbar-btn" id="aiChatAgentToolsBtn" title="Agent 工具" aria-expanded="false" aria-controls="aiChatAgentToolsDropdown">
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>
    <span>工具</span>
    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
</button>
```

说明：`gap: 4px` 已由 `.ai-chat-toolbar-btn` 提供，无需额外 CSS；遵循项目惯例（chevron 静态不随开合旋转，与「更多技能/审批」一致）。

## 四、Assumptions & Decisions（假设与决策）

* 分割线「正常化」采用**保留顶部圆角、底部归零**的方案，而非整体去圆角，以同时满足直线分隔与 hover/focus 圆角反馈。

* 箭头取静态 chevron，不做开合旋转（与同栏「更多技能/审批」按钮保持一致）。

* 不改 JS、不改后端、不加新 CSS 规则。

## 五、Verification（验证）

1. 修改后执行 `npm run build` 重新打包前端（项目规范：CSS/HTML 改动必须重建）。
2. `wails build` 重新生成二进制后，视觉验证：

   * 设置页 Agent 工具面板、AI 助手输入框工具浮层：分组底部分割线为全宽直线、两端无圆角内收。

   * 输入框「工具」按钮右侧显示下拉箭头，与「更多技能/审批」按钮对齐。

