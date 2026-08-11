# Checklist

- [x] 新增 [get_stats.go](internal/agent/tools/get_stats.go)：`getStatsTool` 结构体含 `note` / `vector` / `ctx` 依赖与编译期断言
- [x] `Info()`：Name=`get_stats`，Desc 说明只读概览及与 manage_* 的边界；参数 `action`（overview/month 枚举）、`year`、`month`（1-12）齐全
- [x] `InvokableRun()`：overview 动作返回 GetStats 全量概览 + GetIndexStatus 向量索引状态（字节按 KB/MB 格式化）；month 动作返回 GetMonthCounts 每日笔记数（year/month 缺省当前年月）
- [x] 参数校验：非法 action 枚举、month 越界（非 1-12）返回错误，不崩溃
- [x] `ActionText()`：overview（或缺省）→ `获取数据统计概览`；month → `获取月度笔记统计`；解析失败 → ""；其他 → `执行`
- [x] `NewGetStats(note, vector, ctx)` 构造器导出
- [x] [registry.go](internal/agent/registry.go) 已注册 `get_stats`（WrapWithError 包装，传 `p.deps.Note` / `p.deps.Vector` / `p.ctx`）
- [x] [TOOLS.md](internal/agent/TOOLS.md) §6 工具清单已补 `get_stats` 一行（文件/依赖/功能/结构化收集正确）
- [x] `go build ./...` 通过
- [x] `go vet ./internal/agent/...` 通过
