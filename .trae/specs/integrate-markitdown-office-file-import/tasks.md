# Tasks

- [x] Task 1: go.mod 添加 markitdown 依赖
  - 在 `go.mod` 的 `require` 块中添加 `github.com/conductor-oss/markitdown v0.0.1`
  - 在 `go.mod` 末尾添加 `replace` 指令指向 `./tmp/markitdown-0.0.1`
  - 执行 `go mod tidy` 验证依赖解析成功

- [x] Task 2: 创建 `internal/converter/converter.go` 封装层
  - 定义 `ErrUnsupportedFormat` 和 `ErrConversionTimeout` 错误变量
  - 实现 `IsPlainText(path string) bool` — 检查 PlainTextConverter 是否接受
  - 实现 `IsSupported(path string) bool` — 检查是否有任何转换器能处理
  - 实现 `ConvertToMarkdown(path string) (string, error)` — 转换文件为 Markdown，带 60 秒超时
  - 所有函数接受文件路径参数，内部构建 `StreamInfo` 并驱动 markitdown 引擎
  - 添加函数级注释

- [x] Task 3: 改造 `ImportFiles` — 并发 + markitdown 三段式判定
  - 批处理改为 `sync.WaitGroup` + goroutine 并发
  - 保留原始校验（路径、目录、大小限制）在 goroutine 内
  - 文件类型判定改为三段式：
    - markitdown 纯文本 → `os.ReadFile` 直接读取
    - markitdown 办公文件 → `ConvertToMarkdown` 转换
    - 都不支持 → 拒绝
  - goroutine 内使用 `sync.Mutex` 保护 `results` 写入
  - 添加函数级注释

- [x] Task 4: 改造 `readAIChatFiles` — 同上逻辑
  - 并发处理 + markitdown 三段式判定
  - 办公文件转换后的 Markdown 作为 `Content` 返回
  - 添加函数级注释

- [x] Task 5: 编译验证 + 清理
  - `go mod tidy` 确保依赖完整
  - `go build` 编译通过
  - 确认 `ImportFiles` 和 `readAIChatFiles` 中不再调用 `fs.IsBinaryPath`（`ReadTextFile` 保持不变）

# Task Dependencies

- Task 2 依赖 Task 1
- Task 3/4 依赖 Task 2
- Task 5 依赖 Task 3/4
