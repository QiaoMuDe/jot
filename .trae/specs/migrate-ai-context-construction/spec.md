# AI 对话上下文拼接全量迁移至后端 Spec

## Why

当前 AI 对话的 6 项上下文注入逻辑分散在前后端：

| 上下文类型 | 当前处理位置 | 状态 |
|-----------|-------------|------|
| 更多技能提示词 | 后端 `CallAIStream` | ✅ 已迁移（见 `move-skill-prompts-to-backend` spec） |
| 卡片召回 | 后端 `CallAIStream` | ✅ 已迁移 |
| 联网搜索 | 后端 `CallAIStream` | ✅ 已迁移 |
| **笔记引用** | **前端 `onSend()`** | ❌ 待迁移 |
| **角色扮演** | **前端 `onSend()`** | ❌ 待迁移 |
| **追问引用** | **前端 `onSend()`** | ❌ 待迁移 |
| **上传文件** | **前端 `onSend()`** | ❌ 待迁移 |

这种分散状态导致：
- **职责不统一**：同样做"上下文拼入 system message"这件事，4 项在前端用 JS 拼，3 项在后端用 Go 拼
- **冗余数据传输**：笔记内容（`GetNoteRefContext`）和文件内容（`ReadAIChatFiles`）已由后端提供，前端拿到后仅做简单字符串包装又传回后端，多一次无意义的来回
- **维护成本高**：理解完整提示词构建流程需要跨 `ai-chat.js` 和 `app.go` 两个文件
- **扩展性差**：新增上下文类型需要改前后端两处

本 spec 的目标：**将全部上下文拼接集中在后端 `CallAIStream` 一处完成**，前端只负责：
1. 收集用户输入的元数据（选中的笔记 ID、角色扮演笔记 ID、追问文本、上传文件内容等）
2. 将元数据传给后端
3. 展示后端返回的结果

## 迁移后全量注入流程

迁移完成后，`CallAIStream` 内部完整的消息预处理流程如下：

```
CallAIStream 接收 messages(仅 user/assistant 历史) + 元数据参数
  │
  ├─ 1. 基础身份提示（无技能且无 system 时注入）
  ├─ 2. 角色扮演上下文 ← 本次从前端迁入
  ├─ 3. 笔记引用上下文 ← 本次从前端迁入
  ├─ 4. 追问引用内容   ← 本次从前端迁入
  ├─ 5. 上传文件内容   ← 本次从前端迁入
  ├─ 6. 联网搜索结果   ← 已有
  ├─ 7. 卡片召回结果   ← 已有
  └─ 8. 技能提示词     ← 已有
       │
       ▼ 最终 messages（含完整 system）→ AI API
```

## What Changes

- **修改** `app.go` `CallAIStream` 函数签名，增加 `referencedNoteIDs []uint`、`roleplayNoteIDs []uint`、`followUpRefContent string`、`uploadedFiles []UploadedFileArg` 四个参数
- **修改** `app.go` `CallAIStream` 内部预处理流程，在步骤 1（基础身份提示）之后、步骤 6（联网搜索）之前插入步骤 2-5 的注入逻辑
- **修改** `frontend/src/js/ai-chat.js` `onSend()` 函数：删除 systemContext 拼接代码，改为将元数据传给 `startStreaming`
- **修改** `frontend/src/js/ai-chat.js` `startStreaming()` 函数：删除 `systemContent` 参数，改为将 4 项元数据透传给 `CallAIStream`
- **修改** 前端再生/重发/删除消息函数：调用 `startStreaming` 时不再传递 `systemContext`
- **保留不变**：前端 `referencedNotes`/`roleplayNotes`/`followUpRef`/`uploadedFiles` 状态变量仍用于 UI 展示（chip 渲染），只是不再参与 prompt 拼接

## Impact

