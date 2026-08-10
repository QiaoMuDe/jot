# AI 输入区一体化「输入坞」重构计划

## Summary

将 AI 对话页底部现有的「工具栏行 + 输入框行」两层分离结构，重构为**单张一体式圆角容器（输入坞）**：上层为多行输入区，下层为内嵌横向工具行，最右为圆形发送按钮；技能引用 / 笔记引用 / 上传文件 / 追问等附属条悬浮在容器上方。全部功能（控件 id、下拉、事件绑定、JS 逻辑）保持不变，仅重构 HTML 结构与 CSS 样式。

## 决策记录（用户已确认）

1. **工具行平铺**：+ 添加、Chat/Agent 快速切换、模型选择、深度思考、联网搜索、卡片召回、更多技能（收纳翻译/解题/润色等全部技能）。项目无生图/视频/PPT 功能，不新增。
2. **右侧操作按钮**：仅保留发送按钮（accent 圆形，流式中原位变停止）。不加语音按钮（项目无语音功能）。
3. **边框配色**：全部用现有主题变量（`--border` / `--accent` / `--card-bg`），14 主题自动适配，不改 variables.css。

## 当前状态分析

**HTML（[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1172-L1340)）现有结构：**
```
#aiChatInputArea (.ai-chat-input-area)
 ├─ #aiChatFollowUpBar（追问栏，隐藏）
 ├─ .ai-chat-toolbar（工具栏行）
 │   ├─ .ai-chat-add-wrap（+ 添加，含 border-right 竖线）
 │   ├─ .ai-chat-mode-switch（Chat/Agent 分段）
 │   ├─ .ai-chat-model-select（模型选择）
 │   ├─ .ai-chat-search-toggle（深度思考，迷你 switch）
 │   ├─ .ai-chat-sources-wrap（联网搜索，迷你 switch + 箭头）
 │   ├─ .ai-chat-recall-wrap（卡片召回，迷你 switch + 箭头）
 │   └─ .ai-chat-skills-wrap（更多技能，margin-left:auto）
 ├─ #aiChatRefBar / #aiChatSkillBar / #aiChatFileBar（chips 附属条，隐藏）
 └─ .ai-chat-input-row
     ├─ .ai-chat-input-wrap（textarea + 悬浮优化按钮）
     ├─ #aiChatSendBtn（38px 圆形 accent）
     └─ #aiChatStopBtn（36px 红色，流式中显示）
```

**关键样式（[ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)）：**
- `.ai-chat-input-area`（L1144）：column，`padding: 8px 48px 16px`，`border-top`，max-width 1600px 居中
- `.ai-chat-toolbar`（L1157）：flex 行，gap 12px，wrap
- `.ai-chat-mode-switch`（L1167）：分段控件（`--hover-bg` 底 + border + 激活 accent）
- `.ai-chat-search-toggle`（L1221）：图标+文字+迷你 switch（`.ai-chat-toggle-switch` L1244 / `.ai-chat-toggle-knob`）
- `.ai-chat-input-wrap`（L2016）：`position: relative`；`.ai-chat-input`（L2123）：`--input-bg` + border + radius-md
- `.ai-chat-send-btn`（L2149）：38px 圆形 accent；`.ai-chat-stop-btn`（L2178）：36px 红
- 附属条：`.ai-chat-followup-bar`（L2831）、`.ai-chat-ref-bar`（L2874）、`.ai-chat-skill-bar`（L1817）、`.ai-chat-file-bar`（L2997）——均为「flex 容器 + chips 胶囊」结构，chips 用 accent 8% 底 + accent 18% 边框

**JS 绑定（[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js)）：** 全部基于 id（aiChatInput / aiChatSendBtn / aiChatStopBtn / aiChatModeSwitch / aiChatSearchToggle / aiChatSearchSourcesBtn / aiChatCardRecallToggle / aiChatMoreSkillsBtn / aiChatModelTrigger 等）+ 少量 class（`.is-loading` / `.is-typing` / `.is-single-line` / `.active`），**重构后所有 id 与逻辑 class 保留，JS 零改动**。

