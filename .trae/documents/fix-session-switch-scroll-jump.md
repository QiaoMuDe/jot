# 修复 AI 会话切换消息列表滚动跳跃

## 问题描述

切换有大量消息的 AI 会话时，消息列表先加载到底部（贴近输入框和操作栏），然后突然"跳跃"到操作栏上方露出间距。视觉上存在明显的滚动跳跃。

## 根因分析

`switchSession()` 中采用**分块渲染**（每批 5 条消息 + `yield` 让浏览器绘制），渲染完成后调用 `scrollToBottom()`（通过 `requestAnimationFrame` 推迟到下一帧执行）：

- **分块渲染期间**：`scrollTop = 0`，消息从列表顶部开始填充，最后一批消息恰好出现在靠近输入框的位置
- **渲染完成后**：`scrollToBottom()` 的 rAF 回调设置 `scrollTop = scrollHeight`，`padding-bottom: 72px` 露出，消息整体上移
- 浏览器在中间状态和最终状态之间切换，产生视觉跳跃

## 修复方案

**原则**：切换会话时消息直接以最终状态显示，不让浏览器绘制中间状态。

**修改文件**：`frontend/src/js/ai-chat.js` - `switchSession()` 函数

**具体改动**：
1. 去掉分块渲染的 `yield`（`await new Promise(r => setTimeout(r, 0))`），所有消息一次性同步渲染
2. 去掉循环内的渐进式滚动
3. 所有消息渲染完成后，**同步**设置 `scrollTop = scrollHeight`（临时禁用 `scroll-behavior: smooth`）
4. 不再调用 `scrollToBottom()`（避免其内部的 rAF 延迟）
5. 浏览器只绘制一次最终状态

**不受影响的功能**：
- 流式输出（正常对话）：`addMessage` 的 `skipScroll=false` 场景继续使用原有的 `scrollToBottom()`（带 rAF）
- 新建空会话、清空对话等操作

## 影响文件

| 文件 | 改动 |
|------|------|
| `frontend/src/js/ai-chat.js` | `switchSession()` 中消息渲染由"分块+yield+渐进滚动"改为"一次性渲染+同步滚动" |
| `AGENTS.md` | 更新 AI 对话模块描述、多会话切换架构说明、spec 列表 |