- Affected code:
  - `app.go` — `CallAIStream` 签名 + 预处理流程（约 +30 行）
  - `frontend/src/js/ai-chat.js` — `onSend()`、`startStreaming()`、`handleRegenerate()`、`handleResend()`、`handleDeleteMsg()`（约 -40 行）
- 无需新增文件，纯代码迁移

## ADDED Requirements

### Requirement: 后端统一的 system prompt 注入

系统 SHALL 在 `CallAIStream` 中按固定顺序注入所有上下文到 system message。

#### Scenario: 消息预处理顺序
- **WHEN** `CallAIStream` 接收到消息数组和新参数
- **THEN** 按以下顺序构建 system prompt：
  1. 基础身份提示词（已有逻辑，保持不变）
  2. 角色扮演上下文（本次新增：见下方细化需求）
  3. 笔记引用上下文（本次新增：见下方细化需求）
  4. 追问引用内容（本次新增：见下方细化需求）
  5. 上传文件内容（本次新增：见下方细化需求）
  6. 联网搜索结果（已有逻辑，保持不变）
  7. 卡片召回结果（已有逻辑，保持不变）
  8. 技能提示词（已有逻辑，保持不变）

#### Scenario: system 消息合并规则
- **WHEN** 上述任意步骤需要注入文本
- **THEN** 查找 messages 数组中 role="system" 的消息：
  - **IF** 找到 → 追加内容到末尾（`\n\n` 分隔）
  - **ELSE** → 新建 `{role:"system", content:...}` 插入到 messages 开头
- **THEN** 继续下一项注入

### Requirement: 后端角色扮演上下文注入（本次迁移）

系统 SHALL 在启用了 roleplay 技能且传了角色扮演笔记 ID 时，将笔记内容替换技能模板中的 `{roleplay_context}` 占位符。

#### 迁移前（前端处理）
```javascript
// 前端 ai-chat.js onSend()
roleplayCtx = await getRoleplayContext()  // 调 App.GetNoteRefContext(ids)
// 前端拿到 skillPrompt（含 {roleplay_context}）后，前端用 roleplayCtx 替换
systemContext += roleplayCtx
```

#### 迁移后（后端处理）
```go
// app.go CallAIStream 中
if skillRoles包含"roleplay" && len(roleplayNoteIDs) > 0 {
    refCtx, _ := a.noteService.BuildNoteRefContext(roleplayNoteIDs)
    // skillPrompt 已通过 GetSkillPrompts 获取，含 {roleplay_context} 占位符
    skillPrompt = strings.ReplaceAll(skillPrompt, "{roleplay_context}", refCtx.Context)
    // skillPrompt 随后注入 system message（步骤 8 技能提示词阶段）
}
```

#### Scenario: 角色扮演注入
- **WHEN** `skillIds` 包含 `skill_roleplay` 且 `roleplayNoteIDs` 非空
- **THEN** 后端调用 `noteService.BuildNoteRefContext(roleplayNoteIDs)` 获取笔记内容上下文
- **THEN** 在步骤 2（角色扮演上下文注入）构建文本 `"以下是用户提供的人物设定笔记内容：\n\n" + refCtx.Context`
- **THEN** 将该文本注入 system message
- **AND** 在步骤 8（技能提示词注入）中，将 `skillRoleplay` 的 `{roleplay_context}` 占位符替换为 `refCtx.Context`
- **NOTE**: 角色扮演有两处注入——步骤 2 注入原始笔记上下文作为背景，步骤 8 在技能模板中替换占位符以完成角色设定指令。这与迁移前前端行为一致。

### Requirement: 后端笔记引用上下文注入（本次迁移）

系统 SHALL 在传了引用笔记 ID 时，将笔记内容注入 system message。

#### 迁移前（前端处理）
```javascript
// 前端 ai-chat.js onSend()
systemContext = await getNoteContext()  // 调 App.GetNoteRefContext(ids)
```

