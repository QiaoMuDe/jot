# 新增待办动画优化 Spec

## Why

当前新增待办条目的动画（`todoNew`）只对新条目本身做滑入淡入，已有条目完全不动，缺少"腾出空间"的视觉引导。优化后，已有条目会先向下位移一个位置，然后新条目优雅插入，让"新增"操作有清晰的**空间叙事感**。

## What Changes

- **CSS 新增**：`todoShiftDown` 关键帧动画 + `todo-shifting` class，让已有条目整体向下偏移一个待办项高度
- **CSS 新增**：`todoItemEnter` 关键帧动画，替代现有 `todoNew`，配合"先下移后插入"的两段式节奏
- **JS 修改**：重写 `addTodo()`，实现"先下移 → 再插入"的两段式动画编排
- **JS 修改**：`addTodo()` 中处理 filter 切换后的 `loadTodos()` 场景，确保所有场景下动画一致
- **删除/保留**：保留 `todo-new` CSS class，但改用 `todo-item-enter`（更精确的名称），旧 `todoNew` 动画可保留作为回退

## Impact

- Affected code:
  - `frontend/src/css/components/todo.css` — 新增动画关键帧 + class
  - `frontend/src/main.js` — 重写 `addTodo()` 动画编排逻辑

## 设计方向

采用 **空间叙事（Spatial Narrative）** 动画哲学：
- 动画不是装饰，而是向用户解释"发生了什么"
- 条目向下位移说明"列表在为新条目腾出空间"
- 新条目滑入说明"新内容已就位"
- 缓动曲线使用 `cubic-bezier(0.34, 1.56, 0.64, 1)` — 带轻微弹性的 ease-out，让动画有"活"的感觉，但不夸张

### 动画流程

```
Phase 1 (0-280ms): 已有条目整体下移一个位置
  ┌─────────────────────────────────────────┐
  │ [输入框]                                 │
  │ → 用户按下 Enter                         │
  │ → 异步等待 CreateTodo API 返回           │
  │ → 获取第一个待办项的高度 H               │
  │ → 所有已有条目添加 class="todo-shift"    │
  │ → transform: translateY(H + gap)        │
  │ → transition: 280ms cubic-bezier(...)   │
  └─────────────────────────────────────────┘

Phase 2 (200-500ms, 与 Phase 1 尾部重叠):
  新条目插入 + 入场
  ┌─────────────────────────────────────────┐
  │ → prepend 新条目 DOM (设置不可见)        │
  │ → 等待 50ms (让浏览器确认布局)           │
  │ → 新条目加 class="todo-item-enter"      │
  │ → 从 translateY(-30px) scale(0.96)       │
  │    + opacity 0 → 落到最终位置            │
  │ → 350ms ease-out                         │
  └─────────────────────────────────────────┘

Phase 3 (500ms后): 收尾清理
  ┌─────────────────────────────────────────┐
  │ → 移除已有条目的 todo-shift class       │
  │   (由于新条目已插入，自然位置已正确，    │
  │    移除 transform 无闪烁)                │
  │ → 新条目移除 todo-item-enter class       │
  │ → 更新统计                              │
  └─────────────────────────────────────────┘
```

### 为什么不会有闪烁

关键原理：当新条目 prepend 后，已有条目在文档流中的**自然位置**正好比原来下移了一个条目的高度。而此时它们通过 `transform: translateY(H)` 也在同样的位置。移除 transform 后，浏览器发现自然位置 == 变换位置，不会触发重排动画，零闪烁。

## ADDED Requirements

### Requirement: 新增下移动画

The system SHALL provide a "shift down" animation for existing todo items when a new one is added.

#### Scenario: 已有条目下移

- **WHEN** 用户新增待办
- **THEN** 所有已有待办条目通过 `transform: translateY()` 整体下移一个条目的高度
- **AND** 下移距离为 `(待办项高度 + gap)`，通过 JS 动态获取
- **AND** 下移动画时长 280ms，使用 `cubic-bezier(0.34, 1.56, 0.64, 1)` 缓动
- **AND** 下移完成后（与入场尾部重叠），状态被清理，条目留在自然位置

### Requirement: 新条目入场动画

The system SHALL provide a coordinated entrance animation for the new todo item.

#### Scenario: 新条目插入

- **WHEN** 已有条目下移动画进行到约 200ms 时
- **THEN** 新条目被 prepend 到列表顶部
- **AND** 新条目从 `translateY(-30px) scale(0.96) opacity(0)` 动画到最终位置
- **AND** 入场动画时长 350ms，使用 `ease-out` 缓动
- **AND** 入场动画结束后移除动画 class

## MODIFIED Requirements

### Requirement: addTodo() 逻辑（修改）

原有的简单 prepend + 单一条目动画逻辑，替换为两段式编排：

1. 调用 `CreateTodo` API 获取新条目数据
2. 获取已有条目的实际高度（`offsetHeight`），计算偏移量
3. 对所有已有条目应用 `.todo-shift` class（触发 `transform` transition）
4. 280ms 后清除 `.todo-shift`
5. 新条目 prepend + `.todo-item-enter` class 触发入场动画（与第 3 步尾部重叠，在约 200ms 时启动）
6. 更新统计

#### Scenario: 非"待办"筛选模式下新增

- **WHEN** 当前筛选不是"待办"（如"全部"或"已完成"）
- **THEN** 自动切换到"待办"筛选
- **AND** 用 `loadTodos()` 刷新列表（保持现有行为，因为有筛选切换，无法做两段式动画）

## REMOVED Requirements

None. 旧 `todoNew` 动画保留但不再在新增场景中使用（保留作为回退/备用）。