## 目标布局

```
#aiChatInputArea (column)
 ├─ #aiChatFollowUpBar（追问栏，悬浮条样式，容器上方 6px）
 ├─ #aiChatRefBar / #aiChatSkillBar / #aiChatFileBar（chips 附属条，容器上方 6px，chips 加悬浮阴影）
 └─ .ai-chat-composer（一体圆角容器：border var(--border) / radius 14px / bg var(--card-bg)）
     ├─ .ai-chat-composer-main（flex row, align-items flex-end, gap 10px）
     │   ├─ .ai-chat-input-wrap（textarea 透明化：background transparent、border none）
     │   └─ #aiChatSendBtn / #aiChatStopBtn（40px 圆形，同位置互斥切换）
     └─ .ai-chat-composer-tools（内嵌工具行，gap 6px，flex-wrap）
         ├─ .ai-chat-add-wrap（+ 添加，保留 border-right 竖线）
         ├─ .ai-chat-mode-switch（Chat/Agent 分段，样式不变）
         ├─ .ai-chat-model-select（模型▾）
         ├─ #aiChatSearchToggle（深度思考：按钮式，active 用 accent 底）
         ├─ .ai-chat-sources-wrap（联网搜索▾：按钮式）
         ├─ .ai-chat-recall-wrap（卡片召回▾：按钮式）
         └─ .ai-chat-skills-wrap（更多技能▾，不再 margin-left:auto，随行排布）
```

## 具体改动

### 1. index.html（L1179-1339 区域重构）

- **保留不动**：`#aiChatInputArea` 开标签、`#aiChatFollowUpBar`、三个 chips bar（`#aiChatRefBar` / `#aiChatSkillBar` / `#aiChatFileBar`）
- **重构**：删除 `.ai-chat-toolbar` 的独立行地位与 `.ai-chat-input-row`，替换为：
  ```html
  <div class="ai-chat-composer">
    <div class="ai-chat-composer-main">
      <div class="ai-chat-input-wrap" id="aiChatInputWrap">
        <textarea id="aiChatInput" class="ai-chat-input" ...></textarea>
        <button id="aiChatPolishBtn" class="ai-chat-polish-btn" ...>...</button>
      </div>
      <button id="aiChatSendBtn" class="ai-chat-send-btn" ...>...</button>
      <button id="aiChatStopBtn" class="ai-chat-stop-btn" style="display:none;" ...>...</button>
    </div>
    <div class="ai-chat-composer-tools">
      <!-- 原有 7 个控件原样移入，id/结构零改动，顺序：+ → Chat/Agent → 模型 → 深度思考 → 联网 → 卡片 → 更多 -->
    </div>
  </div>
  ```
- **所有下拉菜单**（`.ai-chat-add-dropdown` / `.ai-chat-model-dropdown` / `.ai-chat-search-sources-dropdown` / `.ai-chat-recall-dropdown` / `.ai-chat-skills-dropdown`）**随各自 wrap 原样移动**，内部结构不变
- placeholder 保持现状（`输入消息...`），不添加语音提示文案

### 2. ai-chat.css

