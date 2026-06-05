# 交互体验与动画增强 Spec

## Why

当前 Jot 应用的 UI 虽已完成视觉重设计（温暖极简风格），但页面切换生硬、交互动画缺失、微交互反馈不足。需要为每一个页面/视图补充完善的动画系统和交互反馈，让应用使用起来更流畅、更有质感。

## Design Direction: 流畅叙事 (Fluid Narrative)

**风格定位**：以流畅的过渡动画和细腻的微交互，营造连贯、自然的笔记操作体验。每处动效都有因果逻辑，而非纯装饰。

### 动画设计原则

| 原则 | 说明 |
|------|------|
| **因果性** | 每个动画必须表达因果关系（如：点击按钮 → 按钮反馈 → 结果出现），不添加纯装饰动效 |
| **连续性** | 视图切换维持空间连贯性（如：编辑模态从卡片位置缩放出现） |
| **性能优先** | 仅使用 `transform` 和 `opacity`，避免触发重排/重绘 |
| **节奏统一** | 统一使用 200-300ms 时长，`cubic-bezier(0.16, 1, 0.3, 1)` 弹簧缓动曲线 |
| **退出更快** | 退出动画时长约为进入的 60%（~150ms），让应用响应更迅速 |
| **无障碍** | 尊重 `prefers-reduced-motion`，用户开启时降级为瞬时切换 |

### 时间规范

| 场景 | 时长 | 缓动函数 |
|------|------|----------|
| 微交互（按钮/悬停） | 150ms | ease-out |
| 视图切换 | 250ms | cubic-bezier(0.16, 1, 0.3, 1) |
| 模态打开 | 300ms | cubic-bezier(0.16, 1, 0.3, 1) |
| 模态关闭 | 180ms | ease-in |
| 列表交错入场 | 200ms + 每项 40ms 延迟 | ease-out |
| Toast 显隐 | 250ms fade + 300ms slide | ease-out / ease-in |
| 主题切换 | 300ms | ease-out |

---

## What Changes（按页面）

### 0. 全局动画基础
- 在 `app.css` 的 `:root` 中添加动画相关 CSS 变量（duration/easing 统一管理）
- 创建通用动画工具类（`.anim-fade-in`, `.anim-slide-up`, `.anim-scale-in`）
- 在 `app.css` 添加 `prefers-reduced-motion: reduce` 媒体查询，统一禁用动画
- 在 `:root` 和 `[data-theme]` 选择器上添加 `transition` 属性，为主题切换过渡做准备

### 1. 所有笔记页面（卡片网格视图）
卡片网格是应用主页面，需要最丰富的动画体验：
- **网格整体入场**：切换到网格视图时，整个网格容器 `opacity: 0 → 1` + `translateY(12px → 0)`，250ms
- **卡片交错入场**：每张卡片 `opacity: 0 → 1` + `translateY(20px → 0)` + `scale(0.97 → 1)`，每项间隔 40ms，最多 12 项有延迟
- **卡片 hover 增强**：现有上移 2px 基础上，增加阴影过渡（`box-shadow` 150ms ease-out）+ 轻微缩放（`scale(1.02)`）
- **卡片点击反馈**：点击卡片时 `scale(0.98)` 瞬间按下感，释放后恢复
- **置顶按钮**：☆ → ★ 切换增加旋转（180deg）+ 缩放（0.8→1.2→1）动画，200ms
- **复选框选中动画**：批量模式下选中复选框时 `scale(0.8 → 1.1 → 1)` 弹跳效果，200ms
- **加载更多**：新加载的卡片从底部淡入（`translateY(30px → 0)` + `opacity: 0 → 1`，200ms）
- **加载动画**：现有旋转动画不变，卡片容器在加载中显示骨架屏效果

### 2. 数据管理页面
- **页面入场**：切换到数据管理时，整体 `opacity + translateY` 过渡，250ms
- **统计卡片入场**：三张统计卡交错入场（每项 80ms 延迟），`scale(0.9 → 1)` + `opacity: 0 → 1`
- **统计数字动画**：数字从 0 计数动画到实际值（count-up 效果，300ms，仅在首次加载时）
- **操作按钮**：hover 微缩放（`scale(1.03)`），active 按下（`scale(0.95)`）
- **导入结果 Toast**：使用增强后的 Toast 动画（底部滑入 + 淡入）

### 3. 回收站页面
- **页面入场**：切换到回收站时，整体 `opacity + translateY` 过渡，250ms
- **列表项交错入场**：每项 30ms 延迟，`translateX(-12px → 0)` + `opacity: 0 → 1`
- **单条恢复动画**：点击恢复 → 该项收缩淡出（`scale(1) + opacity: 1 → scale(0.8) + opacity: 0`，200ms）→ 从列表中移除
- **单条永久删除动画**：点击永久删除 → 该项收缩淡出 + 红色闪烁（150ms）→ 移除
- **全部恢复动画**：所有项依次收缩淡出（每项 30ms 延迟），批处理栏同步退出
- **全部清空动画**：所有项依次收缩淡出 + 红色闪烁
- **空状态**：从空状态恢复到有数据时，列表项正常交错入场

