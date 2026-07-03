# Checklist

- [x] AISession 模型已新增 `IsPinned bool` 字段
- [x] AISessionSummary 已新增 `IsPinned bool` 字段
- [x] GetAISessions 排序逻辑已改为"置顶优先按标题 ASC，其余按 updated_at DESC"
- [x] 后端 TogglePinAISession API 已实现并绑定到 Wails
- [x] 前端删除按钮已替换为"更多"按钮（⋯）
- [x] 点击"更多"按钮弹出下拉菜单，包含"置顶/取消置顶"和"删除会话"
- [x] 点击菜单项执行对应操作，操作完成后菜单关闭
- [x] 点击菜单外部区域关闭菜单
- [x] 菜单定位在按钮附近，不超出侧栏边界
- [x] 置顶会话在列表顶部显示，带置顶图标和视觉区分
- [x] 多个置顶会话之间按标题升序排列
- [x] 置顶会话与普通会话之间有分隔线
- [x] 置顶/取消置顶操作后列表正确刷新
- [x] 现有删除功能不受影响
- [x] 流式输出中禁用置顶/删除操作
