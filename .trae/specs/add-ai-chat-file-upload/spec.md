# AI 聊天上传本地文件 Spec

## Why

AI 助手当前仅支持纯文本输入，用户无法将本地文件内容作为上下文提供给 AI。需要新增文件上传功能，让用户可以选择本地文本文件，将其内容注入 AI 提示词，实现基于本地文件的问答、分析等场景。

## What Changes

- **后端 `app.go`** — 新增 `SelectAIChatFiles()` 方法：打开原生文件多选对话框，对每个文件执行大小校验（10MB）、二进制检测（`IsBinaryPath`）、读取内容、按 `ai_ref_max_chars`（每次从 DB 实时读取）截断，返回结果列表
- **前端 `index.html`** — AI 聊天工具栏新增"上传文件"按钮（自定义 SVG 图标），新增文件 chips 展示栏
- **前端 `ai-chat.css`** — 上传按钮和文件 chips 样式（复用引用笔记 chips 的 CSS 类体系）
- **前端 `ai-chat.js`** — 上传交互逻辑：点击按钮触发后端方法 → 展示文件 chips（复用 refChips 渲染模式）→ 3 个文件以上显示"清除所有"按钮 → 发送时将文件内容拼入 `systemContext`

## Impact

- Affected specs: AI 助手交互
- Affected code:
  - `app.go` — 新增 `SelectAIChatFiles()` 方法和 `AIChatFileResult` 结构体
  - `frontend/index.html` — AI 聊天工具栏和输入区
  - `frontend/src/css/components/ai-chat.css` — 新增样式（复用引用 chips 类）
  - `frontend/src/js/ai-chat.js` — 上传逻辑和消息拼接

## ADDED Requirements

### Requirement: 后端文件选择与读取（每次从 DB 取截断阈值）

The system SHALL provide a Go method `SelectAIChatFiles() ([]AIChatFileResult, error)` that:

1. Opens a native file dialog via `runtime.OpenFileDialog` with multi-select enabled, title "选择要上传的文本文件"
2. Accepts all file types (user can select any file; validation happens per-file)
3. For each selected file:
   - **Size check**: If file size > 10MB, this file's result error = "文件过大（超过 10MB），请选择小于 10MB 的文件"
   - **Binary check**: If `fs.IsBinaryPath()` returns true, this file's result error = "不支持二进制文件，请选择文本文件"
   - **Read**: Read file content via `os.ReadFile`
   - **Truncate**: Call `GetAIRefMaxChars()` (每次实时从 DB 读取，不缓存) 获取截断阈值 N。If content length > N, truncate to first N chars and append `"\n\n...(内容已截断，完整文件共 X 字)"`，设置 `Truncated = true`
4. Returns `[]AIChatFileResult` with fields: `Path`, `Name`, `Content`, `Size`, `Truncated`, `Error`

#### Scenario: 成功选择并读取多个文本文件
- **WHEN** 用户点击上传按钮，在文件对话框中选择了 2 个小于 10MB 的文本文件
- **THEN** 后端返回包含 2 个 `AIChatFileResult` 的数组，每个包含完整文件内容和元数据

#### Scenario: 文件超过大小限制
- **WHEN** 用户选择了一个 15MB 的文本文件
- **THEN** 该文件的 `AIChatFileResult.Error` 返回 "文件过大（超过 10MB），请选择小于 10MB 的文件"，不影响其他文件

#### Scenario: 文件为二进制文件
- **WHEN** 用户选择了一个 `.exe` 或 `.png` 文件
- **THEN** 该文件的 `AIChatFileResult.Error` 返回 "不支持二进制文件，请选择文本文件"

#### Scenario: 文件内容超过截断字数
- **WHEN** 用户选择了一个 2 万字的文本文件，DB 中 `ai_ref_max_chars` 当前为 5000
- **THEN** 返回的 `Content` 为前 5000 字 + `"\n\n...(内容已截断，完整文件共 20000 字)"`，`Truncated` 为 `true`

#### Scenario: 用户取消对话框
- **WHEN** 用户在文件对话框中点击取消
- **THEN** 返回空的 `[]AIChatFileResult`（无错误）

