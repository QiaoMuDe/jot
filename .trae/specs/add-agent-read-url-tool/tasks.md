# Tasks

- [x] Task 1: 添加 eino-ext URL Loader 依赖
  - [x] 执行 `go get github.com/cloudwego/eino-ext/components/document/loader/url` 并 `go mod tidy`
  - [x] 确认 `go.mod` / `go.sum` 新增该依赖
- [x] Task 2: 提取 `intSetting` 为包级函数 `getIntSetting`
  - [x] 在 `internal/agent/tools/web_search.go` 将 `intSetting` 方法改为包级函数 `getIntSetting(setting, key, def, max)`，更新 `InvokableRun` 中两处调用点
- [x] Task 3: 新增 `read_url.go` 工具实现
  - [x] 实现 `readURLTool` 结构体（依赖 `setting`、`ctx`）与编译期断言 `var _ tool.InvokableTool = (*readURLTool)(nil)`
  - [x] 实现 `ActionText(argumentsInJSON)`：解析 `url` 返回"阅读链接 {url}"（截断 30 字符），失败回退"阅读网页链接"
  - [x] 实现 `Info()`：名称 `read_url`，`Desc` 说明何时调用（用户带链接/要求阅读网页/搜索深入），参数 `url` 必填 http/https
  - [x] 实现 `InvokableRun()`：解析参数 → 校验 URL scheme → 构建 `url.NewLoader`（15s 超时 + 浏览器 UA）→ `loader.Load` 提取正文 → 按 `getIntSetting(setting, "ai_web_search_max_chars", 5000, 50000)` 截断 → 返回格式化文本；错误路径含空正文、ctx 取消
  - [x] 实现 `NewReadURL(setting, ctx)` 构造器
- [x] Task 4: 注册工具并同步清单
  - [x] `internal/agent/registry.go` `buildTools` 追加 `{"read_url", tools.WrapWithError("read_url", tools.NewReadURL(p.deps.Setting, p.ctx), p.ctx)}`
  - [x] `internal/agent/tools/meta.go` `BuiltinTools` 追加 `{Name: "read_url", Label: "读取网页链接内容"}`
  - [x] `internal/agent/tools/doc.go` 工具列表与构造器名追加 `read_url` / `NewReadURL`
  - [x] `internal/agent/doc.go` 只读工具列表追加 `read_url`
- [x] Task 5: 构建验证
  - [x] `go build ./...` 通过
  - [x] `go vet ./internal/agent/...` 通过

# Task Dependencies

- [Task 2] 无依赖
- [Task 3] 依赖 [Task 1]（需要 eino-ext url loader 包）
- [Task 4] 依赖 [Task 3]（需先有构造器）
- [Task 5] 依赖 [Task 2][Task 3][Task 4]
