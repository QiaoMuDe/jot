# 首页卡片标签单行化 实施计划

## Summary
首页卡片 `.card-tags` 改为永不换行的单行布局：容器裁切超宽标签、单个超长标签自动省略号，右侧时间不受影响；辅以右缘渐隐 mask 与悬停显示完整标签名。

## Current State
- [main-content.css#L198-L203](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/main-content.css) `.card-tags` 为 `flex-wrap: wrap`，标签放不下换行，footer 变两行挤压 190px 固定卡片
- [main-content.css#L206-L215] `.card-tag` 为 inline-block，无 nowrap，单标签字多时内部逐字折行
- `.card-time` 已有 `nowrap + flex-shrink: 0`（L222-L227），本身安全
- 标签渲染于 [main.js#L3479-L3485]，无 title 属性

## Changes
1. **main-content.css `.card-tags`**：`flex-wrap: nowrap` + `overflow: hidden` + `flex: 1`；加 `mask-image: linear-gradient(90deg, #000 85%, transparent)` 右缘渐隐
2. **main-content.css `.card-tag`**：加 `white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; min-width: 0`
3. **main.js#L3483**：card-tag span 增加 `title="${escapeHtml(tag.name)}"`，悬停显示完整标签名

## Assumptions & Decisions
- 纯 CSS 为主，仅标签处加一行 title 属性
- 不动 `.card-time`；mask 渐隐兜底多标签裁切的生硬感
- 标签可点击语义不变

## Verification
- `npm run build` + `wails build -skipbindings` 通过
- 构造多标签/超长标签笔记：卡片 footer 恒单行、时间右对齐不被挤压、悬停标签显示全文
