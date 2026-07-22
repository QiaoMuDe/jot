# 联网搜索结果内容按引用截断数截断

## 概述

对三个联网搜索来源的每条记录，按 `ai_ref_max_chars`（引用截断字数）配置截断其 `Content`，避免过长内容消耗过多 token。

## 当前状态分析

### 三个搜索函数均无截断

| 搜索来源   | 函数                    | 文件                                                                                                |
| ------ | --------------------- | ------------------------------------------------------------------------------------------------- |
| Tavily | `SearchWeb`           | [search\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/search_service.go)              |
| 知乎搜索   | `SearchZhihuContent`  | [zhihu\_search\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/zhihu_search_service.go) |
| 知乎全网   | `SearchGlobalContent` | [zhihu\_search\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/zhihu_search_service.go) |

三个函数的处理逻辑完全一致：遍历 API 返回结果 → 将 `Content` 原样写入 `FormattedText` 和 `Sources`，**没有任何截断逻辑**。

### 已有截断配置

`ai_ref_max_chars`（`AIRefMaxChars`，引用截断字数）已存在，默认值 **10000**，范围 1\~100000，已用于：

* 笔记引用截断（`BuildNoteRefContext`）

* 卡片召回截断（`CardRecallSearch`，接收 `maxChars` 参数）

* 文件上传截断

* 大文件 Markdown 预览自动切换

### 调用方

`app.go` 中有两处调用搜索（并行 goroutine），一处测试连接调用：

* `CallAIStream` 第一次对话（约第 1755-1771 行）

* `CallAIStream` 再生（约第 2166-2182 行）

* `TestTavilyConnection`（第 1452 行，测试连接不需要截断）

## 修改方案

### 改动文件

共 **3 个文件**：

#### 1. `internal/services/search_service.go`

**`SearchWeb`** **函数**：

* 签名新增 `maxChars int` 参数

* 遍历 `answer.Results` 时，对 `r.Content` 截断：

  * 用 `[]rune` 处理（支持中文）

  * 如果 `len([]rune(content)) > maxChars && maxChars > 0`，截取前 `maxChars` 字符，追加 `"\n\n...(内容已截断)"`

  * 同时更新 `SearchSource.Content` 为截断后的值

#### 2. `internal/services/zhihu_search_service.go`

**`SearchZhihuContent`** **和** **`SearchGlobalContent`** **函数**：

* 签名各新增 `maxChars int` 参数

* 遍历 `items` 时，对 `ContentText` 做相同的截断逻辑

* 同时更新 `SearchSource.Content` 为截断后的值

#### 3. `app.go`

**两处搜索调用（并行 goroutine）**：

* 在 goroutine 启动前读取 `maxChars := a.GetAIRefMaxChars()`

* 传入三个搜索函数：`SearchWeb(ctx, query, apiKey, maxResults, maxChars)`

* 同样传入 `SearchZhihuContent` 和 `SearchGlobalContent`

**测试连接调用**（第 1452 行）：

* 传 `0` 表示不截断：`services.SearchWeb(ctx, "test", apiKey, 1, 0)`

### 截断逻辑（三处一致）

```go
// 用 rune 处理以支持中文
content := strings.TrimSpace(r.Content)
if maxChars > 0 && len([]rune(content)) > maxChars {
    runes := []rune(content)
    content = string(runes[:maxChars]) + "\n\n...(内容已截断)"
}
```

### 不需要修改的文件

* **前端模型文件**：`frontend/wailsjs/go/models.ts` — `SearchSource.Content` 字段已存在，值变为截断后的内容即可

* **前端 JS**：`frontend/src/main.js` — 前端仅展示 `Sources` 数据，不涉及截断逻辑

* **设置 UI**：`ai_ref_max_chars` 的输入框和保存逻辑已存在

## 假设与决策

* 截断策略选择**从头截取**（与文件上传截断一致），而非从关键词附近截取（因为搜索结果是外部网页内容，没有关键词上下文的概念）

* 使用 `[]rune` 处理中文（与 `recall_service.go` 中 `extractContext` 的做法一致），避免按字节截断破坏中文字符

* `maxChars <= 0` 表示不截断（与 `CardRecallSearch` 的约定一致）

* 测试连接传 `0`，因为测试只需要确认 API 连通性，不需要内容

## 验证步骤

1. 确认编译通过：`go build ./...`
2. 确认 `TestTavilyConnection` 仍正常工作
3. 确认联网搜索结果中，超过 `ai_ref_max_chars` 的内容被正确截断，末尾显示 `...(内容已截断)`
4. 确认中文内容截断没有乱码