#### 迁移后（后端处理）
```go
// app.go CallAIStream 中
if len(referencedNoteIDs) > 0 {
    refCtx, _ := a.noteService.BuildNoteRefContext(referencedNoteIDs)
    // 将 refCtx.Context 注入 system message
}
```

#### Scenario: 笔记引用注入
- **WHEN** `referencedNoteIDs` 非空
- **THEN** 后端调用 `noteService.BuildNoteRefContext(referencedNoteIDs)` 获取上下文文本
- **THEN** 将上下文文本追加到 system message

### Requirement: 后端追问引用注入（本次迁移）

系统 SHALL 在传了追问引用内容时，包装后注入 system message。

#### 迁移前（前端处理）
```javascript
// 前端 ai-chat.js onSend()
refText = '用户正在追问以下内容：\n' + followUpRef.slice(0, 500)
systemContext += refText
```

#### 迁移后（后端处理）
```go
// app.go CallAIStream 中
if followUpRefContent != "" {
    refText := "用户正在追问以下内容：\n" + followUpRefContent
    if len([]rune(followUpRefContent)) > 500 {
        refText = "用户正在追问以下内容：\n" + string([]rune(followUpRefContent)[:500])
    }
    // 注入 system message
}
```

#### Scenario: 追问引用注入
- **WHEN** `followUpRefContent` 非空
- **THEN** 后端截断至 500 字符，包装为 `"用户正在追问以下内容：\n" + 截断后内容`
- **THEN** 将包装后的文本追加到 system message

### Requirement: 后端上传文件注入（本次迁移）

系统 SHALL 在传了上传文件列表时，包装文件内容后注入 system message。

#### 迁移前（前端处理）
```javascript
// 前端 ai-chat.js onSend()
fileContext = '用户上传了以下文件内容，请基于这些内容回答用户的提问：\n'
uploadedFiles.forEach(f => {
  fileContext += '\n--- 文件: ' + f.name + ' (' + formatFileSize(f.size) + ') ---\n'
  fileContext += f.content + '\n---'
})
systemContext += fileContext
```

#### 迁移后（后端处理）
```go
// app.go CallAIStream 中
if len(uploadedFiles) > 0 {
    var b strings.Builder
    b.WriteString("用户上传了以下文件内容，请基于这些内容回答用户的提问：\n")
    for _, f := range uploadedFiles {
        sizeStr := formatFileSize(f.Size) // 后端辅助函数
        b.WriteString(fmt.Sprintf("\n--- 文件: %s (%s) ---\n%s\n---", f.Name, sizeStr, f.Content))
    }
    // 注入 system message
}
```

#### Scenario: 上传文件注入
- **WHEN** `uploadedFiles` 非空
- **THEN** 后端遍历文件列表，为每个文件生成 `"\n--- 文件: {name} ({size}) ---\n{content}\n---"` 格式文本
- **THEN** 以 `"用户上传了以下文件内容，请基于这些内容回答用户的提问：\n"` 为前言，拼接所有文件块
- **THEN** 将拼接后的文本追加到 system message

## MODIFIED Requirements

### Requirement: CallAIStream 函数签名

修改 `CallAIStream` 的 Go 绑定函数签名，增加四个参数承载前端元数据。

#### 旧签名
```go
func (a *App) CallAIStream(streamGen int, messages []services.Message, thinkingEnabled bool,
    searchSources []string, cardRecallEnabled bool, sessionID uint, isRegenerate bool, skillIds []string)
```

#### 新签名
```go
type UploadedFileArg struct {
    Name      string `json:"name"`
    Content   string `json:"content"`
    Size      int64  `json:"size"`
    Truncated bool   `json:"truncated"`
    Path      string `json:"path"`
}

func (a *App) CallAIStream(streamGen int, messages []services.Message, thinkingEnabled bool,
    searchSources []string, cardRecallEnabled bool, sessionID uint, isRegenerate bool,
    skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint,
    followUpRefContent string, uploadedFiles []UploadedFileArg)
```

