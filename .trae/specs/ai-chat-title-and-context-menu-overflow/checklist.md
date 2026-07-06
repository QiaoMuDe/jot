# Checklist

- [x] `switchSession()` 设置了 `#aiChatTitle` 文本为当前会话标题
- [x] `showWelcome()` 将 `#aiChatTitle` 重置为 "AI 助手"
- [x] `createSession()` 将 `#aiChatTitle` 设置为 "AI 助手"（通过 showWelcome 间接调用）
- [x] 重命名会话后 `#aiChatTitle` 同步更新
- [x] 无会话时标题为 "AI 助手"
- [x] 有历史消息的会话加载后标题为会话标题（与侧栏一致）
- [x] 右键菜单在底部卡片处触发时完整可见，不溢出视口底部
- [x] 右键菜单在顶部卡片处触发时完整可见，不溢出视口顶部
- [x] 现有 `transform-origin` 逻辑保持不变
