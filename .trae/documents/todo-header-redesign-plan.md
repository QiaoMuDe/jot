# 待办清单顶部组件重新设计 — 执行计划

## 问题
当前 `.todo-input-wrap` 是水平 flex 布局：
```
┌──────────────────────────────────────────┐
│ [textarea auto-expand] │ [待办] [已完成]  │
└──────────────────────────────────────────┘
```
textarea 多行扩展时，右边的筛选栏在变高的容器中垂直居中，视觉上很别扭。

## 设计方案

### 布局变更：水平同排 → 垂直堆叠

改为上下两行，筛选栏放在输入区下方：

```
┌──────────────────────────────────────────┐
│  [textarea - full width, auto-expand]    │
├──────────────────────────────────────────┤
│  ● 待办 3    已完成 5                     │
└──────────────────────────────────────────┘
```

### 设计方向
- **风格**：延续现有"温暖手写感"，保持一致的色彩 token
- **区别度**：将筛选栏从工具栏配角提升为独立导航层级，用 pill 式选中态强化视觉层次
- **交互**：筛选按钮增加计数徽标（已有），添加小型 SVG 图标让视觉更丰富

### 改动清单

#### 1. HTML — 结构调整
**文件**: `frontend/index.html` #L1605-L1613

当前：
```html
<div class="todo-input-wrap">
    <textarea id="todoInput" class="todo-input" ...></textarea>
    <div class="todo-input-divider"></div>
    <div class="todo-filter-bar">
        <button class="todo-filter-btn active" data-filter="active">待办</button>
        <button class="todo-filter-btn" data-filter="done">已完成</button>
    </div>
</div>
```

改为：移除 `.todo-input-divider`（用 CSS 替代），筛选栏移到第二行，按钮增加小图标：
```html
<div class="todo-input-wrap">
    <textarea id="todoInput" class="todo-input" placeholder="添加新的待办事项..." rows="1" autocomplete="off"></textarea>
    <div class="todo-filter-bar">
        <button class="todo-filter-btn active" data-filter="active">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            待办
        </button>
        <button class="todo-filter-btn" data-filter="done">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
            已完成
        </button>
    </div>
</div>
```

#### 2. CSS — 完全重写输入区布局
**文件**: `frontend/src/css/components/todo.css`

**`.todo-input-wrap`**：
- `flex-direction: column` 替代 `row`
- 去掉 `align-items: stretch`
- 保持 card 样式（背景、边框、圆角、阴影）

**`.todo-input`**：
- 去掉 `flex: 1`（不再需要横向分配空间）
- 宽度 100%
- `border-radius: 0`（因为容器已圆角，textarea 本身直角让背景连贯）
- 增加底部内边距略减

**`.todo-input-divider`**（删除 HTML 元素，改为 CSS 伪元素或 border）：
- 新增 `.todo-input-wrap::after` 作为内部水平分割线（在 textarea 和 filter 之间）
- 或者直接用 `.todo-filter-bar` 的 `border-top`

**`.todo-filter-bar`**：
- 不再 `flex-shrink: 0; align-self: center`
- 改为 `width: 100%`，左 padding 与 textarea 对齐
- padding 缩减（8px 16px）
- 添加 `border-top: 1px solid var(--border)` 分割线

**`.todo-filter-btn`**：
- 添加 SVG 图标支持（flex 布局，gap 4px）
- 保持原有的 active/hover 过渡
- transition 保持 0.15s

#### 3. JS — 无变动
筛选按钮的点击委托不依赖 CSS 布局，保持原样。

## 验证
- `npx vite build` 构建无报错
- textarea 多行展开时筛选栏位置不受影响（固定在底部）
- 筛选按钮点击切换正常
- 全 14 套主题下分割线可见、颜色一致
