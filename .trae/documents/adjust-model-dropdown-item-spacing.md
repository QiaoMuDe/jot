# 调整模型下拉菜单条目间距 Plan

## Summary
调整设置页和 AI 助手中模型下拉菜单每个条目之间的垂直间距，使其不再紧贴在一起，提升可读性和视觉舒适度。

## Current State Analysis

### 设置页模型下拉菜单
- **CSS 文件**: `frontend/src/css/components/dropdowns.css`
- **选择器**: `.theme-select-dropdown`（容器）和 `.theme-select-item`（条目）
- **容器 padding（展开时）**: `4px`（`.theme-select-dropdown.open`）
- **条目 padding**: `8px 12px`（`.theme-select-item`）
- **条目间距**: **0** — 相邻条目之间无 margin/gap，直接堆叠

### AI 助手快捷切换模型下拉菜单
- **CSS 文件**: `frontend/src/css/components/ai-chat.css`
- **选择器**: `.ai-chat-model-dropdown`（容器）和 `.ai-chat-model-dropdown .theme-select-item`（条目）
- **容器 padding**: 无
- **条目 padding**: `6px 12px`
- **条目间距**: **0** — 相邻条目之间无 margin/gap，直接堆叠

## Proposed Changes

### 1. 设置页下拉菜单条目间距
- **文件**: `frontend/src/css/components/dropdowns.css`
- **位置**: 在 `.theme-select-item` 定义之后
- **操作**: 新增 `.theme-select-item + .theme-select-item` 选择器，添加 `margin-top: 4px`
- **理由**: 使用相邻兄弟选择器（`+`），只对非第一个条目增加上边距，避免首条条目上方多出多余空间

### 2. AI 助手快捷切换下拉菜单条目间距
- **文件**: `frontend/src/css/components/ai-chat.css`
- **位置**: 在 `.ai-chat-model-dropdown .theme-select-item` 定义之后
- **操作**: 新增 `.ai-chat-model-dropdown .theme-select-item + .theme-select-item` 选择器，添加 `margin-top: 4px`
- **理由**: 与设置页同理，使用相邻兄弟选择器精准控制间距

## Assumptions & Decisions
- 间距值定为 **4px**，既不过大（避免下拉内容太多时滚动过多）也不过小（2px 视觉改善不明显）
- 使用 `margin-top` + `+` 相邻兄弟选择器而非容器 `gap`，因为：
  - 现有容器不是 `display: flex`
  - 改为 flex + gap 会改变现有布局行为，风险较高
  - 相邻兄弟选择器只影响条目之间，不影响首尾条目与容器的关系

## Verification
1. 打开设置页 → AI 设置 → 获取模型列表 → 打开模型下拉菜单，检查条目间距
2. 打开 AI 助手 → 点击模型切换下拉菜单，检查条目间距
3. 确认其他使用 `.theme-select-item` 的下拉菜单（主题选择、代码高亮主题选择、服务商选择）间距也同步改善
