# 修复冲突弹窗条目移除/上移动画不丝滑

## Summary

当前冲突弹窗中处理条目时（覆盖/跳过），`el.remove()` 直接删除 DOM + `renderItems()` 用 `innerHTML = ''` 全量重建，导致移除无动画、剩余条目瞬间跳变。需要改为：先折叠移除条目，再用 FLIP 动画平滑移动剩余条目。

## Current State

**`handleItem`** 流程（[main.js#L8951](frontend/src/main.js#L8951)）：
1. 弹确认框 → 调 API → `el.remove()` 直接删 DOM → `renderItems()` 全量重建
2. 问题：无过渡动画，视觉突兀

**`renderItems`**（[main.js#L8916](frontend/src/main.js#L8916)）：
- `listEl.innerHTML = ''` 清空 → 重新 createElement 所有条目
- 无位置记忆，无法做平滑位移

## Proposed Changes

### 1. CSS — 新增折叠退出动画

**文件**: `frontend/src/css/components/modals.css`

- 新增 `.import-conflict-item.collapsing` 状态：
  - `transition: max-height 0.25s ease, opacity 0.2s ease, padding 0.25s ease, margin 0.25s ease`
  - `max-height: 0; opacity: 0; padding-top: 0; padding-bottom: 0; overflow: hidden`
- `.import-conflict-item` 默认设置 `max-height: 200px; transition: max-height 0.25s ease`（足够容纳单行条目）

### 2. JS — 重写 handleItem 移除逻辑

**文件**: `frontend/src/main.js`

重写 `handleItem` 中 `el.remove()` + `renderItems()` 部分：

```
1. el.classList.add('collapsing')  // 触发折叠动画
2. el.style.maxHeight = el.offsetHeight + 'px'  // 锁定当前高度
3. requestAnimationFrame → el.style.maxHeight = '0'  // 开始折叠
4. 等待 transitionend（约 250ms）
5. el.remove()
6. FLIP 动画重建剩余条目：
   a. 记录所有剩余 .import-conflict-item 的当前位置 (getBoundingClientRect)
   b. 从 items 数组中移除已处理项
   c. 调用 renderItems() 重建 DOM
   d. 对新 DOM 中每个条目：记录新位置，用旧位置差值做 translateY 反向偏移
   e. 一次性触发 translateY(0) + transition，实现平滑上移
```

### 3. JS — renderItems 适配

`renderItems` 不需要大改，只需确保重建后返回新增的 DOM 元素列表供 FLIP 动画使用。可以改为返回 `listEl.children` 或在调用前/后做对比。

## Affected Files

| 文件 | 改动 |
|---|---|
| `frontend/src/css/components/modals.css` | 新增 `.collapsing` 折叠动画样式 |
| `frontend/src/main.js` | 重写 `handleItem` 移除逻辑，添加 FLIP 动画 |

## Assumptions & Decisions

- 折叠动画时长 250ms，与现有 `.import-conflict-item` 的 `conflictItemEnter 0.2s` 保持一致节奏
- FLIP 动画使用 `requestAnimationFrame` + `transitionend`，不依赖第三方库
- 批量操作（全部覆盖/全部跳过）不需要逐条动画，直接重建即可（条目太多逐条动画会很慢）

## Verification

1. 导入触发冲突 → 弹窗中点击单个"覆盖"或"跳过" → 条目应折叠消失，剩余条目平滑上移
2. 批量操作 → 直接重建，无逐条动画
3. 最后一个条目处理完 → 弹窗关闭动画正常
4. 快速连续点击多个条目 → 动画不卡顿、不重叠
