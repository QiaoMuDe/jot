# Checklist

- [x] `Message` 结构体新增 `ReasoningContent` 字段
- [x] `LoadAISessionMessages` 返回的消息包含 `ReasoningContent`
- [x] `addMessage` 支持 `reasoningContent` 参数，AI 消息渲染思考区域
- [x] `switchSession` 先清空消息列表再加载
- [x] `switchSession` 正确判断 `msg.role`，角色不会错乱
- [x] `onAIChatViewActivated` 只在 `activeSessionId === null` 时自动加载
- [x] 侧栏默认折叠（`.collapsed` class）
- [x] 侧栏折叠/展开有 transition 动画
- [x] 折叠按钮在侧栏右边框，位置正确
- [x] 点击折叠按钮切换状态
- [x] 折叠状态通过 localStorage 持久化
- [x] `wails build -s` 编译通过
- [x] `npx vite build` 前端构建通过
