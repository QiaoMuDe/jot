# Agent 反问面板重设计（方案B：输入区上方 + 单选/多选 + 完成任务后隐藏）Spec

## Why
上一版 ask_user 问题卡片嵌在 assistant 气泡尾部，仅支持单选点击即发，长对话中易被忽略、多选场景无法表达。用户选定方案 B：交互面板移到输入区上方（composer 上方固定区域），支持单选/多选/自定义三种回答方式，面板完成任务后即隐藏/移除，不留在页面上。

## What Changes
- **后端协议扩展**（[ask_user.go](internal/agent/tools/ask_user.go)）：Schema 新增 `selection` 字段（string 枚举 `single`|`multiple`，缺省 `single`）；`ai:ask-user` 事件负载扩展为 `{"question", "options", "selection"}`；Desc 补充单选/多选使用说明。
- **前端面板迁移**：移除气泡内嵌卡片（`renderAskCard` → 删除），新增输入区上方固定面板 `#aiAskPanel`（位于 `#aiChatInputArea` 内顶部、追问引用栏之前）。
- **交互增强**：单选点选项即发（选中反馈）；多选选项可勾选（accent 实底 + 对勾），底部"确认提交"汇总发送（`我选择：A、B`）；自定义输入保留。
- **面板生命周期**：收到 `ai:ask-user` 填充并显示；用户回答（选择/提交/自定义）后隐藏；用户绕答直接发新消息、切换会话、清空会话时隐藏。
- **历史回放不变**：assistant 消息正文仍为问句（agent.go pendingQuestion 兜底已实现），面板不重现；工具调用链折叠照常展示。
- **文档更新**：TOOLS.md §7.1 同步新协议与生命周期；AGENTS.md 记忆点滚动追加。

## Impact
- Affected specs: agent 工具链路（ask_user 事件协议）、前端 AI 聊天交互（面板生命周期与消息流解耦）
- Affected code:
  - `internal/agent/tools/ask_user.go`（selection 字段 + 负载 + Desc）
  - `frontend/index.html`（`#aiAskPanel` 容器）
  - `frontend/src/js/ai-chat.js`（删 renderAskCard，新增面板渲染/交互/生命周期）
  - `frontend/src/css/components/ai-chat.css`（`.ai-ask-panel` 系列样式）
  - `internal/agent/TOOLS.md`（§7.1 更新）、`AGENTS.md`（记忆点滚动）

## ADDED Requirements

### Requirement: selection 多选协议
系统 SHALL 在 ask_user 工具中支持单选/多选声明。

- Schema 新增 `selection`（string，"single" | "multiple"，缺省 "single"）
- `ai:ask-user` 事件负载含 `selection` 字段（与 question/options 并列）
- Desc 说明：默认单选；需要用户多选决策（如多选笔记/标签/方案组合）时用 `multiple`

#### Scenario: Agent 发起多选提问
- **WHEN** Agent 调用 ask_user 且 selection=multiple
- **THEN** 前端面板进入多选模式（选项可勾选 + 确认按钮）

### Requirement: 输入区上方反问面板
系统 SHALL 在输入区上方渲染反问面板 `#aiAskPanel`，替代气泡内嵌卡片。

- 面板位于 `#aiChatInputArea` 内顶部（追问引用栏之前），与输入区视觉一体化
- 内容：问句标题 + 选项区 + 自定义输入行（+ 多选时确认按钮）
- 单选：点击选项即发（选中态 accent 反馈）
- 多选：点击选项切换勾选态（对勾 SVG + accent 实底），点"确认提交"汇总为 `我选择：A、B` 发送；发送前至少选中一项或输入自定义文本，否则提示
- 自定义输入：Enter/提交按钮发送输入文本
- 面板淡入动画（0.2s 内），`prefers-reduced-motion` 关闭

#### Scenario: 单选即时回答
- **WHEN** 用户在单选模式点击某个选项
- **THEN** 选项文本作为新 user 消息发送（`sendUserText`），面板隐藏

#### Scenario: 多选确认回答
- **WHEN** 用户勾选多项并点击"确认提交"
- **THEN** 以 `我选择：A、B` 格式作为新 user 消息发送，面板隐藏

### Requirement: 面板生命周期
系统 SHALL 保证反问面板在完成使命后即隐藏，不留在页面。

- 用户回答（单选/多选确认/自定义提交）后隐藏面板
- 用户绕答直接发送新消息（`startStreaming` 开始）时隐藏面板
- 切换会话、清空会话、AI 前端状态重置时隐藏面板
- 新问题到达时替换面板内容（同会话内只保留最新一个问题）

#### Scenario: 用户绕过面板直接发消息
- **WHEN** 面板存在但用户直接在输入框发送新消息
- **THEN** 面板隐藏，新消息正常进入新一轮 Agent 流

#### Scenario: 切换会话
- **WHEN** 用户切换到其他 AI 会话
- **THEN** 面板隐藏，不残留上一个会话的挂起问题

### Requirement: 历史回放退化（保持）
系统 SHALL 保证历史回放时反问交互以普通文本呈现，面板不重现。

- assistant 消息正文为问句（agent.go pendingQuestion 兜底不变）
- 回放仅渲染问句文本 + 工具调用链折叠，无交互控件

## MODIFIED Requirements
### Requirement: ask_user 工具（原）
原"问题卡片与选择发送"要求中"卡片保留在当前 assistant 气泡中，不随流结束移除"改为**面板在输入区上方、完成任务后隐藏**；"点击选项/提交自定义输入"保持新消息续流机制不变（`sendUserText`）。
**Reason**: 用户选定方案 B（显眼度与多选空间），且要求完成任务后不留页面。
**Migration**: 移除 `renderAskCard` 及气泡内卡片 DOM；事件协议向后兼容（新增 selection 字段，旧前端忽略即可）。

## REMOVED Requirements
### Requirement: 气泡内问题卡片
**Reason**: 被方案 B 面板替代（用户选定）。
**Migration**: `renderAskCard` 移除；卡片样式 `.ai-ask-card` 系列替换为 `.ai-ask-panel` 系列。
