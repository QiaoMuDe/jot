# 优化预设管理删除动画方案

## 问题概述

在设置页面的"管理预设"列表中，点击删除按钮后，预设条目会**瞬间消失**——`deleteProfile` 函数直接调用 `renderPresetMgrList()` 刷新整个列表（`presetMgrContainer.innerHTML = ''`），没有任何过渡动画，视觉上非常生硬。

## 当前状态分析

**涉及文件：**

- **入口 / 逻辑：** `frontend/src/main.js`
  - `deleteProfile(id, name)` (L2616-L2630) — 确认后调用 API 删除，然后直接 `renderPresetMgrList()` 全量刷新。
  - `renderPresetMgrList()` (L2633-L2724) — `presetMgrContainer.innerHTML = ''` 立即清除所有行，重新构建 DOM。
  - 预设行 DOM 在 `renderPresetMgrList` 的 `forEach` 循环中创建（L2672-L2719），每行的 `delBtn` 绑定 `deleteProfile`。

- **现有动画资源：** `frontend/src/css/animations.css`
  - 已有 `deleteOut` 动画（L134-L143）：抖动 + 红闪 + 缩小 + 渐隐（效果较强烈，适合永久删除）。
  - 已有 `shrinkOut` 动画（L121-L124）：缩小 + 渐隐（更温和，适合通用删除）。
  - 已有 `view-exit` 动画（L44-L51）：向上滑出 + 渐隐。

- **预设管理列表 CSS：** `frontend/src/css/components/settings-panel.css`
  - `.preset-mgr-list` (L733-L737) — 容器展开入场动画。
  - `.preset-mgr-list.closing` (L739-L741) — 容器关闭出场动画。

## 修改方案

采用 **「选中行先播放删除动画 → 动画结束后再执行 API 删除 + 刷新列表」** 的策略。

### 步骤 1：在 settings-panel.css 中添加预设专用的删除动画

新增一个平缓的列表项删除动画 `presetDeleteOut`，比现有的 `deleteOut` 更温和：
- 平滑向右滑出 + 渐隐 + 高度折叠，避免突兀的抖动/红闪。
- 适合列表项的优雅移除。

```css
/* 预设删除项滑出动画 */
.preset-delete-out {
    animation: presetDeleteOut 300ms ease-in forwards;
}

@keyframes presetDeleteOut {
    0% {
        opacity: 1;
        transform: translateX(0);
        max-height: 48px;
        margin-bottom: 0;
        padding: 6px 8px;
    }
    60% {
        opacity: 0.6;
        transform: translateX(20px);
    }
    100% {
        opacity: 0;
        transform: translateX(40px);
        max-height: 0;
        margin-bottom: 0;
        padding: 0 8px;
    }
}
```

### 步骤 2：修改 main.js 中的 deleteProfile 函数

**修改点：** `deleteProfile` 函数（L2616-L2630）

将直接删除逻辑改为：
1. 找到被删除行对应的 DOM 元素（传入 `rowEl` 参数）。
2. 为该行添加 `.preset-delete-out` 类播放动画。
3. 监听 `animationend` 事件，在动画结束后再执行 API 删除。
4. 删除成功后刷新列表。

**同时修改 delBtn 的事件绑定**（L2710-L2713），将 `row` 元素引用传给 `deleteProfile`。

### 步骤 3：修改 renderPresetMgrList 中 delBtn 的事件绑定

当前代码：
```js
delBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    deleteProfile(p.id, p.name);
});
```

改为：
```js
delBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    deleteProfile(p.id, p.name, row);
});
```

## 影响范围

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `frontend/src/css/components/settings-panel.css` | 新增 | 添加 `presetDeleteOut` 动画（CSS 类 + keyframes） |
| `frontend/src/main.js` | 修改 | `deleteProfile` 函数：增加动画播放 → 等待 → 删除逻辑 |
| `frontend/src/main.js` | 修改 | `renderPresetMgrList` 中 delBtn 回调：传入 `row` 参数 |

## 验证方式

1. 打开设置 → AI → 配置预设 → 管理
2. 点击某条预设的"删除"按钮
3. 确认删除弹窗
4. 观察：该行先向右滑出并渐隐，然后列表自动刷新

## 不做的事

- 不改动其他删除/动画逻辑
- 不修改预设管理列表的展开/关闭动画
- 不修改后端 Go 代码
