# Check List

## 后端改动验证
- [x] 后端 `CallAIStream` 签名已更新，新增 `referencedNoteIDs`/`roleplayNoteIDs`/`followUpRefContent`/`uploadedFiles` 四个参数
- [x] 后端复用 `AIChatFileResult` 结构体作为上传文件参数类型（等价于 spec 的 `UploadedFileArg`）
- [x] 后端预处理流程按 `1→2→3→4→5→6→7→8` 顺序注入全部上下文
- [x] 角色扮演上下文：`{roleplay_context}` 被正确替换为笔记内容
- [x] 笔记引用上下文：`BuildNoteRefContext` 返回的文本被正确注入 system message
- [x] 追问引用上下文：内容被 `"用户正在追问以下内容：\n"` 前缀包装后注入，且不超过 500 字符
- [x] 上传文件上下文：文件内容按 `--- 文件: name (size) ---\ncontent\n---` 格式包装后注入
- [x] 所有注入使用与搜索/卡片召回一致的 system message 合并规则（追加或新建）

## 前端改动验证
- [x] `onSend()` 删除了 systemContext 拼接代码（笔记引用/角色扮演/追问/文件）
- [x] `startStreaming()` 删除了 `systemContext` 参数
- [x] `startStreaming()` 不再提前拼接 system message，直接使用 `chatHistory` 作为 `messages`
- [x] `startStreaming()` 正确地将前端元数据传给 `CallAIStream`
- [x] `handleRegenerate`/`handleResend`/`handleDeleteMsg` 不再传递 `systemContext`

## 功能回归验证
- [x] 构建运行无错误（`wails build` 成功）
- [x] 功能测试：笔记引用正常
- [x] 功能测试：角色扮演正常
- [x] 功能测试：追问引用正常
- [x] 功能测试：上传文件正常
- [x] 功能测试：联网搜索 + 卡片召回 + 技能（已有逻辑不受影响）
- [x] 功能测试：再生/重发/删除消息功能正常
