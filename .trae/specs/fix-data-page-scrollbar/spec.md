# 修复数据管理页面滚动条 Spec

## Why
数据管理页面右侧的滚动条不是全屏的，它只出现在内容区域内部，而回收站、笔记、设置页面的滚动条都是从页面顶部到底部的全屏滚动条。这造成了不一致的滚动体验。

## What Changes
- 移除 `.data-content` 上的 `overflow-y: auto` 和 `scrollbar-gutter: stable`，让内容自然流入 `#mainContent` 的滚动容器
- 将 `.data-content` 上的 `flex: 1` 移除
- 从 `scrollbar.css` 中移除 `.data-content` 的自定义滚动条样式引用
- 确保 `.data-content-inner` 仍然正确居中显示

## Impact
- Affected specs: 数据管理页面布局
- Affected code: `frontend/src/css/components/data-view.css`, `frontend/src/css/scrollbar.css`

## Requirements
### Requirement: 数据管理页面使用全屏滚动条
系统 SHALL 移除数据管理页面的内部滚动容器，改为依赖 `#mainContent` 的全屏滚动。

#### Scenario: 滚动条表现
- **WHEN** 用户打开数据管理页面
- **THEN** 右侧滚动条从页面顶部延伸到底部，与回收站、笔记、设置页面一致

#### Scenario: 内容可正常滚动
- **WHEN** 数据管理页面内容超出视口高度
- **THEN** 用户可通过全屏滚动条滚动查看所有内容
