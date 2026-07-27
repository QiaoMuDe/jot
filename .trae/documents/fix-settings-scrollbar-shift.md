# 修复设置页滚动条导致内容左移问题

## 根因分析

### 滚动容器层级

```
#mainContent (overflow-y: auto, scrollbar-gutter: stable) ← 外层
  └── #viewSettings.view (height: 100%)
        └── .settings-content (flex: 1, overflow: hidden)
              └── .settings-panels (flex: 1, overflow-y: auto, ❌ 无 scrollbar-gutter: stable)
                    └── .settings-panel.active → 内容卡片
```

### 问题原因

- 设置页的实际滚动容器是 **`.settings-panels`**（`overflow-y: auto`），而非 `#mainContent`
- `#mainContent` 有 `scrollbar-gutter: stable`，但它是外层容器，设置页大部分情况不会溢出它
- **`.settings-panels` 没有 `scrollbar-gutter: stable`**，当内容高度超出面板高度时，6px 宽度的滚动条出现，挤占内容水平空间，导致卡片和文字向左位移
- 内容变短后滚动条消失，内容又回到原位——产生"左右晃"的视觉抖动

### 项目已有同类修复

- `#mainContent` 已有 `scrollbar-gutter: stable`
- `.font-family-options`（字体下拉）已有 `scrollbar-gutter: stable`
- `#editorPanes`（编辑器）已有 `scrollbar-gutter: stable`
- `#viewTodo`（待办视图）已有 `scrollbar-gutter: stable`
- AI 对话和日历视图主动设为 `scrollbar-gutter: auto`（因其内部滚动由子容器处理）

## 修改方案

### 1. 给 `.settings-panels` 添加 `scrollbar-gutter: stable`

**文件**: `frontend/src/css/components/settings-panel.css`

```css
.settings-panels {
  flex: 1;
  overflow-y: auto;
  scrollbar-gutter: stable;  /* ← 新增 */
  padding: 0;
  min-width: 0;
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar-thumb) transparent;
}
```

### 2. 可选：同步给 `.settings-nav` 添加

`.settings-nav` 也有 `overflow-y: auto`，虽然 176px 窄侧栏内容不太溢出，但为了一致性也可以加上 `scrollbar-gutter: stable`。

不过考虑到侧栏只有 9 项，基本不会触发滚动条，可以不修改它，避免不必要的变更。

## 涉及文件

| 文件 | 修改内容 |
|------|---------|
| `frontend/src/css/components/settings-panel.css` | `.settings-panels` 添加 `scrollbar-gutter: stable` |

## 验证步骤

1. 进入设置页，切换到一个内容较多的面板（如 AI 设置）
2. 调整窗口高度使内容刚好出现滚动条
3. 观察内容是否不再左右位移
4. 上下滚动正常，内容区宽度稳定
5. 切换不同设置面板，各面板均无滚动条抖动