### Requirement: 前端发送消息流程

前端 `onSend()` 不再拼接 `systemContext`，改为将元数据传给 `startStreaming`。`startStreaming()` 不再接受 `systemContent` 参数。

#### 旧流程
```
onSend():
  getNoteContext()     → 若引用笔记 >0，拼接
  getRoleplayContext() → 若角色扮演激活，拼接
  followUpRef 拼接     → 若有追问引用，拼接
  uploadedFiles 拼接   → 若有上传文件，拼接
  startStreaming(false, systemContext)

startStreaming(isRegenerate, systemContext):
  messages = [{role:system, content:systemContext}, ...chatHistory]
  CallAIStream(myGen, messages, ...)   ← 传给后端的是一个已经拼好的 system message
```

#### 新流程
```
onSend():
  // 只传元数据，不拼接
  startStreaming(false)

startStreaming(isRegenerate):
  messages = chatHistory  // 只有 user/assistant 消息，不含 system
  // 将元数据作为独立参数传给后端，由后端统一拼接
  CallAIStream(myGen, messages, ..., referencedNoteIDs, roleplayNoteIDs, followUpRef, uploadedFiles)
```

### Requirement: 前端再生/重发/删除消息

`handleRegenerate`、`handleResend`、`handleDeleteMsg` 调用 `startStreaming` 时不再传 `systemContext`。

#### 变更
- `handleRegenerate`: `startStreaming(true)`（原为 `startStreaming(true, '')`）
- `handleResend`: `startStreaming(false)`（原为 `startStreaming(false, '')`）

## REMOVED Requirements

### Requirement: 前端 systemContext 拼接逻辑

**Reason**: 拼接逻辑迁移至后端，前端不再需要。
**Migration**: 删除 `onSend()` 中第 2003-2039 行的 systemContext 构建代码；保留 `getNoteContext()` 和 `getRoleplayContext()` 的缓存逻辑仅用于 UI（如更新 chip 展示），但不再用于发送消息。

### Requirement: startStreaming 的 systemContent 参数

**Reason**: system 消息的构建完全移至后端。
**Migration**: 删除 `startStreaming` 函数的 `systemContext` 参数定义，函数体内对应的 `messages = [{role:system, ...}, ...chatHistory]` 逻辑一并删除。

## 迁移前后全景对比

### 迁移前
```
                    前端                                   后端
                ┌─────────────────┐              ┌──────────────────────┐
                │ onSend()        │              │ CallAIStream         │
                │  拼接 systemCtx │  messages    │  追加搜索结果        │
                │  ├─ 笔记引用     │ ──────────→  │  追加卡片召回        │
                │  ├─ 角色扮演     │ (已含system) │  追加技能提示词      │
                │  ├─ 追问引用     │              │                      │
                │  └─ 上传文件     │              │  最终 → AI API       │
                └─────────────────┘              └──────────────────────┘
                     ↑ 4 项在前端                  ↑ 3 项在后端
```

### 迁移后
```
                    前端                                   后端
                ┌─────────────────┐              ┌──────────────────────────┐
                │ onSend()        │  元数据       │ CallAIStream             │
                │  传笔记 IDs      │ ──────────→  │  1. 基础身份提示         │
                │  传角色 IDs      │ (不拼system) │  2. 角色扮演 ← 迁入      │
                │  传追问文本      │              │  3. 笔记引用 ← 迁入      │
                │  传文件内容      │              │  4. 追问引用 ← 迁入      │
                │  传搜索/技能开关 │              │  5. 上传文件 ← 迁入      │
                └─────────────────┘              │  6. 搜索结果 已有        │
                                                 │  7. 卡片召回 已有        │
                                                 │  8. 技能提示词 已有      │
                                                 │                          │
                                                 │  最终 → AI API           │
                                                 └──────────────────────────┘
                                                      ↑ 全部 8 步在后端
```
