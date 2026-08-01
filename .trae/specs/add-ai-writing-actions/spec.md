# AI 写作操作菜单 Spec

## Why

编辑器操作菜单已有格式化、文本转换、文本清理、编码解码、MD 语法 5 个分组，但缺少 AI 驱动的写作辅助功能。用户在编辑器中写作时，频繁需要润色、续写、扩写等操作，当前只能切换到 AI 对话页面手动操作，流程割裂。在操作菜单中直接集成 AI 写作操作，可以选中文本后一键调用 AI 处理，结果直接替换到编辑器中，体验更流畅。

## What Changes

- 后端新增 `AITextOperation(text, operation string) (string, error)` 绑定，加载 AI 配置后构造对应 prompt 调用 `aiService.CallAI`，复用已有的 `aiStreamCancel` + `CancelAIStream()` 取消机制
- 前端 `executeAction` 改为 `async`，支持异步 handler（`await` 后向兼容）
- 新建 `frontend/src/js/editor-actions/ai-writing.js`，定义 8 个 AI 写作操作项
- 前端 `editor-actions.js` 导入并展开 `AI_WRITING_ACTIONS`
- 编辑器面板新增 `#aiStatusBar`（编辑器内持久状态栏），AI 处理中显示 spinner + 文字 + 取消按钮，处理完毕后自动移除
- 前端调用 `window.go.main.App.CancelAIStream()` 取消正在进行的 AI 操作

## Impact

- Affected specs: 编辑器操作菜单（新增 AI 写作分组）
- Affected code:
  - `app.go`（新增 `AITextOperation` 绑定 + prompt 构造逻辑）
  - `frontend/src/js/editor-actions.js`（executeAction 改为 async + 导入新模块）
  - `frontend/src/js/editor-actions/ai-writing.js`（新建）
  - `frontend/src/css/components/editor.css`（新增 `#aiStatusBar` 样式）
  - 通知系统 `notification.js`（新增 `showPersistent` 方法，或使用现有 `showAction` + 手动取消）

## ADDED Requirements

### Requirement: 后端 AITextOperation 绑定

系统 SHALL 提供一个新的 Wails 绑定 `AITextOperation(text, operation string) (string, error)`。

#### Scenario: 成功调用
- **WHEN** 前端调用 `window.go.main.App.AITextOperation("选中的文本", "polish")`
- **THEN** 后端读取当前 AI 配置，构造对应 system prompt，调用 `aiService.CallAI(ctx, messages)`，返回处理后的文本

#### Scenario: 未配置 AI
- **WHEN** AI 未配置 API Key 或 Base URL
- **THEN** 返回错误提示"请先配置 AI 服务"

#### Scenario: 网络超时
- **WHEN** AI API 调用超过 60s
- **THEN** 返回超时错误

**支持的 operation 值及对应 prompt**：
- `polish`：润色文本，改进语法、表达和风格
- `continue`：根据选中文本的内容和风格续写
- `expand`：扩写文本，增加细节和例子
- `condense`：缩写文本，保留关键信息
- `proofread`：校对文本，修正语法和拼写错误
- `rewrite`：改写文本，保持原意改变表达方式
- `translate`：翻译成中文
- `translate-en`：翻译成英文

### Requirement: executeAction 支持异步

`executeAction` 函数 SHALL 改为 `async function`，使用 `await handler(sourceText)` 调用操作 handler。同步 handler（返回普通值）和异步 handler（返回 Promise）均正常工作。

#### Scenario: 同步操作
- **WHEN** 点击格式化/文本转换等现有操作
- **THEN** `handler` 同步返回结果，`await` 立即 resolve，行为与之前完全一致

#### Scenario: AI 异步操作
- **WHEN** 点击润色等 AI 操作
- **THEN** `handler` 返回 Promise，`await` 等待 Promise resolve 后写回结果

#### Scenario: AI 操作失败
- **WHEN** AI 调用失败
- **THEN** 显示错误通知"AI 处理失败: 原因"

### Requirement: AI 写作操作项

系统 SHALL 提供「AI 写作」分组，包含以下操作项，每个操作项 `type: 'ai'`，handler 异步调用后端 `AITextOperation`。

| 操作 | 无选中文本 | 选中文本 |
|------|-----------|---------|
| 润色 | 提示"请先选择要处理的文本" | 替换为润色后的文本 |
| 续写 | 提示"请先选择要处理的文本" | 在选中文本后追加续写内容 |
| 扩写 | 提示"请先选择要处理的文本" | 替换为扩写后的文本 |
| 缩写 | 提示"请先选择要处理的文本" | 替换为缩写后的文本 |
| 校对 | 提示"请先选择要处理的文本" | 替换为校对后的文本 |
| 改写 | 提示"请先选择要处理的文本" | 替换为改写后的文本 |
| 翻译成中文 | 提示"请先选择要处理的文本" | 替换为中文翻译 |
| 翻译成英文 | 提示"请先选择要处理的文本" | 替换为英文翻译 |

#### Scenario: 无选中文本
- **WHEN** 用户未选中文本时点击 AI 操作
- **THEN** 显示提示通知"请先选择要处理的文本"

#### Scenario: 处理中状态
- **WHEN** AI 调用开始后
- **THEN** 编辑器面板顶部出现 `#aiStatusBar`，包含旋转 spinner、"AI 处理中..."文字和"取消"按钮；`editorActionsBtn` 禁用防止重复点击
- **WHEN** AI 调用完成（成功或失败）
- **THEN** `#aiStatusBar` 移除，`editorActionsBtn` 恢复可用

#### Scenario: 取消操作
- **WHEN** 用户点击 `#aiStatusBar` 中的"取消"按钮
- **THEN** 前端调用 `window.go.main.App.CancelAIStream()`
- **THEN** `#aiStatusBar` 移除，`editorActionsBtn` 恢复可用
- **THEN** 编辑器内容不变（不写回 AI 结果）

## MODIFIED Requirements

### Requirement: executeAction 函数（原有）

`executeAction` 函数签名保持不变，内部改为 `async`，用 `await` 调用 handler。错误处理增加 AI 分支（显示"AI 处理失败"而非"不是合法的 X"）。

## REMOVED Requirements

无