### 4. 设置页面
- **页面入场**：切换到设置时，整体 `opacity + translateY` 过渡，250ms
- **分段控件滑动动画**：现有 iOS 风格分段控件的滑动指示器动画已实现，保持不变
- **开关切换动画**（toggle-switch）：滑块移动 + 背景色过渡，200ms
- **字体族下拉菜单**：打开时展开动画（`max-height: 0 → 500px` + `opacity: 0 → 1`，200ms），关闭时收起
- **字体下拉搜索**：无额外动画，保持即时响应
- **设置项分隔线**：无动画变化

### 5. 帮助页面（快捷键说明）
- **模态覆盖层入场**：打开时遮罩 `opacity: 0 → 1`（200ms）+ 卡片 `scale(0.9 → 1)` + `opacity: 0 → 1`（300ms），弹簧缓动
- **快捷键列表项**：列表项交错入场（每项 30ms 延迟），`translateY(10px → 0)` + `opacity: 0 → 1`
- **关闭**：卡片 `scale(0.95)` + `opacity: 1 → 0`（150ms），遮罩同步淡出
- **ESC 关闭**：同上

### 6. 新建笔记页面（编辑器模态框）
- **模态打开**：从触发位置（通常是右下角"+"按钮）`scale(0.85)` + `opacity: 0` → 居中 `scale(1)` + `opacity: 1`，300ms
- **遮罩**：同步 `opacity: 0 → 1` 淡入，200ms
- **内容区域延迟入场**：遮罩动画完成后，编辑器内容（标题/内容/底部栏）以 `translateY(16px → 0)` + `opacity: 0 → 1` 入场，200ms
- **顶部品牌色条**：从中间向两端展开（`scaleX(0 → 1)`，300ms，`transform-origin: center`）
- **关闭**：`scale(0.95)` + `opacity: 1 → 0`，180ms，遮罩同步淡出
- **ESC 关闭**：同上
- **点击遮罩关闭**：同上

### 7. 编辑笔记页面（编辑器模态框 - 编辑模式）
- **打开**：与新建页面相同的打开动画（缩放 + 淡入）
- **自动保存指示器**：保存时绿色圆点脉冲动画（`scale(1 → 1.3 → 1)`，500ms），仅保存完成时触发
- **标签选择交互**：选中/取消选中标签时，标签 chip 脉冲缩放（`scale(1 → 1.15 → 1)`，200ms），现有基础上微调时序
- **颜色选择**：点击颜色圆点时，选中态扩散动画（`scale(0.8 → 1.2 → 1)`，200ms）
- **字数统计**：数字变化时轻微颜色闪烁，150ms

### 8. 查看笔记页面（编辑器模态框 - 只读查看模式）
- **打开**：与编辑器相同的打开动画（缩放 + 淡入）
- **Markdown 渲染区域**：渲染完成后内容 `opacity: 0 → 1` 淡入，200ms
- **代码块加载**：highlight.js 着色完成后淡入，100ms
- **关闭**：与编辑器相同关闭动画

### 9. 关于页面
- **模态覆盖层入场**：与帮助页面一致 — 遮罩淡入 + 卡片缩放淡入
- **品牌 Logo**：Logo 文字入场时 `scale(0.8 → 1)` 弹性动画，300ms，弹簧缓动
- **版本号**：底部版本号淡入，延迟 100ms 出现
- **项目链接**：hover 时轻微上移 + 颜色过渡，150ms
- **关闭**：卡片缩放淡出 + 遮罩淡出
- **ESC/点击遮罩关闭**：同上

### 10. 笔记全屏动画（快速笔记模式）
- **进入全屏**：编辑器面板从原有位置展开到全屏（`scale(1)` + 位置过渡），300ms
- **退出全屏**：编辑器面板收缩回原位（`scale(0.95)` + 位置过渡），200ms
- **与现有 toggleEditorFullscreen 集成**：在现有函数基础上添加过渡动画

### 11. 批量模式过渡（跨页面）
- **进入批量模式**：
  - 批处理栏从底部滑入 `translateY(100% → 0)` + `opacity: 0 → 1`，250ms
  - 卡片复选框淡入显示，交错 20ms
  - topbar 批量操作按钮淡入，150ms
- **退出批量模式**：
  - 批处理栏滑出 `translateY(0 → 100%)` + `opacity: 1 → 0`，150ms
  - 卡片复选框淡出
  - topbar 批量操作按钮淡出

### 12. Toast 通知动画（跨页面）
- **入场**：从底部滑入 + 淡入（`translateY(24px → 0)` + `opacity: 0 → 1`，250ms）
- **离场**：滑出 + 淡出（`translateY(0 → -12px)` + `opacity: 1 → 0`，200ms）
- **多条堆叠**：旧 Toast 上移 56px（`translateY` 过渡，200ms）
- **撤销按钮**：hover 时轻微缩放，150ms

