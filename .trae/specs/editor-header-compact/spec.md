# 编辑器 Header / 标题 / 标签区垂直压缩 Spec

## Why
当前编辑器的头部和标题区域占用过多顶部空间，与用户追求「内容区最大化」的诉求冲突。具体来说：`.editor-header` 顶部 12px padding、标题 8px 上下 padding、标签区 24px 底部 margin 共同导致核心内容区（CodeMirror / Markdown 预览）被挤压，在小窗口编辑时一行都写满就滚动。

## What Changes
- `.editor-header` 的 `padding-top` 从 `12px` 降低到 `4px`（省 8px）
- `.editor-input`（标题）的 `padding` 从 `8px 0` 降低到 `2px 0`（省 12px）
- `.editor-section`（标签容器）的 `margin-bottom` 从 `24px` 降低到 `6px`（省 18px）
- 不引入额外间距：标签和正文之间保持 6px 净间距（与 `.editor-body` 的 `padding-bottom: 8px` 配合，正文底部仍留 8px）

合计省 **38px** 垂直空间（与「激进压缩」选择一致），顶部和内容区视觉重心明显上移。

## Impact
- Affected specs: 无（这是视觉微调，不涉及功能变化）
- Affected code:
  - `frontend/src/style.css` — 3 处规则各改 1 个值（L967-972、L1044-1050、L1136-1138）

## ADDED Requirements

### Requirement: 编辑器顶部区域垂直压缩
系统 SHALL 压缩编辑器头部、标题、标签三个区域的总占用高度，让核心内容区（CodeMirror / Markdown 预览）多获得 38px 垂直空间。

#### Scenario: header padding 降低
- **WHEN** 用户打开任意笔记（编辑/新建/查看模式）
- **THEN** 顶部 4 个控制按钮（编辑、查看模式切换、全屏、关闭）的上间距为 4px（之前 12px）
- **AND** 按钮与标题之间的距离略收紧（标题 padding 同步降低）

#### Scenario: 标题 padding 降低
- **WHEN** 用户在编辑模式下看到标题输入框
- **THEN** 标题上下内边距为 2px（之前 8px）
- **AND** 标题底部与标签之间的距离明显收紧
- **AND** 标题字号、字重、颜色不变
- **AND** 标题输入框的 focus 高亮（border-bottom accent）正常显示

#### Scenario: 标签区 margin 降低
- **WHEN** 用户看到笔记的标签 chip 列表
- **THEN** 标签底部 margin 为 6px（之前 24px）
- **AND** 标签和正文（CodeMirror / 预览区）之间的间距为 6px
- **AND** 不影响标签 chip 自身的 padding / border / 颜色

#### Scenario: 内容区多获空间
- **WHEN** 压缩生效后
- **THEN** CodeMirror 编辑器在固定窗口高度下可见行数 +1
- **AND** Markdown 预览模式下首屏可见内容增加
- **AND** 三种模式（编辑/新建/查看）都生效

## MODIFIED Requirements
无（这是视觉调整，不修改任何功能需求）。

## REMOVED Requirements
无（没有移除任何功能）。
