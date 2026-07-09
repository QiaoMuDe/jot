# 修复灯箱 ESC 关闭笔记的问题

## 总结

灯箱打开时按 ESC，会关闭笔记编辑器而非灯箱。根因是全局 `handleKeyboardNavigation`（先注册）和灯箱本地 ESC 监听器（后注册）都在 `document` 的 bubble 阶段，全局 handler 先触发。当前方案用 DOM 查询 `document.querySelector('.image-lightbox')` 在全局 handler 中检查，但不可靠。改为状态标志 + 修复监听器泄漏。

## 当前状态分析

### 相关代码位置

| 位置 | 内容 | 角色 |
|------|------|------|
| [`main.js:5080`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L5080) | `document.addEventListener('keydown', handleKeyboardNavigation)` | 全局键盘处理（先注册） |
| [`main.js:5242-5267`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L5242-L5267) | 全局 ESC 处理块 | 检查灯箱存在、退出全屏、关闭编辑器 |
| [`main.js:3531-3533`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3531-L3533) | 灯箱 ESC 监听器 | 关闭灯箱（后注册） |

### 问题根因

1. **事件顺序**：`handleKeyboardNavigation` 在页面加载时注册，灯箱的 ESC 监听器在点击图片时注册。两者都在 `document` 的 bubble 阶段，全局 handler 先触发。
2. **DOM 查询不可靠**：`document.querySelector('.image-lightbox')` 看似正确但用户反馈无效。改用 `window.__lightboxOpen` 布尔标志更可靠。
3. **监听器泄漏**：`overlay.addEventListener('remove', ...)` 无效——DOM 元素没有 `remove` 标准事件，`onKey` 监听器在灯箱关闭后仍残留。

## 改动方案

### 1. `main.js` — 灯箱代码改用状态标志

**文件**：[`main.js:3494-3535`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3494-L3535)

**What**: 在全局设置 `window.__lightboxOpen = true/false` 标志，替代 DOM 查询。

**Why**: 布尔标志比 DOM 查询更可靠，避免 `querySelector` 因各种原因找不到元素的问题。

**How**:
- 创建灯箱 overlay 时：`window.__lightboxOpen = true`
- `close()` 函数中：`window.__lightboxOpen = false`
- 移除无效的 `overlay.addEventListener('remove', ...)` 清理逻辑
- 在 `close()` 中主动 `document.removeEventListener('keydown', onKey)` 清理监听器

### 2. `main.js` — 全局 ESC handler 改用标志检查

**文件**：[`main.js:5259-5263`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L5259-L5263)

**What**: 将 `document.querySelector('.image-lightbox')` 改为 `window.__lightboxOpen`。

**Why**: 消除 DOM 查询的不确定性，直接使用可靠的内存标志。

**How**:
```javascript
// 替换
if (document.querySelector('.image-lightbox')) {
// 为
if (window.__lightboxOpen) {
```

### 3. `main.js` — 移除多余的 stopPropagation

**文件**：[`main.js:3531`](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3531)

**What**: 去掉 `ke.stopPropagation()`，因为事件在 `handleKeyboardNavigation` 中已被标志拦截，灯箱 handler 不再需要干预事件传播。

**Why**: 简化代码，`stopPropagation()` 对同级 `document` 上的其他 listener 无实际阻止效果。

## 决策记录

- **状态标志优于 DOM 查询**：避免 DOM 查询的不可靠性（查询时机、元素状态等问题）
- **使用 `window.__lightboxOpen` 命名空间**：简单、全局可访问，不需要引入复杂的状态管理
- **不修改事件阶段或顺序**：保持事件监听器在 bubble 阶段的注册顺序不变，通过标志拦截更简洁

## 验证步骤

1. `npx vite build` — 前端构建通过
2. 运行时验证：
   - 打开一个 .md 笔记 → 预览模式 → 点击图片 → 灯箱打开
   - 按 ESC → 灯箱关闭，笔记不关闭
   - 再按 ESC → 笔记关闭
3. 验证不影响其他 ESC 行为：
   - 编辑器全屏模式 → ESC 退出全屏
   - 无灯箱时 → ESC 正常关闭笔记
