# 优化新增预设入场动画

## 问题概述

新增预设保存后，预设管理列表会完全重新渲染（`presetMgrContainer.innerHTML = ''`），所有行瞬间弹出，没有任何过渡动画。这与删除预设的平滑滑出动画形成鲜明对比，显得突兀。

## 当前状态分析

**触发流程：**
1. 用户点击新增 → `savePresetModal()` 调用 API
2. 成功后 → `closePresetModal()` + `loadProfiles()` + `renderPresetMgrList()`（如果列表已展开）
3. `renderPresetMgrList()`（[main.js#L2645-L2732](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2645)）执行 `presetMgrContainer.innerHTML = ''` 清空容器，然后遍历 `profiles` 重建所有 DOM 节点
4. 所有行同时插入 DOM，无任何入场动画

**现有动画资源：**
- 容器在首次展开时有 `mgrSlideDown` 动画（`settings-panel.css#L743-L754`）
- 删除行有 `presetDeleteOut` 动画（`settings-panel.css#L891-L912`）
- 全局有 `anim-slide-up`、`view-enter` 等入场动画模式（`animations.css`）
- 搜索弹窗结果列表使用 staggered 延迟入场

## 修改方案

采用 **CSS animation + staggered 延迟** 为每行添加入场动画。

### 步骤 1：在 settings-panel.css 中添加行入场动画

在 `presetDeleteOut` 动画之后添加：

```css
/* 预设列表行入场动画 */
.preset-row-enter {
    animation: presetRowEnter 300ms ease-out both;
}

@keyframes presetRowEnter {
    from {
        opacity: 0;
        transform: translateY(-6px) scale(0.97);
    }
    to {
        opacity: 1;
        transform: translateY(0) scale(1);
    }
}
```

设计理念：
- `translateY(-6px)` + `scale(0.97)` → `(0, 1)`：轻微上弹 + 放大，比纯 fade-in 更有质感
- 300ms `ease-out`：快速到达终点，尾部平缓收束
- `both`：动画开始前应用起始态，结束后保持终态

### 步骤 2：在 renderPresetMgrList 中应用 staggered 动画

**文件：** `frontend/src/main.js`

**修改点：** `renderPresetMgrList` 中 `profiles.forEach` 循环（L2681-L2729）

在创建每行后、appendChild 前添加：

```js
// 添加入场动画（错开延迟避免同时弹出）
row.classList.add('preset-row-enter');
row.style.animationDelay = `${index * 50}ms`;
```

**当前代码：**
```js
profiles.forEach(p => {
    const row = document.createElement('div');
    // ... 构建 row 内容 ...
    presetMgrContainer.appendChild(row);
});
```

**改为：**
```js
profiles.forEach((p, index) => {
    const row = document.createElement('div');
    // ... 构建 row 内容（不变） ...
    row.classList.add('preset-row-enter');
    row.style.animationDelay = `${index * 50}ms`;
    presetMgrContainer.appendChild(row);
});
```

效果：每行按 `index * 50ms` 错开入场，第 0 条延迟 0ms，第 1 条延迟 50ms，第 2 条延迟 100ms……最多 n-1 条 * 50ms。视觉上像波一样逐条显现。

仅修改 `profiles.forEach` 回调的这两处，其余逻辑完全不变。

### 步骤 3：更新项目记忆 AGENTS.md

在 AGENTS.md 中记录预设弹窗键盘快捷键（ESC 关闭、Enter 保存）的支持信息，方便后续维护。

## 影响范围

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `frontend/src/css/components/settings-panel.css` | 新增 | 添加 `preset-row-enter` 动画（CSS 类 + keyframes） |
| `frontend/src/main.js` | 修改 | `renderPresetMgrList` 中每行添加 `preset-row-enter` 类 + staggered `animationDelay` |
| `AGENTS.md` | 修改 | 记录预设弹窗键盘快捷键支持 |

## 验证步骤

1. 打开设置 → 配置预设 → 管理
2. 点击「新增」，填写信息后保存
3. 观察：关闭弹窗后，预设列表中的行应从上到下逐条显现（轻弹入效果）
4. 编辑预设保存后也应该有相同效果
5. 首次点击「管理」展开列表时，行也应有入场动画（而非硬出现）

## 不做的事

- 不改动删除动画
- 不改动容器展开/关闭动画（`mgrSlideDown`/`mgrSlideUp`）
- 不改动预设弹窗本身的动画
- 不改动后端 Go 代码
