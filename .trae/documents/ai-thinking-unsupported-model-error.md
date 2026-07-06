# 计划：处理不支持深度思考的模型报错转为人类可读提示

## Summary

在 `.trae/specs/thinking-unsupported-model-error/` 目录下创建 Spec 规范文档，然后仅修改 `internal/aicli/errors.go` 一个文件，新增错误分类，使模型因不支持 `enable_thinking` 参数返回的 400 错误被识别并展示为清晰的中文提示。

## 当前状态分析

**问题链路**：

1. 用户开启"深度思考"开关 → `enableThinking = true`
2. 后端 `openai.go:41-43` 设置 `ChatTemplateKwargs = {"enable_thinking": true}`
3. 不支持该参数的模型（如部分第三方兼容模型）返回 HTTP 400
4. `errors.go` 的 `classifyBadRequest()` 仅检测 `content_filter` / `context_length` 两种模式
5. 不匹配时回退到 `CategoryInvalidRequest` → 用户看到"请求参数有误，请检查输入内容"

**根因**：`errors.go` 缺少对"模型不支持 thinking/reasoning"相关错误的检测分支。

## 修改方案

### 涉及文件

仅修改 **[errors.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/aicli/errors.go)**

### 具体变更

#### Step 1: 新增错误分类常量

在 `errors.go` 的常量区域（第 14-26 行）新增：

```go
CategoryModelNotSupportThinking = "model_not_support_thinking"
```

#### Step 2: 新增用户提示

在 `userMessages` 映射（第 36-48 行）新增：

```go
CategoryModelNotSupportThinking: "当前模型不支持"深度思考"功能，请在输入框上方关闭深度思考开关后重试",
```

#### Step 3: 新增检测逻辑

在 `classifyBadRequest()` 函数（第 167-179 行）的 `switch` 中追加一个 case：

```go
case strings.Contains(lower, "enable_thinking") ||
    (strings.Contains(lower, "reasoning") && strings.Contains(lower, "not supported")):
    return NewAIError(CategoryModelNotSupportThinking, rawMsg(msg))
```

### 不修改的地方

* 前端 `ai-chat.js`：无需改动，现有的 `showNotification(errData.user_msg)` 自动展示新消息

* 后端 `app.go` / `ai_service.go`：无需改动，错误已经通过 `ClassifyError` → `ToJSON` → `ai:stream-error` 事件传递

* 其他错误分类逻辑保持不变

## 验证步骤

1. 启动应用，选择一个不支持深度思考的模型
2. 打开深度思考开关
3. 发送任意消息
4. 预期：弹出通知"当前模型不支持"深度思考"功能，请在输入框上方关闭深度思考开关后重试"
5. 关闭深度思考后重试，应正常输出

