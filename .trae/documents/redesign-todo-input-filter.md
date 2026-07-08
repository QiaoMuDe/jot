# 待办清单输入框 + 筛选按钮视觉重新设计

## 当前问题

输入框和筛选按钮并排时存在视觉不协调：

| 元素 | 边框 | 圆角 | 高度(约) |
|------|------|------|---------|
| 输入框 `.todo-input` | `2px dashed` | `--radius-lg` (10px) | 44px |
| 筛选按钮 `.todo-filter-btn` | `1px solid` | `--radius-md` (8px) | 26px |

- **边框风格冲突**: 输入框是虚线(dashed)、按钮是实线(solid)，并排放置时视觉割裂
- **高度不匹配**: 输入框 44px，按钮 26px，不在同一水平线上
- **圆角不一致**: 10px vs 8px，缺乏视觉连贯性
- **缺乏整体感**: 两个元素各自为政，没有"这是一个完整的输入工具栏"的感觉

## 设计方向

**概念**: "一体化输入工具栏" — 将输入框和筛选按钮融合成一个完整的视觉单元

**风格基调**:
- 温暖、精致、细节丰富
- 柔和的容器背景包裹整体
- 用细微的间距和分割线区分功能区
- 聚焦态有优雅的发光反馈

## 具体改动方案

### 1. `.todo-input-wrap` — 整体容器重设计

**设计**: 一个`--input-bg`背景的容器，包含输入区和筛选区，高度统一，整体圆角

| 属性 | 值 | 说明 |
|------|-----|------|
| `display` | `flex` | 保持水平布局 |
| `align-items` | `stretch` | 子项等高 |
| `background` | `var(--input-bg)` | 统一的柔和背景 |
| `border-radius` | `var(--radius-lg)` | 统一大圆角 |
| `border` | `1px solid var(--border)` | 整体边框 |
| `box-shadow` | `var(--shadow-sm)` | 轻微浮起感 |
| `transition` | 所有 0.2s ease | 聚焦动画 |

### 2. `.todo-input` — 输入框重设计

**设计**: 无边框，融入容器背景，聚焦时容器整体高亮

| 属性 | 值 | 说明 |
|------|-----|------|
| `border` | `none` | 去掉虚线边框 |
| `background` | `transparent` | 透出容器背景 |
| `padding` | `12px 16px` | 保持舒适间距 |
| `font-size` | `0.9rem` | 不变 |
| `outline` | `none` | 去掉聚焦外轮廓 |
| `flex` | `1` | 撑满剩余空间 |

### 3. 新增 — 垂直分割线

在输入框和筛选按钮之间加一条细分割线，强化功能分区

```css
.todo-input-divider {
    width: 1px;
    background: var(--border);
    align-self: center;
    height: 24px;
    flex-shrink: 0;
}
```

### 4. `.todo-filter-bar` — 筛选区重设计

| 属性 | 值 | 说明 |
|------|-----|------|
| `padding` | `4px` | 内部留白让按钮不贴边 |
| `display` | `flex` | 保持水平 |
| `align-items` | `center` | 居中 |
| `gap` | `2px` | 紧凑排列 |

### 5. `.todo-filter-btn` — 筛选按钮重设计

**设计**: 无边框的纯文字按钮，选中态用填充背景

| 属性 | 旧值 | 新值 | 说明 |
|------|------|------|------|
| `border` | `1px solid var(--border)` | `none` | 去掉边框，更干净 |
| `background` | `transparent` | `transparent` | 默认透明 |
| `padding` | `4px 10px` | `6px 14px` | 稍大一点增加点击区域 |
| `border-radius` | `var(--radius-md)` | `var(--radius-md)` | 保持 |
| `font-size` | `0.75rem` | `0.8rem` | 略微增大提升可读性 |
| `font-weight` | `normal` | `500` | 加强字重 |

选中态:
| `background` | `var(--accent)` | `var(--accent)` | 品牌色填充 |
| `color` | `#fff` | `#fff` | 白色文字 |
| `box-shadow` | (无) | `0 1px 3px rgba(...)` | 轻微浮起 |

悬停态(未选中):
| `background` | `transparent` | `var(--hover-bg)` | 悬停有背景反馈 |

### 6. 聚焦态 — 容器整体反馈

给 `.todo-input-wrap:focus-within` 添加：

| 属性 | 值 |
|------|-----|
| `border-color` | `var(--accent)` |
| `box-shadow` | `0 0 0 3px var(--accent-lighter)` |

### 7. 文字按钮的13px垂直对齐优化

由于容器 `align-items: stretch`，筛选按钮高度与输入框一致，用 `padding` 控制内部间距。

## 涉及的 HTML 改动 (`frontend/index.html`)

在输入框和筛选栏之间增加分割线：

```html
<div class="todo-input-wrap">
    <input type="text" id="todoInput" ...>
    <div class="todo-input-divider"></div>   <!-- ← 新增 -->
    <div class="todo-filter-bar">
        ...
    </div>
</div>
```

## 涉及的文件

| 文件 | 改动类型 |
|------|---------|
| `frontend/index.html` | 输入框和筛选栏之间加 `.todo-input-divider` |
| `frontend/src/css/components/todo.css` | 完整重写 `.todo-input-wrap`、`.todo-input`、`.todo-filter-bar`、`.todo-filter-btn` 样式，新增 `.todo-input-divider` 和 `focus-within` |

## 验证方式

1. 输入框和筛选按钮在同一水平线上，视觉上是一个整体容器
2. 输入框没有独立边框，聚焦时容器整体高亮（accent 边框 + 发光阴影）
3. 筛选按钮为无边框文字按钮，选中态填充品牌色
4. 输入框和筛选区之间有垂直分割线
5. 在所有主题（亮色/暗色/Nord等）下颜色协调
