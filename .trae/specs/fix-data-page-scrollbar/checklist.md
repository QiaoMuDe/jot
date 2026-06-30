# Checklist
- [x] 数据管理页面右侧滚动条从页面顶部延伸到底部（逻辑确认：`.data-content` 不再有内部滚动条，内容流入 `#mainContent`）
- [x] `.data-content` 不再有内部滚动条（`overflow-y: auto` 和 `scrollbar-gutter: stable` 已移除）
- [x] `scrollbar.css` 中不再引用 `.data-content` 作为滚动容器
- [x] 数据管理内容仍然正确居中显示（`max-width: 760px; margin: 0 auto` 保持不变）
- [x] 回收站、笔记、设置页面的滚动条不受影响（仅修改数据管理相关代码）
- [x] `#viewData.view` 的 `padding-right: 0` 已移除，与其他视图保持一致的 `32px` 右内边距
- [x] `main.js` 中不再将 `els.dataContent` 作为滚动容器处理
