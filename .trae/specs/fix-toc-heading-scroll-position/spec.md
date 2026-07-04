# TOC 标题点击跳转定位不精准 Bug 修复

## Why

预览模式下，点击 TOC（目录大纲）中的标题项时，预览区的滚动定位不准确——标题经常跳转到预览区顶部过近位置（标题紧贴顶部边缘，上方无留白），给用户"跳到标题上面"的错觉。部分情况下还会因平滑滚动动画导致定位微偏，表现为"有时偏上、有时偏下"的不一致体验。

## Root Cause

TOC 点击处理函数使用 `matched.scrollIntoView({ behavior: 'smooth', block: 'start' })` 进行跳转，该 API 将标题元素的 **border-box 上边缘** 对齐到滚动容器的 **内容区上边缘**（padding 内侧）。导致：

1. 标题的 `margin-top: 1.5em` 被推到可视区之外（上方不可见）
2. 标题文本直接紧贴预览区顶部边缘，没有任何留白空间
3. 平滑滚动动画的微小抖动叠加后，用户感受到"偏上或偏下"的不一致

根本原因在于 **滚动容器缺少 `scroll-padding-top`**，该 CSS 属性专门用于为 `scrollIntoView` 等滚动操作提供顶部内边距偏移。

## What Changes

- 在 `.md-rendered`（预览区滚动容器）的 CSS 中增加 `scroll-padding-top` 属性
- 确保 TOC 点击跳转时标题上方有合理的留白空间，定位准确一致

## Impact

- Affected specs: 编辑器预览模块、TOC 侧栏模块
- Affected code:
  - `frontend/src/css/components/editor.css` — `.md-rendered` 增加 `scroll-padding-top`

## ADDED Requirements

### Requirement: TOC 点击跳转精确定位

The system SHALL 保证点击 TOC 标题项后，预览区精准滚动到对应标题位置，标题上方有适当留白。

#### Scenario: 点击 TOC 标题项
- **WHEN** 用户点击 TOC 侧栏中的任意标题项
- **THEN** 预览区平滑滚动到该标题位置
- **THEN** 标题上方留有约 `0.75rem` 的留白空间（与预览区自身 `padding-top` 一致）
- **THEN** 标题文本不紧贴预览区顶部边缘

#### Scenario: 重复点击同一标题
- **WHEN** 用户再次点击同一个 TOC 标题项
- **THEN** 定位行为与首次点击完全一致，无偏移

#### Scenario: 不同标题层级
- **WHEN** 点击 H1~H6 任意层级的标题
- **THEN** 定位精度一致，不因标题层级不同而产生差异

## MODIFIED Requirements

暂无

## REMOVED Requirements

暂无
