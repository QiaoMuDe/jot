# Checklist

## 后端协议
- [x] ask_user Schema 含 `selection`（single/multiple，缺省 single）
- [x] `ai:ask-user` 事件负载含 `selection` 字段（与 question/options 并列）
- [x] Desc 说明单选/多选使用场景
- [x] `go build ./...` 通过

## 前端面板
- [x] `#aiAskPanel` 容器存在于 `#aiChatInputArea` 内顶部（追问引用栏之前）
- [x] `renderAskCard` 及其调用已移除，无残留引用
- [x] `ai:ask-user` 监听回调改为显示面板（解析 question/options/selection）
- [x] 面板显示问句标题 + 选项区 + 自定义输入行
- [x] 单选：点击选项即发（`sendUserText`）+ 隐藏面板
- [x] 多选：选项勾选态（Set + 对勾图标），"确认提交"发送 `我选择：A、B` + 隐藏面板；未选且无输入时不发送
- [x] 自定义输入：Enter/提交发送 + 隐藏面板
- [x] 面板生命周期：`startStreaming` 开头、`switchSession`、清空会话时均隐藏
- [x] 新问题到达替换旧面板内容

## 样式
- [x] `.ai-ask-card` 系列样式已移除，`.ai-ask-panel` 系列已实现（含选中态/确认按钮）
- [x] 主题自适应（CSS 变量），`prefers-reduced-motion` 生效

## 历史回放
- [x] 历史回放不渲染面板（仅问句正文 + 工具折叠，agent.go 兜底不变）

## 构建验证
- [x] `go build ./...` 通过
- [x] 前端 `npm run build` 通过
- [x] TOOLS.md §7.1 已同步新协议与生命周期；AGENTS.md 记忆点滚动完成（1-10 连续）
