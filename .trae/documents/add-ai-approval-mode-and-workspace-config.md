# 方案：AI 助手工作目录 + 会话审批模式字段 + 前端审批模式选择器

## Summary

本轮只做三块基础设施，**不实现**文件系统/命令工具本身，也不做审批暂停/续跑机制（用户明确"先干完这些，再考虑后面的"）：

1. 工作目录：在 `~/.jot/` 下规划并自动创建 `workspace/`（新增配置常量 + 创建逻辑）。
2. 会话审批模式字段：在会话配置模型 / 传输结构 / 保存加载逻辑 / 前端加载保存逻辑中新增 `approval_mode`（三值：`confirm_every` / `auto` / `review`，默认 `confirm_every`）。
3. 前端 AI 助手界面：新增一个审批模式选择器组件，切换即持久化到该会话。

## Current State Analysis

* `~/.jot` 子目录统一由 [config/config.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/config/config.go) 管理，已有 `DirData/DirBackup/DirImages/DirLogs` 四个常量 + `JotHomeDir()`/`SubDir()`；现有子目录（如 data）在 [database/db.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/database/db.go#L29-L34) 用 `os.MkdirAll` 创建。

* 会话配置持久化链路已打通，含三层：

  * 模型 [models/ai\_session\_config.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/ai_session_config.go#L4-L13)（GORM，`AISessionConfig`）

  * 传输结构 [services/ai\_service.go#L46-L55](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L46-L55) `SessionConfig`

  * 保存/加载 [services/ai\_service.go#L549-L607](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L549-L607) `SaveSessionConfig` / `LoadSessionConfig`，其中 `Mode` 字段走 `modeOrDefault` 兜底（空/非法→`agent`），且保存时空值不覆写。

* 前端加载在 [ai-chat.js#L1976-L2007](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1976-L2007)（解析 `config.*`），保存全量写在 [ai-chat.js#L7612-L7625](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L7612-L7625) `saveCurrentSessionConfig`，另有单字段保存先例 `saveCurrentMode`（[ai-chat.js#L7598-L7607](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L7598-L7607)）。

* AI 助手顶栏在 [index.html#L1230-L1262](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1230-L1262)：`#aiModeToggle`（Chat/Agent/Plan）、`#aiChatAgentToolsWrap`（工具，仅 Agent/Plan 显隐）、`#aiChatSearchToggle`（深度思考）等。`.ai-mode-toggle`/`.ai-mode-btn` 样式在 [ai-chat.css#L1384](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1384) 附近。

## Proposed Changes

### 1. 工作目录（config + 创建）

* [config/config.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/config/config.go)：常量区新增
  `DirWorkspace = "workspace"`；新增 `WorkspaceDir() (string, error)`（`filepath.Join(JotHomeDir(), DirWorkspace)`）与
  `EnsureWorkspaceDir() error`（`os.MkdirAll(dir, 0755)`，幂等）。

* 在应用启动处调用 `EnsureWorkspaceDir()`（沿 [database/db.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/database/db.go#L29-L34) 创建 data 目录的既有模式；放在 app 初始化/DB 初始化附近，确保首次进入前目录已存在）。

### 2. 会话审批模式字段（后端三层 + 前端存取）

* [models/ai\_session\_config.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/ai_session_config.go#L13)：新增列
  `ApprovalMode string \`gorm:"size:20;default:'confirm\_every'" json:"approval\_mode"\`\`。GORM AutoMigrate 会自动加列；存量行默认空，靠加载兜底。

* [services/ai\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L46-L55)：`SessionConfig` 新增 `ApprovalMode string \`json:"approval\_mode"\`\`。

* [services/ai\_service.go#L549-L569](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L549-L569) `SaveSessionConfig`：`assign` 增加 `"approval_mode": cfg.ApprovalMode`；沿用 `Mode` 的"空值不覆写"防护（`if cfg.ApprovalMode != ""`）。

* [services/ai\_service.go#L571-L607](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L571-L607) `LoadSessionConfig`：返回结构加入 `ApprovalMode: approvalModeOrDefault(record.ApprovalMode)`；极端空默认分支补 `ApprovalMode: "confirm_every"`；新增 `approvalModeOrDefault(m string) string`：非三合法值一律回退 `confirm_every`（仿 `modeOrDefault`，[L601-L607](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L601-L607)）。

* 前端加载 [ai-chat.js#L1977-L2007](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1977-L2007)：读取 `approvalMode = config.approval_mode || 'confirm_every'`，并调用新的同步函数刷新组件选中态。

* 前端保存 [ai-chat.js#L7615-L7623](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L7615-L7623) `saveCurrentSessionConfig`：请求体新增 `approval_mode: approvalMode`（全量覆写模式，必须携带）。

### 3. 前端审批模式选择器

采用"顶栏独立按钮 + 下拉分段选择"（与现有 `.ai-mode-toggle` 分段控件视觉一致，符合工具人习惯；后续审批机制接入时此处显隐逻辑可复用）。仅在 Agent/Plan 模式显示（与 [aiChatAgentToolsWrap](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1256-L1262) 一致的显隐入口，`chat` 隐藏）。

* \[index.html]：紧邻 `#aiChatAgentToolsWrap` 新增

  ```
  <div class="ai-chat-approval-wrap" id="aiChatApprovalWrap">
    <button class="ai-chat-toolbar-btn" id="aiChatApprovalBtn" title="执行审批模式" aria-expanded="false">
      <svg shield 图标></svg><span>审批</span>
    </button>
    <div class="ai-chat-approval-dropdown" id="aiChatApprovalDropdown">
      <!-- .ai-approval-item data-value=... + 小段分段控件 + 说明文案 -->
    </div>
  </div>
  ```

  下拉内含三选项分段：`confirm_every`（每次确认）/ `auto`（全自动）/ `review`（智能审批），每项带一行中文说明。

* \[js/ai-chat.js]：

  * 新增全局 `let approvalMode = 'confirm_every';`

  * 新增 `syncApprovalToggle()`：根据 `approvalMode` 置选中态、更新按钮文字。

  * 新增 `saveApprovalMode()`（仿 `saveCurrentMode`，[L7598-L7607](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L7598-L7607)）：`load → approval_mode = approvalMode → SaveSessionConfig`。

  * 绑定点击切换逻辑（含下拉开关、`aria-expanded`）、外部点击关闭。

  * 在下拉面板选择某项时 `approvalMode=value; syncApprovalToggle(); saveApprovalMode();`。

  * 显隐：在现有 `window.__setAiChatAgentToolsVis?.(...)` 处同步 `#aiChatApprovalWrap` 的显隐（仅 agent/plan 显示）。

  * `loadSessionConfig` 内调用 `syncApprovalToggle()` 刷新（见第 2 节）。

* \[css/ai-chat.css]：新增 `.ai-chat-approval-wrap / .ai-chat-approval-btn / .ai-chat-approval-dropdown / .ai-approval-item` 样式，采用主题自适应变量（`var(--*)`），视觉对齐现有工具按钮与下拉（参考 [ai-chat.css#L1384](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1384) 附近的分段控件与工具栏样式）。**预期加载 frontend-design / ui-ux-pro-max 设计规范约束落地，避免风格割裂。**

### 4. 重建

按项目约定重建前端资源与二进制：`cd frontend && npm run build`，再在根目录 `wails build`（CSS/HTML 变更需重建才生效，见项目内存约束）。

### 明确不做（后续阶段）

* 4 个文件/命令工具（`ls/read_file/write_file/edit_file/glob/execute`）与对应 `adk/filesystem` 后端、路径越界校验、命令黑名单。

* 审批暂停/续跑机制、`ApprovalWaiter`、`ApproveToolCall`、审批弹层。

* 本会话 `approval_mode` 字段会先落地并持久化，但**暂不接入任何执行逻辑**，仅为后续工具接入预留。

## Assumptions & Decisions

* 审批模式为**每会话**字段（与 `Mode` 同级），默认 `confirm_every`；非三合法值一律兜底 `confirm_every`。

* 选择器仅 Agent/Plan 模式显示，与既有工具按钮显隐一致。

* Wails 后端绑定签名不变（`SaveSessionConfig(uint, SessionConfig)`），新增字段不破坏桥接，无需手工改 wailsjs 生成文件。

* 工作目录创建幂等；工具本身与校验逻辑不在本轮范围。

## Verification

* 后端单测（新增）：`approvalModeOrDefault` 对三合法值 + 空/非法回退 `confirm_every`；`LoadSessionConfig` 对新字段默认值正确。

* 手动：`go build` + `npm run build` 通过。

* 手测：AI 助手 Agent 模式顶栏出现"审批"组件且 Agent/Plan 可见、chat 隐藏；切换三选项选中态跟随、下拉可开合、点击外部关闭；切换会话后再次进入，选择保留（经 `LoadSessionConfig` 回读）；重启后仍保留（持久化）。

* 确认 `~/.jot/workspace/` 目录在启动后已自动创建。

