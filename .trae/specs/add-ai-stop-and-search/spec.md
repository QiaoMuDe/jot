# AI 助手 — 停止生成 + 对话搜索 Spec

## Why

- **停止生成**: 当前流式输出时用户无法中断过长或错误的回复，只能干等流式结束，体验不佳
- **对话搜索**: 会话列表只按更新时间排序，会话一多难以快速找到特定对话

## What Changes

### 停止生成
- 后端：`CallAIStream` 支持 context 取消，新增 `CancelAIStream` 绑定方法
- 前端：流式输出时发送按钮变为停止按钮，点击后取消当前流式请求
- 保持已接收的流式内容不丢失，不再继续追加

### 对话搜索
- 前端：会话侧栏顶部新增搜索输入框
- 实时过滤：输入关键词后按会话标题前端过滤
- 纯前端实现，无需后端改动

## Impact

- Affected specs: AI 助手功能
- Added: `app.go` 新增 `CancelAIStream` 方法
- Modified: `internal/services/ai_service.go` 的 `CallAIStream` 签名/实现
- Modified: `frontend/src/js/ai-chat.js` 的流式处理和会话列表渲染
- Modified: `frontend/index.html` 的 AI 聊天 UI

## Requirements

### Requirement: 停止生成

#### Scenario: 点击停止按钮取消流式输出
- **WHEN** 用户在流式输出过程中点击停止按钮
- **THEN** 流式请求被取消
- **THEN** 已接收的内容保留在消息列表中
- **THEN** 不再追加新内容
- **THEN** 输入框恢复正常可用状态

#### Scenario: 流式正常结束后停止按钮隐藏
- **WHEN** 流式输出正常结束
- **THEN** 停止按钮恢复为发送按钮
- **THEN** 其他功能如常

### Requirement: 对话搜索

#### Scenario: 按标题搜索会话
- **WHEN** 用户在侧栏搜索框中输入关键词
- **THEN** 会话列表实时过滤，仅显示标题包含关键词的会话
- **THEN** 匹配的关键词可高亮

#### Scenario: 清空搜索恢复列表
- **WHEN** 用户清空搜索框内容
- **THEN** 会话列表恢复显示所有会话

#### Scenario: 创建新会话后搜索状态
- **WHEN** 用户在搜索状态下创建新会话
- **THEN** 新会话出现在列表中（如果匹配搜索条件）
- **THEN** 搜索框不清空