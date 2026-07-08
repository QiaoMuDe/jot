# 待办条目视觉重新设计 — "温暖卡片"方向

## 当前状态

当前待办条目的设计较为平淡：

* `.todo-item`: 简单白底卡片，hover 仅变背景色，无阴影层次

* `.todo-checkbox`: 20px 标准圆形，勾选 SVG 通过 opacity 显隐

* `.todo-text.done`: 仅变灰 + 加删除线，缺少完成感的视觉区分

* `.todo-delete-btn`: 简单的透明度切换，无动效

* **已完成条目与待办条目视觉差异不足**，缺少"已完成"特有的安放感

## 设计方向

**概念**: "温暖卡片" — 每个待办像一张有触感的纸质卡片

**风格基调**:

| 状态   | 感觉       | 关键词            |
| ---- | -------- | -------------- |
| 待办   | 鲜活的、待处理的 | 白净、微阴影、略浮起、有弹性 |
| 已完成  | 安放的、归档的  | 柔和、左缘标记、轻透明、温和 |
| 删除按钮 | 隐藏的工具    | 滑入、红色反馈、轻快     |

## 具体改动

### 1. HTML/JS 改动

**`frontend/src/main.js`** **— renderTodos 函数**
在 `.todo-item` 的 class 中添加 `todo.done ? 'completed' : ''`，使已完成条目获得独立样式。

```js
// 改动前
<div class="todo-item" data-id="${todo.id}">

// 改动后
<div class="todo-item ${todo.done ? 'completed' : ''}" data-id="${todo.id}">
```

### 2. CSS 完整重写 (`frontend/src/css/components/todo.css`)

#### `.todo-item` — 基础卡片

| 属性            | 旧值                      | 新值                        | 说明       |
| ------------- | ----------------------- | ------------------------- | -------- |
| padding       | `10px 12px`             | `14px 16px`               | 更大空间，更舒适 |
| border-radius | `--radius-md`(8px)      | `--radius-lg`(10px)       | 与输入工具栏统一 |
| box-shadow    | 无                       | `var(--shadow-sm)`        | 卡片有轻微浮起感 |
| border        | `1px solid transparent` | `1px solid var(--border)` | 定义卡片边界   |
| gap           | `12px`                  | `14px`                    | 元素间距略增   |

#### `.todo-item:hover` — 悬停态

| 属性           | 值                  | 说明                 |
| ------------ | ------------------ | ------------------ |
| box-shadow   | `var(--shadow-md)` | 阴影加深，卡片"抬起"        |
| transform    | `translateY(-1px)` | 轻微上浮               |
| border-color | `transparent`      | 悬停时去掉边框，更干净        |
| background   | `var(--card-bg)`   | 保持白色（不变成 hover-bg） |

#### `.todo-item.completed` — 已完成态（新增）

| 属性           | 值                                         | 说明           |
| ------------ | ----------------------------------------- | ------------ |
| background   | `var(--hover-bg)`                         | 柔和底色，区分"已完成" |
| opacity      | `0.8`                                     | 轻微降低存在感      |
| border-color | `transparent`                             | 去掉边框         |
| box-shadow   | 无                                         | 去掉阴影，更平      |
| /\* 左缘标记 \*/ | `box-shadow: inset 3px 0 0 var(--accent)` | 左侧琥珀色竖条，视觉锚点 |

hover 时 completed 条目只轻微改变 shadow，不上浮。

#### `.todo-checkbox` — 复选框

| 属性           | 旧值                     | 新值                                                |
| ------------ | ---------------------- | ------------------------------------------------- |
| width/height | 20px                   | 22px                                              |
| border-color | `var(--text-muted)`    | `var(--border)`                                   |
| background   | transparent            | `var(--card-bg)`                                  |
| hover        | `border-color: accent` | `transform: scale(1.12)` + `border-color: accent` |
| transition   | `all 0.15s`            | `all 0.2s cubic-bezier(...)` 弹性曲线                 |

checked 态新增 `box-shadow: 0 2px 4px rgba(var(--accent-rgb), 0.3)` 给勾选按钮增加浮起感。

勾选 SVG 改为 opacity + scale 双重动画，更生动。

#### `.todo-text` — 文本

| 属性        | 旧值       | 新值      |
| --------- | -------- | ------- |
| font-size | 0.875rem | 0.9rem  |
| padding   | 2px 4px  | 4px 6px |

**`.todo-text.done`** — 已完成文本

| 属性                        | 值                   | 说明                           |
| ------------------------- | ------------------- | ---------------------------- |
| color                     | `var(--text-muted)` | 保持                           |
| text-decoration           | `line-through`      | 保持                           |
| text-decoration-color     | (默认)                | `var(--accent)` — 琥珀色删除线，更温暖 |
| text-decoration-thickness | (默认)                | `1.5px` — 略粗，更清晰             |

#### `.todo-delete-btn` — 删除按钮

新增滑入动效：

* **默认**: `opacity: 0; transform: translateX(6px) scale(0.9);`

* **卡片悬停**: `opacity: 0.5; transform: translateX(0) scale(1);`

* **按钮悬停**: `opacity: 1 !important; background: var(--danger-bg); color: var(--danger); transform: scale(1.15);`

尺寸从 24px 改为 26px，点击区域更大。

## 涉及的文件

| 文件                                     | 改动                                               |
| -------------------------------------- | ------------------------------------------------ |
| `frontend/src/main.js`                 | renderTodos 中给 `.todo-item` 添加 `completed` class |
| `frontend/src/css/components/todo.css` | 重写 106-217 行所有 todo item 相关样式                    |

## 验证方式

1. 待办条目显示为白底卡片，有轻微阴影浮起
2. 鼠标悬停时卡片微微抬起（阴影加深 + 上移 1px）
3. 已完成条目显示柔和底色 + 左侧琥珀色竖条标记，删除线为琥珀色
4. 复选框 hover 有轻微放大效果
5. 删除按钮从右侧滑入，hover 变红色
6. 条目入场动画正常

