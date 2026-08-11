# Tasks

- [x] Task 1: 新增 get_stats 工具（[internal/agent/tools/get_stats.go](internal/agent/tools/get_stats.go)）
  - [x] 结构体 `getStatsTool`：依赖 `note *services.NoteService`、`vector *services.VectorService`、`ctx *Context`；编译期断言 `var _ tool.InvokableTool = (*getStatsTool)(nil)`
  - [x] `Info()`：Name `get_stats`，Desc 说明只读概览、与 manage_* 的边界（查看具体列表用 manage_*，本工具答总量/概览/月度趋势）；参数 `action`（enum: overview/month，缺省 overview）、`year`（Number，仅 month 用，缺省当前年）、`month`（Number，仅 month 用，1-12，缺省当前月）
  - [x] `InvokableRun()`：解析参数 → 校验（action 非法枚举、month 越界报错）→ 分发：
    - overview（缺省）：调用 `note.GetStats()` 与 `vector.GetIndexStatus()`，格式化友好文本（笔记总数/回收站/置顶/笔记本/标签/待办完成比/AI 会话与消息/总 token/平均响应与思考时长/DB 大小 + 向量索引状态：已量化笔记数/片段总数/占用字节，占用字节按 KB/MB 格式化）
    - month：year/month 缺省用当前年月（time.Now()），调用 `note.GetMonthCounts(year, month)`，输出"YYYY-MM 每日笔记数"（无笔记的日不列出，全部为零时返回"YYYY-MM 暂无笔记"）
  - [x] `ActionText()`：overview（或缺省）→ `获取数据统计概览`；month → `获取月度笔记统计`；解析失败 → ""；其他 → `执行`
  - [x] `NewGetStats(note *services.NoteService, vector *services.VectorService, ctx *Context) tool.InvokableTool` 构造器
  - [x] 验证：`go build ./...` 通过

- [x] Task 2: 注册 get_stats（[internal/agent/registry.go](internal/agent/registry.go)）
  - [x] `buildTools` 追加 `tools.WrapWithError("get_stats", tools.NewGetStats(p.deps.Note, p.deps.Vector, p.ctx), p.ctx)`
  - [x] 验证：`go build ./...`、`go vet ./internal/agent/...` 通过

- [x] Task 3: 更新工具清单文档（[internal/agent/TOOLS.md](internal/agent/TOOLS.md)）
  - [x] §6 现有工具清单表格追加一行：`get_stats` | [get_stats.go](internal/agent/tools/get_stats.go) | `note`、`vector`、`ctx` | 数据概览 / 月度笔记统计（只读） | 无
  - [x] 确认无其他小节需要同步（§2 第 5 步 / §3 / §5 / §8 的动作文案机制已通用，无需改动）

# Task Dependencies

- [Task 2] depends on [Task 1]（构造器存在才能注册）
- [Task 3] depends on [Task 1]-[Task 2]（文档描述最终行为）
- 验证放最后统一执行：`go build ./...`、`go vet ./internal/agent/...`