| 目标 | 改动 |
|---|---|
| `.ai-chat-composer`（新增） | `border: 1px solid var(--border); border-radius: 14px; background: var(--card-bg); padding: 10px 12px 12px; transition: border-color .2s, box-shadow .2s`；textarea 聚焦时通过 `.ai-chat-composer:focus-within` 提升：`border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent)` |
| `.ai-chat-composer-main`（新增） | `display: flex; align-items: flex-end; gap: 10px;` |
| `.ai-chat-input`（改） | 背景 `transparent`、`border: none`、`border-radius: 0`、`padding: 6px 8px`、`min-height: 44px; max-height: 150px`、font-size `.9rem`；focus 样式移除边框/阴影（由容器 focus-within 承担） |
| `.ai-chat-input-wrap.is-loading::after`（改） | 圆角从 radius-md 改为 10px |
| `.ai-chat-polish-btn`（改） | 定位微调（`right: 8px; top: 6px`），其余逻辑不变 |
| `.ai-chat-composer-tools`（新增，复用 `.ai-chat-toolbar` 样式） | `display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-top: 6px; padding: 8px 4px 0 4px; border-top: 1px solid var(--border);`（工具行与输入区用细线分隔，呼应"内嵌底部工具行"） |
| `.ai-chat-search-toggle`（改） | 保持"图标+文字"形态；**隐藏迷你 switch**：`.ai-chat-toggle-switch { display: none; }`（DOM 保留，激活逻辑仍走 `.active` 类）；激活态：`.active { color: var(--accent); background: color-mix(in srgb, var(--accent) 10%, transparent); }` |
| `.ai-chat-mode-option`（改） | 保持分段样式不变，仅 font-size 不变（工具行内自然适配） |
| `.ai-chat-skills-wrap`（改） | 移除 `margin-left: auto`，随行排布 |
| `.ai-chat-send-btn` / `.ai-chat-stop-btn`（改） | 尺寸统一 40px 圆形；`flex-shrink: 0; align-self: flex-end;`（与 textarea 底部对齐） |
| 附属条悬浮（改） | `.ai-chat-followup-bar` / `.ai-chat-ref-bar` / `.ai-chat-skill-bar` / `.ai-chat-file-bar`：`margin-bottom: 6px;`（与容器上方间距）；chips 增加悬浮感：`box-shadow: 0 1px 2px var(--shadow);` |
| Agent 模式隐藏开关 | `.ai-chat-toolbar .ai-chat-mode-hidden { display: none !important; }` 选择器改为同时覆盖 `.ai-chat-composer-tools .ai-chat-mode-hidden`（或直接改为 `.ai-chat-composer-tools` 命名，保留原规则并追加） |

### 3. ai-chat.js

**零改动**。所有功能逻辑（发送/停止切换、模型选择、深度思考/联网/召回开关、技能下拉、添加下拉、优化按钮、加载遮罩、打字光标、Agent 模式隐藏开关）均基于 id 与既有 class，结构与 id 未变。

## 假设与说明

1. 工具行内 `+` 按钮保留 `border-right` 竖线（呼应参考设计"细竖线分隔首个按钮"），并作为"添加"与其它工具的自然分组。
2. 迷你 switch（拨动开关）视觉移除，但 DOM 保留、`.active` 切换逻辑不变——避免 JS 改动，且视觉统一为按钮激活态（accent 文字+浅底，符合"颜色+图标双重状态表达"）。
3. 追问栏 / 三个 chips 条保持在容器**上方**（悬浮感通过间距 + chips 阴影实现，不采用绝对定位浮层，避免遮挡与 z-index 问题）。
4. 容器 `focus-within` 聚焦态替代原 textarea 独立边框聚焦，焦点视觉更聚焦整卡。
5. 主题：全部使用现有语义变量，14 主题零改动。

## 验证

1. `cd frontend && npm run build` 通过（无 JS/CSS 语法错误）
2. `wails dev` 手动验证：
   - 输入区呈现一体圆角容器：textarea 无内边框、容器聚焦时 accent 描边 + 光环
   - 工具行 7 项按钮可点：+ 添加（引用/上传）、Chat/Agent 切换、模型切换、深度思考/联网/卡片召回开关激活态变色、更多技能下拉
   - 发送按钮空输入禁用 → 有输入启用 → 流式中原位变红色停止按钮 → 结束后恢复
   - 引用笔记 / 上传文件 / 技能 chips 显示于容器上方、间距合适、可删除
   - Agent 模式下深度思考/联网/卡片召回按钮隐藏，容器布局不塌陷
   - 14 主题下容器配色正常（浅色/深色均可见边框与聚焦态）
   - 减少动态效果模式下无异常动画
