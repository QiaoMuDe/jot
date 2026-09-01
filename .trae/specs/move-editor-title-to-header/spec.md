# 编辑器标题迁移至顶栏（双击编辑）Spec

## Why
当前笔记标题以大号输入框形式占据编辑器 body 顶部，视觉冗余且与"标题只在首页展示"的诉求冲突。将标题收敛到编辑器顶栏左侧、以"伪静态文字 + 双击编辑"的形式呈现，可统一新建/编辑/查看三种模式的体验，且不改变任何数据流。

## What Changes
- 将 `#editorNoteTitle` 输入框从 editor-body 顶部移入 `.editor-header` 左侧（顶栏布局变为：左标题、右操作按钮）
- 标题默认呈现为"伪静态文字"样式（透明背景、无边框、ellipsis 截断），双击进入编辑态
- 编辑/新建模式：双击可编辑，Enter 或失焦提交，Esc 取消恢复原标题；hover 显示可编辑提示（淡下划线 + tooltip「双击编辑标题」）
- 查看模式：标题为纯文本，双击无效，无可编辑提示（复用现有 `readOnly` + `editor-input-readonly` 机制）
- 编辑器 body 顶部原标题输入区移除，标签选择器上移，需补偿间距
- 数据流零改动：`_defaultNewNoteTitle`、`_editSnapshot.title`、closeEditorSafe 脏检测、createNote/updateNote 校验逻辑全部沿用

## Impact
- Affected specs: editor-header-compact（顶栏布局）、add-save-success-notification（保存流程不变，无影响）
- Affected code:
  - `frontend/index.html`（editor-header 结构、editorNoteTitle 位置）
  - `frontend/src/css/components/editor.css`（header 布局、标题伪文字样式、编辑态样式、body 间距）
  - `frontend/src/main.js`（openEditor 中标题填充位置不变；新增双击/Enter/Esc/blur 交互；switchEditorReadOnly 同步）

## ADDED Requirements

### Requirement: 顶栏标题展示
编辑器顶栏左侧 SHALL 常驻显示当前笔记标题；右侧操作按钮区保持不变。

#### Scenario: 新建笔记打开编辑器
- **WHEN** 用户点击新建按钮打开编辑器
- **THEN** 顶栏左侧显示自动生成的默认标题（`YYYY-MM-DD HH:MM ☺️`），与原 editor-body 顶部行为一致

#### Scenario: 编辑/查看已有笔记
- **WHEN** 用户从首页卡片打开笔记
- **THEN** 顶栏左侧显示该笔记原标题

#### Scenario: 标题超长
- **WHEN** 标题超过顶栏可用宽度
- **THEN** 默认态以 ellipsis 截断，hover 时通过原生 title 属性可查看完整标题

### Requirement: 双击编辑标题
编辑/新建模式下，用户 SHALL 可通过双击顶栏标题进入编辑态。

#### Scenario: 双击进入编辑
- **WHEN** 用户在编辑/新建模式下双击顶栏标题
- **THEN** 标题变为可输入状态并获得焦点，光标定位便于修改

#### Scenario: 提交修改
- **WHEN** 编辑态下按 Enter 或点击标题以外区域（blur）
- **THEN** 提交新标题并退出编辑态；若提交时标题为空（trim 后），恢复为进入编辑态前的原标题，不保存空值

#### Scenario: 取消修改
- **WHEN** 编辑态下按 Esc
- **THEN** 放弃修改，恢复进入编辑态前的原标题并退出编辑态

#### Scenario: 查看模式双击无效
- **WHEN** 用户在查看（只读）模式下双击顶栏标题
- **THEN** 标题不进入编辑态，保持纯文本展示

#### Scenario: 保存拦截兜底
- **WHEN** 用户通过其他途径使标题为空并触发保存
- **THEN** 沿用现有校验，提示「标题不能为空，请输入标题后再保存」

### Requirement: 可编辑性提示
编辑/新建模式下标题 SHALL 有可编辑的视觉暗示；查看模式 SHALL 无此暗示。

#### Scenario: hover 提示
- **WHEN** 用户在编辑/新建模式下将鼠标悬停在顶栏标题上
- **THEN** 显示淡下划线提示，鼠标指针为 text，且有 tooltip「双击编辑标题」

#### Scenario: 查看模式无提示
- **WHEN** 用户在查看模式下悬停标题
- **THEN** 无下划线提示，鼠标指针为 default

### Requirement: 只读状态同步
标题编辑能力 SHALL 跟随编辑器查看/编辑模式切换同步。

#### Scenario: 查看切编辑
- **WHEN** 用户在查看模式点击「编辑」按钮
- **THEN** 标题切换为双击可编辑状态（与 switchEditorReadOnly 现有 readOnly 切换联动）

#### Scenario: 编辑切查看
- **WHEN** 用户点击「保存并查看」
- **THEN** 标题退出编辑态并切换为纯文本展示

## MODIFIED Requirements

### Requirement: 编辑器 body 布局
editor-body 顶部 SHALL 不再包含标题输入框；标签选择器为 body 首个元素，其间距需补偿标题移除后的视觉空缺。

#### Scenario: 打开编辑器检查间距
- **WHEN** 打开任意模式的编辑器
- **THEN** 标签选择器位于 body 顶部，与顶栏的间距与移除标题前视觉协调，无突兀贴近

## REMOVED Requirements

### Requirement: body 顶部大标题输入框
**Reason**: 标题职责已迁移至顶栏，双输入位置会造成数据源二义。
**Migration**: 原 `#editorNoteTitle` 元素整体移入 `.editor-header`，id 不变，所有 `els.editorNoteTitle` 引用（保存、快照、脏检测、默认标题、MD 语法示例等 20 余处）无需修改。

## 非目标（Out of Scope）
- 不修改后端 title 字段、保存校验、向量分块元信息逻辑
- 不做首页卡片行内编辑标题（保持现状）
- 不引入"从正文提取标题"能力（方案 B，后续可选）
