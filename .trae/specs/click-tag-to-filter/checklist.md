# Check List

- [x] `searchByTag` 函数改为接受 tagId/tagName/tagColor，调用 `GetNotesByTag` API
- [x] 卡片网格标签点击按标签 ID 筛选（非批量模式）
- [x] 搜索结果标签点击按标签 ID 筛选
- [x] 搜索视图顶部显示标签筛选指示器（颜色圆点 + 名称 + 关闭按钮）
- [x] 点击关闭按钮清空筛选回到网格首页
- [x] `switchView('grid')` 时清空 `state.filterTag`
- [x] 批量模式下点击标签不触发筛选
- [x] 后端不可用时 Mock 降级工作
- [x] 搜索弹窗功能不受影响