# Tasks

- [x] Task 1: 新增 `reasoningFramework` 常量，重组 System Prompt
  - 在 `app.go` 中新增 `reasoningFramework` 包级常量（4 步内部推理框架）
  - 在 `baseNormsBoundaries` 末尾拼接 `reasoningFramework`
  - 确保 `baseSystemPrompt` 的拼接方式不变（`baseIdentity + "\n\n" + baseNormsBoundaries`）

- [x] Task 2: 搜索结果 `FormattedText` 加来源标签
  - `internal/services/search_service.go`：`SearchWeb()` 的 `FormattedText` 开头加「以下是通过 Tavily 搜索到的相关内容（来源：Tavily 联网搜索）：」
  - `internal/services/zhihu_search_service.go`：`SearchZhihuContent()` 加知乎站内搜索标签
  - `internal/services/zhihu_search_service.go`：`SearchGlobalContent()` 加知乎全网搜索标签

- [x] Task 3: 卡片召回 `FormattedText` 加来源标签
  - `internal/services/recall_service.go`：`CardRecallSearch()` 中 `FormattedText` 开头改为「以下是用户笔记库中与问题相关的笔记（来源：本地笔记，优先级最高）」

- [x] Task 4: 引用笔记/角色扮演/上传文件注入处加来源标签
  - `app.go` 步骤 2（角色扮演）：标签改为「来源：角色设定笔记」
  - `app.go` 步骤 3（手动引用）：注入前加「来源：手动引用笔记」
  - `app.go` 步骤 5（上传文件）：标签改为「来源：上传文件」
  - `CallAIStreamRegenerate` 中对应位置同步修改

# Task Dependencies

- Task 1 无依赖，可优先执行
- Task 2、Task 3、Task 4 无相互依赖，可与 Task 1 并行
