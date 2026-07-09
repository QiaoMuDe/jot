# 合并引用笔记与上传文件按钮 Spec

## Why
工具栏按钮过多（引用、上传、模型选择、联网搜索、技能、发送等），操作栏拥挤。将功能关联的"引用笔记"和"上传文件"两个具有相同"引用/附件"语义的按钮合并到一个"+"菜单中，释放横向空间。

## What Changes
- **HTML**: 删除单独的 `#aiChatRefBtn` 和 `#aiChatFileBtn`，替换为一个 `#aiChatAddBtn`（显示"+" + 文字"添加" + 下拉箭头 SVG）
- **CSS**: 新增 `.ai-chat-add-wrap`、`.ai-chat-add-dropdown`、`.ai-chat-add-item` 样式，完全复用现有下拉菜单的弹出/关闭动画体系
- **JS**:
  - 新增 `addBtn`/`addDropdown` 变量 + DOM 获取 + 事件绑定（open/close/document click 同技能菜单模式）
  - 菜单内"引用"项点击 → 调用 `openNoteRefModal()`
  - 菜单内"上传"项点击 → 调用上传文件异步逻辑
  - 引用按钮的 `.has-ref` 高亮状态迁移到"+"按钮上

## Impact
- Affected specs: AI Chat 工具栏布局
- Affected code: `index.html`（删除两个按钮 + 新增菜单结构）、`ai-chat.css`（新增菜单样式）、`ai-chat.js`（新增变量/事件/逻辑迁移）

## ADDED Requirements

### Requirement: "+" 菜单按钮
The system SHALL provide a single "+" button replacing the separate "引用" and "上传" buttons.

#### Scenario: 默认显示
- **WHEN** 页面加载完成且 AI 聊天工具栏渲染
- **THEN** 原有引用按钮和上传按钮位置显示一个"+"按钮，包含"+" SVG 图标 + "添加"文字 + 下拉箭头
- **THEN** 点击该按钮时，向上弹出一个包含"引用笔记"和"上传文件"两个选项的菜单

#### Scenario: 菜单交互
- **WHEN** 用户点击 "+" 按钮
- **THEN** 菜单以向上弹出动画打开（opacity + translateY + scale，同现有下拉菜单）
- **THEN** 菜单项逐项滑入（translateX + transition-delay，每项间隔约 40ms）
- **WHEN** 用户点击菜单外任意区域
- **THEN** 菜单关闭（反向动画）
- **WHEN** 用户点击菜单内的"引用笔记"
- **THEN** 打开笔记引用选择器，菜单关闭
- **WHEN** 用户点击菜单内的"上传文件"
- **THEN** 触发原生文件选择对话框，菜单关闭

#### Scenario: 高亮状态
- **WHEN** 已有引用笔记（referencedNotes.length > 0）或已有上传文件（uploadedFiles.length > 0）
- **THEN** "+" 按钮添加 `.has-ref` 类，颜色变为主题色（`var(--accent)`）
- **WHEN** 所有引用和文件清除
- **THEN** `.has-ref` 类移除，恢复默认颜色

#### Scenario: 菜单项样式
- **WHEN** 菜单打开
- **THEN** 每个菜单项包含对应 SVG 图标 + 文字，悬停时高亮（同现有下拉菜单项样式）

### Requirement: 移除旧按钮
The system SHALL remove the standalone `#aiChatRefBtn` and `#aiChatFileBtn` from the toolbar.

- **WHEN** 页面加载
- **THEN** 工具栏中不再显示独立的"引用"和"上传"按钮
- **THEN** 相关的引用栏 (`#aiChatRefBar`) 和文件栏 (`#aiChatFileBar`) 保持不变，显示位置不受影响

## REMOVED Requirements
### Requirement: 独立的引用按钮和上传按钮
**Reason**: 合并到"+"菜单以释放工具栏空间
**Migration**: 引用的 `has-ref` 状态迁移到"+"按钮；引用模态框打开和文件上传逻辑迁移到菜单项点击事件
