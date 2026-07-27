# 批量栏按钮风格统一 + 无选中提示

## 现状分析

### 按钮风格不一致

| 按钮 | 背景 | 边框 | 圆角 | 备注 |
|------|------|------|------|------|
| 置顶/添加标签/移除标签/移动 (`.batch-btn`) | `var(--hover-bg)` | `1px solid var(--border)` | `--radius-md` (8px) | 填充按钮 |
| 批量删除 (`.batch-btn.btn-danger `) | transparent / `var(--danger-bg)` | `1px solid var(--border)` | `--radius-md` | 特殊处理 |
| 退出批量 (`.batch-cancel`) | transparent | **无边框** (`transparent`) | 默认 | 纯文字样式 |
| 全选 (`.batch-select-all-btn`) | transparent | `1px solid var(--border)` | `--radius-sm` (较小) | 文字 accent 色 |

### 无选中时按钮行为

| 操作 | 当前行为 | 行号 |
|------|---------|------|
| `batchDeleteSelected()` | 静默 `return` | main.js#5087 |
| `batchPinSelected()` | 静默 `return` | main.js#5111 |
| `batchMoveBtn` 点击 | 静默 `return` | main.js#5733 |
| `openBatchTagPicker()` | ✅ 已调用 `nm.show('请先选择笔记', 'warning')` | main.js#5158-5161 |

## 修改方案

### 1. CSS：统一非删除按钮风格（main-content.css）

让 `.batch-cancel` 和 `.batch-select-all-btn` 与普通 `.batch-btn` 保持一致的视觉语言（相同的边框、圆角、hover 效果），通过颜色区分重要性。

#### 1a. 修改 `.batch-cancel`

```css
/* 之前 */
.batch-cancel {
  background: transparent;
  color: var(--text-muted);
  border-color: transparent;
}
.batch-cancel:hover {
  color: var(--text-secondary);
  background: var(--hover-bg);
}

/* 之后 */
.batch-cancel {
  background: var(--hover-bg);
  color: var(--text-muted);
  border: 1px solid var(--border);
  /* 使用 .batch-btn 相同的 border-radius、padding、font-size 等 */
}
.batch-cancel:hover {
  color: var(--text-secondary);
  background: var(--border);
}
```

- 添加边框 `1px solid var(--border)`，与其他按钮一致
- 使用 `var(--hover-bg)` 背景，与 `.batch-btn` 一致
- hover 时背景变为 `var(--border)`，与 `.batch-btn:hover` 一致
- 保留 `color: var(--text-muted)` 以体现"退出"的次要操作性质

#### 1b. 修改 `.batch-select-all-btn`

```css
/* 之前 */
.batch-select-all-btn {
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 3px 10px;
  font-size: 0.75rem;
  color: var(--accent);
  cursor: pointer;
  font-family: inherit;
  transition: var(--transition);
  white-space: nowrap;
}
.batch-select-all-btn:hover {
  background: var(--accent-lighter);
  border-color: var(--accent);
}

/* 之后 */
.batch-select-all-btn {
  background: var(--hover-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 6px 14px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-primary);
  cursor: pointer;
  font-family: inherit;
  transition: var(--transition);
  white-space: nowrap;
}
.batch-select-all-btn:hover {
  background: var(--border);
}
```

- 使用 `var(--hover-bg)` 背景 + `var(--radius-md)` 圆角 + 相同 padding，与 `.batch-btn` 一致
- hover 效果与 `.batch-btn:hover` 一致
- 文字色改为 `var(--text-primary)` 保持一致

#### 1c. 批量删除按钮保持现状

`.batch-btn.btn-danger` 相关的 4 态（禁用/激活/hover/active）保留不变。

### 2. JS：无选中时显示通知（main.js）

三个函数/处理器的 `return` 前添加 `nm.show('请先选择笔记', 'warning')`：

#### 2a. `batchDeleteSelected()` — 第 5085-5087 行

```javascript
async function batchDeleteSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }
    // ...
```

#### 2b. `batchPinSelected()` — 第 5109-5111 行

```javascript
async function batchPinSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }
    // ...
```

#### 2c. `batchMoveBtn` 点击处理 — 第 5732-5734 行

```javascript
els.batchMoveBtn.addEventListener('click', () => {
    if (state.selectedNoteIds.size === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }
    openMoveDialog([...state.selectedNoteIds]);
});
```

## 涉及文件

| 文件 | 修改内容 |
|------|---------|
| `frontend/src/css/components/main-content.css` | 统一 `.batch-cancel` 和 `.batch-select-all-btn` 的边框/背景/圆角 |
| `frontend/src/main.js` | 3 处添加 `nm.show('请先选择笔记', 'warning')` |

## 验证步骤

1. 进入批量管理模式，查看所有按钮是否视觉一致（边框/背景/圆角/hover）
2. 无选中时点击批量删除 → 弹出黄色 warning 提示
3. 无选中时点击批量置顶 → 弹出黄色 warning 提示
4. 无选中时点击移动到 → 弹出黄色 warning 提示
5. 选中笔记后点击各按钮 → 正常执行操作
6. 选中笔记后 hover 批量删除 → 实色红底白字 + 阴影
