# AI 助手空对话：快捷指令卡片 + 时段问候

## Summary

在 AI 助手空对话欢迎区新增两组能力：

1. **快捷指令卡片（建议提示词 chips）**：欢迎语下方显示 4 个精选技能 chip（翻译 / 内容摘要 / 文本润色 / 深度研究），点击 = 激活对应技能 + 填入示例 prompt 到输入框（不自动发送），给 13 项技能系统做曝光引导。
2. **时段问候**：打字机欢迎语按当前小时加权选词——早/中/晚/深夜各有专属文案池，与通用池混合随机，零布局成本。

附带一个轻量入场过渡（欢迎区整体 fade-in-up，class-driven、播完即移除 class，遵循 WebView2 禁 `fill: both/forwards` 的既有教训）。

## Current State Analysis

| 位置 | 现状 |
|------|------|
| `frontend/index.html` L1179-1181 | `#aiChatWelcome` 容器内仅一个 `.ai-chat-welcome-text` 空段落 |
| `frontend/src/css/components/ai-chat.css` L2659-2689 | `.ai-chat-welcome` flex 单行居中；文字 1.35rem 粗体 + 闪烁光标 |
| `frontend/src/js/ai-chat.js` L5063-5072 `showWelcome()` | 显示欢迎容器、隐藏消息列表、启动打字机 |
| `frontend/src/js/ai-chat.js` L5087-5149 `startTypewriter()` | 24 条消息常量数组，纯随机选取，打印→停 2.5s→擦除→重选循环 |
| `frontend/src/js/ai-chat.js` L5151-5158 `stopTypewriter()` | 清定时器 + 清文本 |
| `frontend/src/js/ai-chat.js` L1416-1442 | 技能菜单点击激活路径：`activeSkills = {}`（translate 特判 `{ source: 'english', target: 'chinese' }`）→ `renderSkillChips()` → `updateSkillsMenuActiveState()` → `saveCurrentSessionConfig()` |
| `frontend/src/js/ai-chat.js` L6390-6396 | 输入框填入惯例：`inputEl.value = ...; inputEl.focus(); inputEl.dispatchEvent(new Event('input', { bubbles: true }))` |
| `frontend/index.html` L1271-1322 | 13 个技能菜单项（`data-skill` + 内联 SVG 图标 + 中文名），chips 图标从这里复制 |

空对话时欢迎区下方大片空白，无任何引导性组件。

## Proposed Changes

### 1. `frontend/index.html` — 欢迎容器加 chips 挂载点

`#aiChatWelcome` 内 `.ai-chat-welcome-text` 之后追加：

```html
<div id="aiChatWelcomeChips" class="ai-chat-welcome-chips"></div>
```

空容器、由 JS 渲染，便于将来增减条目而不动 HTML。

### 2. `frontend/src/js/ai-chat.js` — 逻辑改动（4 处）

**2a. 新增常量 `WELCOME_SUGGESTIONS`**（放在欢迎区代码块内）：

```js
const WELCOME_SUGGESTIONS = [
    { skill: 'translate',     label: '翻译',        prompt: '帮我翻译一段英文到中文' },
    { skill: 'summary',       label: '内容摘要',    prompt: '帮我把一篇长文总结成摘要' },
    { skill: 'polish',        label: '文本润色',    prompt: '帮我润色一段文字，让它更流畅' },
    { skill: 'deep_research', label: '深度研究',    prompt: '帮我深入研究一个感兴趣的话题' },
];
```

图标：新增 `WELCOME_CHIP_ICONS` 常量（SVG 字符串），从 index.html 技能菜单原样复制对应 4 个 SVG（14×14，`stroke="currentColor"`），与既有 `CHIP_ICON_SVG` 同风格。

**2b. 新增 `renderWelcomeChips()`**：

- 动态生成 4 个 `button.ai-chat-welcome-chip`（内联 SVG + label），挂到 `#aiChatWelcomeChips`
- 点击处理（严格复用既有路径）：
  1. 激活技能：`activeSkills = {}`（translate 特判 `{ source: 'english', target: 'chinese' }`）→ `renderSkillChips(); updateSkillsMenuActiveState(); saveCurrentSessionConfig();`
  2. 填入 prompt：`inputEl.value = prompt; inputEl.focus(); inputEl.dispatchEvent(new Event('input', { bubbles: true }));`
  3. 不自动发送；欢迎语保持显示，发送消息后由既有 `hideWelcome` 流程自然收起

**2c. `showWelcome()` 增加两行**：调用 `renderWelcomeChips()`；触发入场动画（`.entering` class，450ms 后移除）。

**2d. `startTypewriter()` 选词逻辑改造**：

- 新增 `TIME_GREETINGS` 分时段文案池（早 5-11 / 午后 11-18 / 晚 18-23 / 深夜 23-5，每时段 3~4 条）
- 新增 `pickWelcomeMessage()`：约 60% 概率从时段池选、40% 从原 24 条通用池选
- 初始选词与擦完重选都改调 `pickWelcomeMessage()`，原 `MESSAGES` 数组保留为通用池

### 3. `frontend/src/css/components/ai-chat.css` — 样式（3 处）

- `.ai-chat-welcome` 改纵向：`flex-direction: column; gap: 20px;`
- `.ai-chat-welcome-chips`：flex wrap 居中，gap 8px，max-width 420px
- `.ai-chat-welcome-chip`：胶囊按钮，全 CSS 变量取色（border `var(--border)`、bg `var(--bg-secondary)`，hover 转 `var(--accent)`/`var(--accent-lighter)`），active 缩放 0.97
- 入场动画 `@keyframes welcome-fade-up`（opacity 0→1 + translateY 10px→0，0.35s，不用 fill，播完 JS 移除 class）；`@media (prefers-reduced-motion: reduce)` 禁用动画与缩放

## Assumptions & Decisions

- chips 精选 4 个静态常量（非 13 技能随机）：文案可控、避免条目跳变，常量数组集中易改
- 点击只填入不发送（按用户原话"点击即填入输入框"）
- translate 默认英→中，与技能菜单激活路径一致
- chips 随 `#aiChatWelcome` 容器整体显隐，无独立定时器负担
- 纯前端改动，不涉及 Go 后端

## Verification

1. `npm run build`（frontend）+ `wails build` 重新编译
2. 空会话显示欢迎语 + 4 chips + 入场过渡；时段文案与当前时间匹配
3. 点击 chip：技能激活（输入框上方技能 chip 出现）+ prompt 填入 + 焦点聚焦，不自动发送
4. 切换有消息会话隐藏欢迎区，切回恢复
5. prefers-reduced-motion 下无动画；深浅主题下颜色正常
