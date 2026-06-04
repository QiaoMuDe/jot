# 编辑增强与撤销功能 Spec

## Why
提升笔记编辑体验：用户编辑时能看到字数反馈、内容自动保存不丢失、删除操作可撤销、笔记可快速复制。

## What Changes
- 笔记编辑器底部显示实时字数统计
- 右键菜单新增「复制笔记」选项，一键创建内容相同的笔记
- 编辑器内容变化 3 秒后自动调用 UpdateNote 保存
- 删除笔记时弹出 Toast"已删除，点击撤销"，5 秒内可恢复

## Impact
- Affected specs: 笔记编辑、回收站/删除、右键菜单
- Affected code:
  - `frontend/index.html` — 编辑器底部字数区域
  - `frontend/src/main.js` — 字数统计、自动保存、复制笔记、撤销删除
  - `frontend/src/style.css` — 字数统计样式、Toast 样式
  - `services/note_service.go` — 可能需要恢复方法
  - `app.go` — 无新增绑定，复用现有 API

## ADDED Requirements

### Requirement: 字数统计
编辑器底部 SHALL 实时显示当前笔记的字数和字符数。

#### Scenario: 编辑时显示字数
- **WHEN** 用户在编辑器输入或修改内容
- **THEN** 编辑器底部实时显示「字数：X ｜ 字符：Y」
- **WHEN** 编辑器关闭或清空
- **THEN** 字数统计归零

### Requirement: 笔记复制
右键菜单 SHALL 提供「复制笔记」选项，一键复制笔记内容为新笔记。

#### Scenario: 复制笔记
- **WHEN** 用户在笔记卡片上右键
- **THEN** 右键菜单显示「复制笔记」选项
- **WHEN** 用户点击「复制笔记」
- **THEN** 调用 CreateNote 以相同标题+内容+颜色创建新笔记，不复制标签
- **THEN** 刷新笔记列表，Toast 提示"已复制"

### Requirement: 自动保存
编辑器内容变化后 3 秒无操作 SHALL 自动保存。

#### Scenario: 编辑后自动保存
- **WHEN** 用户在编辑器中修改标题或内容
- **THEN** 启动 3 秒定时器
- **WHEN** 3 秒内无新输入
- **THEN** 自动调用 UpdateNote 保存
- **WHEN** 3 秒内有新输入
- **THEN** 重置定时器重新计时
- **WHEN** 笔记为新创建（无 ID）
- **THEN** 不触发自动保存

### Requirement: 回收站撤销
删除笔记后 SHALL 显示 Toast 提示，5 秒内可撤销恢复。

#### Scenario: 删除后撤销
- **WHEN** 用户执行删除操作
- **THEN** 实际调用 DeleteNote 软删除
- **THEN** 页面底部显示 Toast「笔记已删除，点击撤销」
- **WHEN** 用户在 5 秒内点击「撤销」
- **THEN** 调用 RestoreNote 恢复笔记
- **THEN** Toast 消失，刷新笔记列表
- **WHEN** 5 秒倒计时结束
- **THEN** Toast 自动消失，不作恢复
