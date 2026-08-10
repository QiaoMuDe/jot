# Checklist

- [x] `agent-demo/` 目录存在，含独立 go.mod，`go build ./...` 编译通过
- [x] base URL / API key / 模型 ID 三要素可通过命令行 flag 或环境变量配置，flag 优先
- [x] 启动时打印脱敏后的配置摘要，确认生效配置
- [x] Agent 使用 `adk.NewChatModelAgent` + ReAct 循环，模型能自主发起工具调用
- [x] 至少一个自定义工具（`tool.BaseTool`）可被模型调用并返回结果
- [x] 终端展示工具调用名称/参数、工具结果、最终回答流式文本
- [x] 无工具需求的普通问题直接流式回答，不触发工具调用
- [x] `agent-demo/README.md` 说明配置方式与运行示例
- [x] 主项目（`app.go`、`internal/aicli` 等）无任何改动