### 13. 右键菜单动画（跨页面）
- **打开**：从鼠标位置缩放展开（`scale(0.9 → 1)` + `opacity: 0 → 1`，150ms）
- **关闭**：缩放收起（`scale(1 → 0.95)` + `opacity: 1 → 0`，100ms）

### 14. 更多菜单动画
- **打开**：从触发按钮位置缩放展开（`scale(0.85 → 1)` + `opacity: 0 → 1`，150ms，`transform-origin: top right`）
- **关闭**：缩放收起，100ms

### 15. 主题切换过渡
- 所有受影响的 CSS 属性（background-color, color, border-color, box-shadow）添加 `transition: 300ms ease-out`
- 在 `:root` 和所有 `[data-theme]` 选择器上统一设置

---

## Impact

- **Affected specs**: 所有已有 UI 相关 spec
- **Affected code**:
  - `frontend/src/main.js` — `switchView()`, `renderCardGrid()`, `openEditor()`, `closeEditor()`, `toggleEditorFullscreen()`, `showContextMenu()`, `showToast()`, `enterBatchMode()`, `exitBatchMode()` 等函数
  - `frontend/src/style.css` — 新增动画 CSS class 和 keyframes
  - `frontend/src/app.css` — 添加动画 CSS 变量、transition 属性、prefers-reduced-motion 降级
- **不涉及**：后端 Go 代码、模型、业务逻辑

---

## ADDED Requirements

### Requirement: 全局动画基础设施
The system SHALL define centralized animation tokens and utility classes.

#### Scenario: CSS 动画变量
- **GIVEN** 页面加载
- **WHEN** CSS 被解析
- **THEN** `:root` 中定义了 `--anim-duration-fast: 150ms`, `--anim-duration-normal: 250ms`, `--anim-duration-slow: 300ms`, `--anim-easing-spring: cubic-bezier(0.16, 1, 0.3, 1)` 等变量

#### Scenario: 无障碍降级
- **GIVEN** 用户开启系统"减少动效"
- **WHEN** 任何动画被触发
- **THEN** `@media (prefers-reduced-motion: reduce)` 将所有动画时长设为 0ms

### Requirement: 视图切换动画
The system SHALL animate transitions between all main views (grid/data/trash/settings).

#### Scenario: 切换视图时
- **GIVEN** 用户在网格视图
- **WHEN** 点击"设置"进入设置视图
- **THEN** 当前视图淡出（150ms），设置视图淡入（250ms）
- **AND** 动画仅使用 `opacity` 和 `transform` 属性

### Requirement: 卡片网格动画
The system SHALL animate card grid with staggered entrance and interactive feedback.

#### Scenario: 加载笔记列表
- **GIVEN** 笔记数据已加载
- **WHEN** `renderCardGrid()` 被调用
- **THEN** 卡片逐个交错出现（40ms 间隔），animation: `translateY(20px)` + `opacity: 0 → 1` + `scale(0.97 → 1)`
- **AND** 最多 12 张卡片有延迟，其余同时出现

### Requirement: 编辑器/查看模态动画
The system SHALL animate editor modal (create/edit/view) with scale+fade transitions.

#### Scenario: 打开编辑器
- **GIVEN** 用户点击新建/编辑笔记
- **WHEN** `openEditor()` 被调用
- **THEN** 模态框缩放淡入（300ms）
- **AND** 遮罩同步淡入
- **AND** 内容区域延迟 50ms 后入场

### Requirement: 覆盖层模态动画（帮助/关于）
The system SHALL animate overlay modals (help/about) consistently.

#### Scenario: 打开帮助页
- **GIVEN** 用户点击"帮助"
- **WHEN** `openShortcuts()` 被调用
- **THEN** 遮罩淡入（200ms），卡片缩放淡入（300ms，弹簧缓动）
- **AND** 快捷键列表交错入场（30ms 间隔）

### Requirement: 快速笔记全屏过渡
The system SHALL animate fullscreen toggling for quick note mode.

#### Scenario: 进入全屏
- **GIVEN** 快速笔记模式已启用
- **WHEN** 新建笔记被触发
- **THEN** 编辑器面板展开到全屏（300ms）

### Requirement: 回收站操作动画
The system SHALL animate trash operations with visual feedback.

#### Scenario: 恢复单条笔记
- **GIVEN** 用户在回收站页面
- **WHEN** 点击"恢复"
- **THEN** 该项收缩淡出（200ms）
- **AND** 列表重新排列

### Requirement: 数据管理统计卡片动画
The system SHALL animate stats cards on the data management page.

#### Scenario: 进入数据管理页
- **GIVEN** 用户切换到数据管理视图
- **WHEN** 统计卡片被渲染
- **THEN** 三张卡片交错缩放淡入（80ms 间隔）
- **AND** 数字从 0 计数到实际值（首次加载时）

### Requirement: 设置页交互反馈
The system SHALL provide smooth interaction feedback on settings page.

#### Scenario: 开关切换
- **GIVEN** 用户在设置页面
- **WHEN** 点击 toggle 开关
- **THEN** 滑块平滑移动（200ms），背景色渐变过渡
