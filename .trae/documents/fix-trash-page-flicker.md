# 修复回收站页面闪烁问题

## 摘要

修复回收站页面进入时"闪 3 下"的问题。两个独立 bug 叠加导致：视图入场与数据加载不同步（2 次闪）+ `trashEnter` 动画自身抖动（1 次闪）。

## 变更 1：修复 trashEnter 动画（同卡片入场方案）

**文件**: `frontend/src/css/animations.css` — 重写 `trashEnter` 关键帧

**改动**:
```css
/* 之前：3 帧 + 弹性曲线 + forwards，导致抖动和阻塞 */
@keyframes trashEnter {
    0%   { opacity: 0; transform: translateY(16px) scale(0.95); }
    60%  { opacity: 1; transform: translateY(-3px) scale(1.01); }
    80%  { transform: translateY(1px) scale(0.998); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}

/* 之后：2 帧 + 简单缓出 + backwards，和卡片入场一致 */
@keyframes trashEnter {
    0%   { opacity: 0; transform: translateY(12px) scale(0.95); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}
```

**文件**: `frontend/src/js/trash-page.js:324` — 修改 JS 动画属性

**改动**:
```javascript
// 之前：
item.style.animation = `trashEnter 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) forwards`;

// 之后：
item.style.willChange = 'transform, opacity';
item.style.animation = `trashEnter 0.35s cubic-bezier(0.16, 1, 0.3, 1) backwards`;
```

**文件**: `frontend/src/css/components/main-content.css:560` — 移除 `.trash-item` 的 `opacity: 0`

**原因**: `backwards` 在延迟期间自动应用 0% 关键帧的 `opacity: 0`，CSS 的 `opacity: 0` 不再是必需的。移除后避免与 `backwards` 冲突。

**改动**:
```css
/* 之前 */
.trash-item {
    ...
    opacity: 0;
}

/* 之后 */
.trash-item {
    ...
    /* opacity 由 animation backwards 控制 */
}
```

## 变更 2：修复视图入场与数据加载不同步

**文件**: `frontend/src/main.js:653-654` — 在 `switchView` 中修改回收站视图的逻辑

**问题**: `showTargetView()` 中先让视图可见（`.active`），然后添加 `.view-enter` 淡入动画，同时异步加载数据。导致视图在数据加载完成前闪现空白。

**方案**: 对回收站视图，**跳过 `.view-enter` 淡入动画**。因为回收站视图的 items 已经有自己的 `trashEnter` 交错入场动画，不需要额外的视图级淡入。

**改动**:
```javascript
// 在 showTargetView 中，修改 requestAnimationFrame 回调：
case 'trash':
    loadTrashNotes();
    // 回收站视图使用 items 自身的 trashEnter 动画，无需视图级淡入
    break;
```

对应的修改在 `requestAnimationFrame` 回调中，跳过对 `trash` 视图添加 `.view-enter` 类：

```javascript
requestAnimationFrame(() => {
    // 回收站视图跳过 view-enter 淡入，避免与异步数据加载产生闪烁
    if (view !== 'trash') {
        targetView.classList.add('view-enter');
    }
    // 非回收站视图仍保留 view-enter 动画
    if (view === 'trash') {
        _viewAnimating = false;
        return;
    }
    targetView.classList.add('view-enter');
    targetView.addEventListener('animationend', function onEnterEnd() {
        targetView.removeEventListener('animationend', onEnterEnd);
        targetView.classList.remove('view-enter');
        _viewAnimating = false;
    }, { once: true });
});
```

## 受影响文件清单

| 文件 | 变更 |
|------|------|
| `frontend/src/css/animations.css` | 重写 `trashEnter` 关键帧（2 帧 + 简单缓出） |
| `frontend/src/js/trash-page.js` | 修改 JS 动画属性和缓动曲线 |
| `frontend/src/css/components/main-content.css` | 移除 `.trash-item` 的 `opacity: 0` |
| `frontend/src/main.js` | 回收站视图跳过 `.view-enter` 淡入 |

## 验证步骤

1. 运行 `npx vite build` 构建前端
2. 重启 Wails 应用
3. 切换到回收站页面，检查：
   - 是否还有 3 次闪烁
   - items 是否流畅入场（淡入 + 微上移）
   - 入场动画是否平滑
4. 切换到其他视图（笔记、设置等），检查：
   - 其他视图的 `.view-enter` 淡入是否正常
5. 从回收站切回笔记首页，检查：
   - 卡片入场动画是否正常