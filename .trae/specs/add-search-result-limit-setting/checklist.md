# Checklist

- [x] SearchWeb 函数签名新增 maxResults int 参数，替换硬编码 5
- [x] app.go 中 SearchWeb 调用处已传入 maxResults
- [x] 后端 GetAISearchResultLimit 绑定返回 int，空值时返回 5
- [x] 后端 SetAISearchResultLimit 绑定含范围校验（1-20）
- [x] 前端 HTML 新增搜索结果数输入框（id: aiSearchResultLimit）
- [x] 前端 loadAISettings 中调用 GetAISearchResultLimit 并填入输入框
- [x] 前端输入框 change 事件自动调用 SetAISearchResultLimit 保存
