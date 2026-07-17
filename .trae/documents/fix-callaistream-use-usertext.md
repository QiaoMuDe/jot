# 修复 `CallAIStream` 搜索精炼未使用前端传入的 `userText` 参数

## 当前状态

`CallAIStream` 的签名中接收了前端传过来的 `userText` 参数，但在搜索精炼阶段没有直接使用，而是从数据库加载消息列表再反向遍历找最后一条 user 消息：

```go
// app.go line 1718-1724
var query string
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "user" {
        query = messages[i].Content
        break
    }
}
```

这段代码是当初从 `CallAIStreamRegenerate` 复制粘贴过来的——`Regenerate` 版本没有 `userText` 参数，所以必须反向遍历。`CallAIStream` 已经有了 `userText`，无需再绕弯。

## 改动方案

修改 [app.go](file:///d:\峡谷\Dev\本地项目\jot\app.go) 中 `CallAIStream` 方法的搜索精炼阶段（第1718-1724行），将反向遍历查找替换为直接使用 `userText` 参数。

### 具体变更

**文件**: `app.go`，`CallAIStream` 方法

将：

```go
			var query string
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == "user" {
					query = messages[i].Content
					break
				}
			}
```

替换为：

```go
			query := userText
```

### 影响范围

* **`CallAIStream`** — 仅此一处改动，搜索精炼直接取 `userText`

* **`CallAIStreamRegenerate`** — 保持不变，仍保留反向遍历逻辑（它没有 `userText` 参数）

### 验证方式

1. 项目能正常编译：`go build ./...`
2. 启用联网搜索发送消息，精炼阶段能正常使用用户输入生成搜索关键词，搜索行为不受影响