#### Scenario: 截断阈值实时读取
- **WHEN** 用户先通过设置页将 `ai_ref_max_chars` 从 5000 改为 1000，然后立即上传文件
- **THEN** 后端按 1000 字截断，不使用任何缓存值

### Requirement: 前端上传按钮与交互（复用引用 chips 模式）

The system SHALL provide an upload button in the AI chat toolbar that:

1. **按钮位置**：在工具栏"引用笔记"按钮之后，保持 `.ai-chat-toolbar-btn` 样式
2. **图标**：自定义设计的上传相关 SVG 图标（符合场景气质），title 为 "上传文件"
3. **点击行为**：调用 `window.go.main.App.SelectAIChatFiles()`，期间按钮置 disabled 防重复点击
4. **结果处理**：
   - 有错误的文件：`window.showNotification(error, 'error')` 依次提示
   - 成功的文件：追加到文件列表中，重新渲染 chips 栏
5. **文件 chips 栏 `#aiChatFileBar`**：
   - **直接复用引用笔记的 CSS 类体系**：`.ai-chat-ref-bar` → `.ai-chat-file-bar`（样式复用，语义区分）、`.ai-chat-ref-chip` → `.ai-chat-file-chip`、`.ai-chat-ref-chip-remove` → `.ai-chat-file-chip-remove`
   - 位置：技能指示条 `#aiChatSkillBar` 之后、输入行之前
   - 初始 `display:none`
   - 每个 chip 显示：文件图标 SVG + 文件名 + 文件大小（如 "config.json (2.3 KB)"）+ 截断标记 `(内容已截断)` + 删除按钮 (×)
   - 支持多文件同时展示
   - 点击 × 从列表中移除该文件
   - 点击上传按钮继续追加文件（不覆盖已有列表）
6. **批量清除**：当文件数量 ≥ 3 时，在 chips 末尾追加"清除所有上传文件"按钮（复用 `.ai-chat-ref-chip-remove-all` 样式），点击后清空全部文件

#### Scenario: 上传成功展示文件标签
- **WHEN** 用户选择两个文件上传成功
- **THEN** 输入区上方显示文件 chips 栏，包含两个文件 chip，每个显示文件图标、文件名和大小

#### Scenario: 部分文件失败
- **WHEN** 用户选择了 3 个文件，其中 1 个是二进制文件
- **THEN** 成功的 2 个文件显示 chips，失败的 1 个弹出错误通知

#### Scenario: 删除已选文件
- **WHEN** 用户点击某个文件 chip 的 × 按钮
- **THEN** 该文件从列表中移除，如果列表为空则隐藏 chips 栏

#### Scenario: 批量清除上传文件
- **WHEN** 文件列表数量 ≥ 3
- **THEN** chips 末尾显示"清除所有上传文件"按钮（虚线边框、红色样式）
- **WHEN** 用户点击"清除所有上传文件"
- **THEN** 全部文件从列表移除，隐藏 chips 栏

### Requirement: 文件内容注入 AI 提示词

The system SHALL inject uploaded file contents into the AI context when sending a message:

1. **拼接位置**：在 `onSend()` 中，将文件内容拼入 `systemContext`（与笔记引用、追问引用同级，按顺序：笔记引用 → 追问引用 → 文件内容）
2. **拼接格式**：
   ```
   用户上传了以下文件内容，请基于这些内容回答用户的提问：
   
   --- 文件: filename.txt (2.3 KB) ---
   {file content}
   ---
   ```
3. **多文件**：多个文件依次拼接，用 `---` 分隔
4. **内容截断标记**：如果文件 `truncated === true`，在文件块末尾文件大小后追加 `(内容已截断，完整文件共 X 字)`
5. **发送后清空**：消息发送后自动清空 `uploadedFiles` 列表并隐藏 chips 栏

#### Scenario: 带文件发送消息
- **WHEN** 用户上传了一个文件后输入问题并点击发送
- **THEN** 文件内容按格式拼入 `systemContext`，随消息一起发送给 AI，发送后文件列表自动清空

#### Scenario: 多文件 + 笔记引用同时存在
- **WHEN** 用户同时引用了笔记和上传了多个文件
- **THEN** 文件内容拼在笔记引用内容之后，一起作为 `systemContext`
