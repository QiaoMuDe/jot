# 更多技能菜单固定高度与滚动支持

## Why

AI 助手工具栏的"更多技能"下拉菜单向上展开时，若窗口高度不足或工具栏位置偏上，菜单内容会被顶部视口遮挡，部分技能项无法查看和点击，影响用户体验。

## What Changes

- 给 `.ai-chat-skills-dropdown` 设置固定 `max-height`，超出时显示纵向滚动条
- 将 `overflow: visible` 改为 `overflow-y: auto`，支持列表滚动查看
- 移除或调整逐项动画的 `transition-delay`，避免因滚动容器导致动画异常
- 确保滚动条样式与应用的细滚动条风格一致（`scrollbar-width: thin`）
- 处理"翻译"技能的特殊行为：翻译项点击后展开子选项（方向选择），子选项使用 `max-height` 动画 + `overflow: hidden`，需确保在滚动父容器中正常展开/收起，不与 `overflow-y: auto` 冲突

## Impact

- Affected specs: AI 助手模块的 UI 交互
- Affected code: `frontend/src/css/components/ai-chat.css` 中 `.ai-chat-skills-dropdown` 相关样式

## ADDED Requirements

### Requirement: 固定高度与滚动
系统 SHALL 为"更多技能"下拉菜单设置最大高度，超出时显示纵向滚动条。

#### Scenario: 菜单内容过多
- **WHEN** 用户点击"更多技能"按钮展开下拉菜单
- **THEN** 菜单最大高度为 300px，内容超出时出现纵向滚动条
- **AND** 菜单项可滚动查看，不会被顶部视口遮挡

#### Scenario: 滚动条样式
- **WHEN** 菜单出现纵向滚动条
- **THEN** 滚动条使用 `scrollbar-width: thin` 保持与应用一致的细滚动条风格

## MODIFIED Requirements

### Requirement: 菜单展开动画
- 移除过长的逐项 `transition-delay`（当前最后一个项延迟 0.54s），改为更短的统一延迟或缩短延迟时间，避免滚动容器中子项动画不同步

## REMOVED Requirements

### Requirement: 旧的逐项动画延迟
**Reason**: 在滚动容器中，逐项滑入动画的延迟会导致部分项在可见区域外完成动画，视觉效果不一致
**Migration**: 缩短延迟时间，使动画更紧凑流畅