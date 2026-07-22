# Tasks

- [x] Task 1: 定义三层结构的基础 prompt 常量
  - 在 `app.go` 中定义包级常量 `baseSystemPrompt`，包含身份层/规范层/边界层
  - 删除 `CallAIStream` 和 `CallAIStreamRegenerate` 中两处重复的硬编码字符串
  - 两处统一引用 `baseSystemPrompt` 常量
  - 代码逻辑不变（无技能时注入，有技能时跳过）

# Task Dependencies

无依赖