# Tasks

- [ ] Task 1: 修改后端 `CallAIStream` 函数签名和预处理逻辑
  - [ ] 1.1 在 `app.go` 中新增 `UploadedFileArg` 结构体
  - [ ] 1.2 修改 `CallAIStream` 函数签名，增加 `referencedNoteIDs []uint`、`roleplayNoteIDs []uint`、`followUpRefContent string`、`uploadedFiles []UploadedFileArg` 参数
  - [ ] 1.3 在 `CallAIStream` goroutine 的预处理阶段，在步骤 1（基础身份提示）之后、步骤 6（联网搜索）之前插入步骤 2-5 的注入逻辑：
    - 步骤 2-角色扮演：skill_roleplay + roleplayNoteIDs → 查 `BuildNoteRefContext` → 注入 system + 替换 `{roleplay_context}`
    - 步骤 3-笔记引用：referencedNoteIDs → 查 `BuildNoteRefContext` → 注入 system
    - 步骤 4-追问引用：followUpRefContent → 500 字符截断 → 前缀包装 → 注入 system
    - 步骤 5-上传文件：uploadedFiles → 文件名/大小/内容格式化 → 前言包装 → 注入 system
  - [ ] 1.4 确保 `BuildNoteRefContext` 可在 `CallAIStream` 中调用（`a.noteService` 已有该服务）

- [ ] Task 2: 修改前端 `onSend()` 函数，删除 systemContext 拼接
  - [ ] 2.1 删除 `getNoteContext()` 调用和 systemContext 赋值（第 2004-2007 行）
  - [ ] 2.2 删除角色扮演 `getRoleplayContext()` 调用和拼接（第 2009-2017 行）
  - [ ] 2.3 删除追问引用拼接（第 2019-2025 行）
  - [ ] 2.4 删除上传文件拼接（第 2027-2039 行）
  - [ ] 2.5 将 `startStreaming(false, systemContext)` 改为 `startStreaming(false)`

- [ ] Task 3: 修改前端 `startStreaming()` 函数签名和调用方式
  - [ ] 3.1 删除 `systemContext` 参数（函数签名第 2051 行的 `systemContext = ''`）
  - [ ] 3.2 删除函数体内 systemContent 拼接逻辑（第 2449-2454 行的 messages 构建）
  - [ ] 3.3 直接使用 `chatHistory` 作为 `messages`（不含 system）
  - [ ] 3.4 在 `CallAIStream` 调用中增加 `referencedNoteIDs`、`roleplayNoteIDs`、`followUpRef`、`uploadedFiles` 四个参数

- [ ] Task 4: 修改前端再生/重发/删除消息函数
  - [ ] 4.1 `handleRegenerate`: `startStreaming(true, '')` → `startStreaming(true)`
  - [ ] 4.2 `handleResend`: `startStreaming(false, '')` → `startStreaming(false)`
  - [ ] 4.3 `handleDeleteMsg`: 确认无 `systemContext` 残留引用

## Task Dependencies

- [Task 1] 先完成，[Task 2/3/4] 的代码才能编译验证
- [Task 2] 依赖 [Task 3] 的 `startStreaming` 签名变更（因为 `onSend` 调用 `startStreaming`），建议按 Task 3 → Task 2 顺序实施
- [Task 4] 独立，可并行实施

## 验证方式

1. 运行 `wails dev` 启动应用
2. 逐一测试：
   - 选中笔记 → 发送消息 → 确认 system 中包含笔记内容（可通过日志或调试）
   - 激活角色扮演 + 选人物设定笔记 → 发送 → 确认角色设定生效
   - 右键追问 → 发送 → 确认追问引用生效
   - 上传文件 → 发送 → 确认文件内容随请求发出
   - 联网搜索 + 卡片召回 + 更多技能 → 确认原有功能不受影响
   - 再生/重发/删除消息 → 确认正常工作
