# 抽取 `appendToSystemMessage` 辅助函数

## 摘要

在 `app.go` 的 `CallAIStream` 和 `CallAIStreamRegenerate` 两个方法中，"往 system 消息追加内容，没有则新建"的代码块重复出现 14 次。抽取为公共辅助函数，消除重复。

## 当前状态

每个追加模式固定 10 行代码：

```go
found := false
for i := range messages {
    if messages[i].Role == "system" {
        messages[i].Content = messages[i].Content + "\n\n" + content
        found = true
        break
    }
}
if !found {
    messages = append([]services.Message{{Role: "system", Content: content}}, messages...)
}
```

14 处位置（按行号）：

| #  | 行号   | 方法                       | 追加内容                              |
| -- | ---- | ------------------------ | --------------------------------- |
| 1  | 1637 | `CallAIStream`           | 角色扮演 `roleplayText`               |
| 2  | 1655 | `CallAIStream`           | 笔记引用 `refCtx.Context`             |
| 3  | 1675 | `CallAIStream`           | 追问引用 `refText`                    |
| 4  | 1700 | `CallAIStream`           | 上传文件 `b.String()`                 |
| 5  | 1799 | `CallAIStream`           | 搜索结果 `r.result.FormattedText`     |
| 6  | 1891 | `CallAIStream`           | 卡片召回 `recallResult.FormattedText` |
| 7  | 1925 | `CallAIStream`           | 技能提示词 `skillPrompt`               |
| 8  | 2086 | `CallAIStreamRegenerate` | 角色扮演 `roleplayText`               |
| 9  | 2104 | `CallAIStreamRegenerate` | 笔记引用 `refCtx.Context`             |
| 10 | 2124 | `CallAIStreamRegenerate` | 追问引用 `refText`                    |
| 11 | 2149 | `CallAIStreamRegenerate` | 上传文件 `b.String()`                 |
| 12 | 2247 | `CallAIStreamRegenerate` | 搜索结果 `r.result.FormattedText`     |
| 13 | 2331 | `CallAIStreamRegenerate` | 卡片召回 `recallResult.FormattedText` |
| 14 | 2360 | `CallAIStreamRegenerate` | 技能提示词 `skillPrompt`               |

## 变更方案

### Step 1: 在 `app.go` 中添加辅助函数

在文件末尾（或在现有工具函数附近）添加：

```go
// appendToSystemMessage 往消息列表中的 system 角色消息追加内容。
// 如果不存在 system 消息，则在头部插入一条新的 system 消息。
func appendToSystemMessage(msgs []services.Message, content string) []services.Message {
	for i := range msgs {
		if msgs[i].Role == "system" {
			msgs[i].Content = msgs[i].Content + "\n\n" + content
			return msgs
		}
	}
	return append([]services.Message{{Role: "system", Content: content}}, msgs...)
}
```

### Step 2: 替换 14 处调用点

每处将 10 行代码替换为 1 行函数调用，例如：

```go
// 替换前（10行）：
found := false
for i := range messages {
    if messages[i].Role == "system" {
        messages[i].Content = messages[i].Content + "\n\n" + roleplayText
        found = true
        break
    }
}
if !found {
    messages = append([]services.Message{{Role: "system", Content: roleplayText}}, messages...)
}

// 替换后（1行）：
messages = appendToSystemMessage(messages, roleplayText)
```

所有 14 处替换模式完全一致，只是 `content` 变量名不同。

## 假设与决策

* 辅助函数放在 `app.go` 中（与两个调用方同文件），无需跨文件引用

* 函数签名 `msgs []services.Message` 而非指针，返回值覆盖原变量（Go slice 传引用特性）

* 不额外处理 `ctx.Err()` 检查——辅助函数只做追加逻辑，取消检查由调用方负责，职责清晰

## 验证步骤

1. `go build ./...` 编译通过
2. 检查任意一处替换是否正确：确认 `messages =` 赋值存在（接收返回值）

