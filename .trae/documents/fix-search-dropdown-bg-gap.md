# 修复搜索弹窗标签筛选下拉选中态背景右侧留白

## 问题

点击标签筛选下拉菜单后，选中项（.search-modal-filter-option.selected）的琥珀色背景没有铺满整行，右侧留有空白。

## 根因排查

**根因：**.search-modal-filter-dropdown 设置了 `overflow-y: auto` + `max-height: 240px`。当标签数量超过 240px 高度时，下拉容器内部出现滚动条（Windows 默认滚动条宽约 17px）。滚动条会占据容器内容区的一部分宽度，而 .search-modal-filter-option 的 `width: 100%` 或 `align-items: stretch` 拉伸到的宽度是**减去滚动条宽度后**的内容区宽度。因此选中态背景只覆盖到滚动条左侧，滚动条区域及其右侧留白。

**如果没有超过 max-height（无滚动条出现），则无此问题。** 多数用户的标签数量会超过 240px 的可视行数（约 10-12 个标签），所以该问题普遍出现。

**之前修改无效的原因：** 添加 `width: 100%; box-sizing: border-box;` 和 `display: flex; flex-direction: column; align-items: stretch;` 都无法绕过 CSS 滚动条占据内容区宽度的固有行为。

## 修复方案

### 方案：自定义薄滚动条 + overflow-x: hidden

在 `.search-modal-filter-dropdown` 上设置自定义 WebKit 滚动条样式，将滚动条宽度从默认 ~17px 缩减为 6px，同时隐藏水平滚动条。使留白区从 17px 缩小到几乎不可见。

### 修改文件

**`frontend/src/style.css`** — `.search-modal-filter-dropdown` 规则块后新增自定义滚动条样式

```css
.search-modal-filter-dropdown::-webkit-scrollbar {
    width: 6px;
}
.search-modal-filter-dropdown::-webkit-scrollbar-track {
    background: transparent;
}
.search-modal-filter-dropdown::-webkit-scrollbar-thumb {
    background: var(--border);
    border-radius: 3px;
}
```

同时给 `.search-modal-filter-dropdown` 添加 `overflow-x: hidden` 防止水平溢出产生干扰滚动条。

### 其他文件无需修改

- `index.html` — 不涉及
- `main.js` — 不涉及（渲染逻辑不变）

## 验证

1. 打开搜索弹窗（Ctrl+F）
2. 点击标签筛选按钮，展开下拉菜单
3. 选中任一标签，观察选中态背景是否铺满整行（右侧不再有明显留白）
4. 测试多个标签的情况（确保超过 max-height 触发滚动条的情况也正常）
