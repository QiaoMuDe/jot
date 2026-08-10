# web_search 工具：模型自选来源 + 按 URL 去重

## Summary

优化 [web_search.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/web_search.go) 工具：

1. **模型自选来源**：新增可选 `sources` 数组参数（枚举 `tavily` / `zhihu_search` / `zhihu_global`），模型按需指定要搜索的来源（单源或多个）；不传时回退全部来源（向后兼容，行为与现在一致）。
2. **按 URL 去重**：合并多个来源结果时，按 URL 去重后统一重建格式化文本，避免同一网页在多个来源中重复出现浪费 token，前端 `Collector.Sources` 同样只保留唯一 URL。

采用**单工具 + sources 参数**方案（用户已确认），改动集中在 `web_search.go`，不动 registry / agent.go / context.go / 前端。

## Current State Analysis

- [web_search.go#L64-L190](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/web_search.go#L64-L190)：`InvokableRun` 固定并发执行全部 3 个来源（`defaultSearchSources`），合并时**无任何去重**——`b.WriteString(r.result.FormattedText)` 直接拼接，同一 URL 可能同时出现在 Tavily 与知乎全网结果中。
- 每个来源返回 [SearchWebResult](file:///d:/峡谷/Dev/本地项目/jot/internal/services/search_service.go#L21-L25)：`FormattedText`（services 层预生成的整段文本）+ `Sources`（结构化条目，含 `Title/URL/Content/SourceLabel`，Content 已按 maxChars 截断）。`Sources` 字段足以重建等价格式化文本。
- eino schema 的 `ParameterInfo` 支持数组（`ElemInfo`）与枚举（`Enum`）参数（[tool.go#L249-L264](file:///D:/xiazai/gopath/pkg/mod/github.com/cloudwego/eino@v0.9.13/schema/tool.go#L249-L264)），可实现 `sources` 数组参数，模型只能传合法枚举值。
- 来源标识常量与 `SourceLabel` 值一致：`tavily` / `zhihu_search` / `zhihu_global`（[web_search.go#L23-L30](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/web_search.go#L23-L30)、services 三个搜索函数）。
- 前端工具状态展示是通用事件驱动（`tool_start`/`tool_result` 带工具名），来源展示基于 `SearchSource` 数据——**前端无需改动**。
- [app.go#L2157-L2230](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2157-L2230)（问答模式）的多源搜索逻辑本次**不动**（避免连带回归），仅 web_search 工具受益。

## Proposed Changes

### 文件：`internal/agent/tools/web_search.go`（唯一改动文件）

#### 1. `Info()`：新增 `sources` 数组参数并更新工具描述

- 参数表新增：
  - `sources`：`Type: schema.Array`，`ElemInfo: {Type: schema.String, Enum: []string{"tavily", "zhihu_search", "zhihu_global"}}`，`Required: false`，`Desc: "要搜索的来源列表。tavily=Tavily 通用互联网搜索；zhihu_search=知乎站内内容；zhihu_global=知乎全网搜索。按用户问题性质选择一个或多个来源；不传或传空则搜索全部可用来源。"`
- `Desc` 更新：说明模型应根据问题选择来源（如知乎相关话题优先 `zhihu_search`，一般时事/新闻用 `tavily`，需跨站全网信息可多个），并保留"口语化 query 先调 refine_search_query"的指引。

#### 2. `InvokableRun()`：解析 sources、按所选源执行

- 参数结构体新增 `Sources []string \`json:"sources"\``。
- 解析后过滤：剔除非法枚举值；去重（模型可能传重复）；**为空则回退 `defaultSearchSources`**。
- 保留现有"未配置密钥的源 → `failedParts` 标记跳过"的探测逻辑（只对所选源探测）。

#### 3. 合并结果：按 URL 去重 + 统一重建格式化文本

- 收集所有成功源的结果条目到一个切片（`[]services.SearchSource`），用 `seen map[string]struct{}` 按 URL 去重，保留**首次出现顺序**；URL 为空或空白也跳过。
- 新增私有函数 `formatDedupResults(query string, sources []services.SearchSource) string`，基于去重后的 `Sources` **按来源分组**重建文本（分组顺序固定为 tavily → zhihu_search → zhihu_global，条目序号全局连续）：

```
以下是为“{query}”搜索到的相关内容（已按 URL 去重，共 {N} 条）：

【Tavily 联网搜索】
[{i}] {Title}
    来源: {URL}
    内容: {Content}

【知乎站内搜索】
...
```

  - `SourceLabel` → 中文名映射：`tavily`→Tavily 联网搜索、`zhihu_search`→知乎站内搜索、`zhihu_global`→知乎全网搜索。
  - 若去重后为空（全部来源无有效条目），返回错误（沿用现有 `b.Len()==0` 分支的文案）。
- `Collector.Sources` 追加去重后的 `Sources`（与格式化文本一致，前端展示不重复）。
- **部分失败机制不变**：`failedParts` 登记、`AddPartial`、成功内容在前 + 失败说明后缀的现有逻辑原样保留。

#### 4. 其余保持不变

- 并发 goroutine + channel 收集模式、每源 5s 超时（services 内建）、`ctx.Err()` 取消检查、`intSetting` 读取设置——全部不动。
- 构造器签名 `NewWebSearch(ai, setting, ctx)` 不变，`registry.go` / `agent.go` 无需修改。

## Assumptions & Decisions

1. **不传 sources 回退全部来源**：保证模型漏传或传空时行为与现状一致，不退化。
2. **URL 完全匹配去重**：不做 host 级模糊去重（避免误杀不同页面的同站链接）；URL 用 `strings.TrimSpace` 后比较。
3. **重建格式化文本而非拼接原 `FormattedText`**：原 `FormattedText` 是整段预生成文本，无法精确定位并剔除重复 URL 对应条目；`Sources` 已含全部展示字段，重建即可在模型侧与前端侧同时去重。app.go 问答模式仍用 services 原样输出，不受影响。
4. **不新增去重后的总条数/总长度上限**：单条长度已由 `maxChars` 设置控制（默认 5000），本次不扩大改动面。
5. **不做配置读取缓存**（上轮分析的 DB 查询优化）：与本次需求无关，保持范围最小。

## Verification

1. `go build ./...` 通过。
2. `go vet ./internal/agent/... ./internal/services/...` 通过。
3. 手动验证（Agent 模式）：
   - 模型只传 `sources=["zhihu_search"]` 时，仅知乎站内搜索被执行（日志 `active_sources` 只有一项）。
   - Tavily 与知乎全网返回同一 URL 时，最终文本与前端来源列表均无重复 URL 条目，且文本按【Tavily 联网搜索】【知乎站内搜索】【知乎全网搜索】分组展示。
   - 模型不传 `sources` 时仍执行全部来源（回归验证）。
