# 修复迭代上限设置项的审查问题

## 摘要

针对上一轮代码审查报告指出的问题逐项修复。核心是**把散落各处的迭代上限数值收敛到** **`internal/config`** **常量**（用户指定位置），并借此消除主/子 Agent 运行时钳制策略的不一致、补齐新增分支的测试覆盖。

覆盖：P1-1（测试缺口）、P2-2（钳制不一致）、P2-3（魔法数字收敛）、P3-4（术语统一）、P3-5（AGENTS.md 滞后）、P3-6（注释含变更历史）。P3-7（CSS 列宽的 UI 目视验证）需 `wails build` + 人工观察，本环境无法执行，单列为交付后待办。

## 现状分析

### 数值散落情况（P2-3 的实测）

| 数值             | 出现位置                                                                                                                                                                                                                                     |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 主 Agent 默认 100 | [db.go L220](internal/database/db.go)`"100"`、[types.go L136](internal/services/types.go)`100`、[agent.go L45](internal/agent/agent.go)`const DefaultMaxIterations = 100`                                                                  |
| 主 Agent 上限 500 | [types.go L191-L192](internal/services/types.go)、agent.go 运行时**无**上限、[index.html L651](frontend/index.html)`max="500"`、[main.js L2993-L2995](frontend/src/main.js)                                                                       |
| 子 Agent 默认 50  | [db.go L222](internal/database/db.go)`"50"`、[types.go L137](internal/services/types.go)、[subagent\_os.go L27](internal/agent/subagent_os.go) 常量、[index.html L659](frontend/index.html)`value="50"`、[main.js L3009](frontend/src/main.js) |
| 子 Agent 上限 200 | [types.go L196-L197](internal/services/types.go)、[subagent\_os.go L38-L39](internal/agent/subagent_os.go)、index.html `max="200"`、main.js L3013-L3014                                                                                     |

Go 侧可收敛（前端无法引用 Go 常量，保持现状）。

### 钳制语义不一致（P2-2 的实测）

* [agent.go L653-L659](internal/agent/agent.go)：主 Agent 运行时读取 `n > 0` 即采用，**无上限钳制**

* [subagent\_os.go L28-L42](internal/agent/subagent_os.go)：子 Agent 读取时 `> 200 → 200`

两条同语义读取策略不同；且该逻辑与 [tools/context.go L42](internal/agent/tools/context.go) 的 `getIntSetting` 属同一套语义（跨包 unexported，无法复用）。

### 测试覆盖缺口（P1-1 的实测）

[subagent\_os.go L28-L42](internal/agent/subagent_os.go) 有 4 条分支（nil / 解析失败或 `<1` / `>200` 钳制 / 正常值），而 [subagent\_test.go](internal/agent/subagent_test.go) 的 8 处 `buildOSSubAgent(...)` **全部传** **`nil`** → 仅覆盖第 1 条。

### 可复用的测试基建（已核实）

[internal/agent/tools/browse\_notes\_test.go L20-L37](internal/agent/tools/browse_notes_test.go) 已确立内存 SQLite 模式：`gorm.Open(sqlite.Open(":memory:"))` + `sqlDB.SetMaxOpenConns(1)`（内存库必须单连接）+ `AutoMigrate(&models.Setting{})` + `services.NewSettingService(db)` + `setting.Set(key, value)`。`glebarez/sqlite` 已在 go.mod 中，agent 包测试可直接采用。

### 依赖与循环校验（已核实）

`internal/config` 只 import 标准库（叶子包）→ `services` / `database` / `agent` 均可安全 import。现状：[db.go L8](internal/database/db.go)、[services/workspace\_service.go L22](internal/services/workspace_service.go)、[agent/tools/fs\_base.go L17](internal/agent/tools/fs_base.go) 已在引用；[types.go](internal/services/types.go) 与 [agent.go](internal/agent/agent.go) **尚未** import config，需补；[db.go](internal/database/db.go) 缺 `strconv`。

## 修改方案

### 1. `internal/config/config.go` — 新增取值范围常量（P2-3，用户指定位置）

在现有 `DirData/DirBackup/...` 常量块之后新增独立常量块（沿用本包「同一主题一组 const」风格）：

```go
// AI ReAct 循环迭代上限：默认值与上限（供种子初始化、设置读写校验、运行时装配共用，
// 避免同一数值在多处硬编码后相互漂移）。下限统一为「至少 1 轮」，见各 clamp 处。
const (
	AIAgentMaxIterationsDefault = 100 // 主 Agent 默认迭代上限
	AIAgentMaxIterationsMax     = 500 // 上限（超过取上限）

	AISubAgentMaxIterationsDefault = 50  // 子 Agent（os_agent）默认迭代上限
	AISubAgentMaxIterationsMax     = 200
)
```

> 为何不设 `Min` 常量：下限恒为「至少 1 轮」，写成通用常量反而引入 `parseIterationLimit` 的参数顺序歧义（三个同类型 int 相邻易错位）；保持字面量 `1` 与本文件其它 clamp（`LogLevel < 0`、`PageSize < 1`）风格一致。

同时把包注释（[L1-L2](internal/config/config.go)）从「仅路径解析」扩为「路径解析 + 应用级共享常量」，避免文档与内容不符。

### 2. `internal/database/db.go` — 种子引用常量（P2-3）

* import 补 `"strconv"`

* [L220-L222](internal/database/db.go) 改为：

```go
{Key: "ai_agent_max_iterations", Value: strconv.Itoa(config.AIAgentMaxIterationsDefault)},
// 子 Agent（os_agent）内层 ReAct 循环最大迭代次数
{Key: "ai_sub_agent_max_iterations", Value: strconv.Itoa(config.AISubAgentMaxIterationsDefault)},
```

### 3. `internal/services/types.go` — 默认值与 clamp 引用常量（P2-3）

* import 补 `"jot/internal/config"`

* `GetAllSettings`（[L136-L137](internal/services/types.go)）两处默认值 → `config.AIAgentMaxIterationsDefault` / `config.AISubAgentMaxIterationsDefault`

* `SaveAllSettings` clamp（[L189-L198](internal/services/types.go)）的**上限** → `config.AIAgentMaxIterationsMax` / `config.AISubAgentMaxIterationsMax`；**下限分支保持字面量** **`1`**（与相邻 clamp 写法一致），但回退值改为对应 Default 常量

> 注意：**读取默认值必须与种子值一致**是本项目硬性约定（settings-maintenance.md 第六节），改用同一常量后从机制上保证一致。

### 4. `internal/agent/agent.go` — 抽出共用读取函数 + 运行时钳制（P2-2 / P1-1 / P3-6）

* import 补 `"jot/internal/config"`

* **删除** `DefaultMaxIterations` 常量（[L44-L45](internal/agent/agent.go)，注释中的「已由 20 上调为 100」属变更历史，一并消除 → 修 P3-6）

* 在该位置新增两个函数（把「解析+钳制」抽成纯函数以便无 DB 单测）：

```go
// parseIterationLimit 解析并钳制迭代上限的设置原始值：缺失/非数字/小于 1 时回退 def，
// 超过 max 时取 max（纯函数，便于单测覆盖各分支）。
func parseIterationLimit(raw string, def, max int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// iterationLimitFromSetting 从设置服务读取迭代上限；未注入设置服务时回退 def。
// 主 Agent 与子 Agent 共用，保证两侧读取/钳制语义一致。
func iterationLimitFromSetting(setting *services.SettingService, key string, def, max int) int {
	if setting == nil {
		return def
	}
	return parseIterationLimit(setting.Get(key), def, max)
}
```

* `Run` 内读取（[L653-L659](internal/agent/agent.go)）替换为：

```go
	// 读取配置的迭代上限（未配置时回退默认值），防止 ReAct 循环死循环
	maxIterations := iterationLimitFromSetting(s.deps.Setting, "ai_agent_max_iterations",
		config.AIAgentMaxIterationsDefault, config.AIAgentMaxIterationsMax)
```

（顺序保持在「深度研究技能提升至 200」之前，该逻辑不变；主 Agent 默认 100 < 200 仍可被提升，用户设 500 时不降级。）

### 5. `internal/agent/subagent_os.go` — 复用共用读取函数（P2-3 / P2-2）

* 删除常量 `osSubAgentDefaultMaxIterations`（[L24-L26](internal/agent/subagent_os.go)）与函数 `subAgentMaxIterations`（[L28-L42](internal/agent/subagent_os.go)）

* `osAgentConfig.maxIterations`（[L75](internal/agent/subagent_os.go)）→ `config.AISubAgentMaxIterationsDefault`

* `buildOSSubAgent`（[L83-L86](internal/agent/subagent_os.go)）覆盖逻辑 →

```go
	cfg.maxIterations = iterationLimitFromSetting(setting, "ai_sub_agent_max_iterations",
		config.AISubAgentMaxIterationsDefault, config.AISubAgentMaxIterationsMax)
```

* import 调整：删 `"strconv"`（不再使用，否则编译报未使用导入）、补 `"jot/internal/config"`；`"jot/internal/services"` 保留（`buildOSSubAgent` 参数类型）

### 6. `internal/agent/subagent_test.go` — 补测试（P1-1）

* 新增 `TestParseIterationLimit`（纯函数表驱动，无需 DB，覆盖全部分支）：

  | 输入 raw  | def | max | 期望  |
  | ------- | --- | --- | --- |
  | `""`    | 50  | 200 | 50  |
  | `"abc"` | 50  | 200 | 50  |
  | `"0"`   | 50  | 200 | 50  |
  | `"-5"`  | 50  | 200 | 50  |
  | `"1"`   | 50  | 200 | 1   |
  | `"120"` | 50  | 200 | 120 |
  | `"200"` | 50  | 200 | 200 |
  | `"999"` | 50  | 200 | 200 |

* 新增 `TestIterationLimitFromSettingNil`：`iterationLimitFromSetting(nil, "k", 50, 200) == 50`（覆盖 nil 分支）

* 新增 `TestSubAgentMaxIterationsFromSetting`（内存 SQLite，沿用 [browse\_notes\_test.go L20-L37](internal/agent/tools/browse_notes_test.go) 模式）：

  * 迁移 `&models.Setting{}`，`setting.Set("ai_sub_agent_max_iterations", "120")` → `buildOSSubAgent(ctx, &openai.ChatModel{}, innerCtx, setting).cfg.maxIterations == 120`（验证端到端接线）

  * 不设该键（空库）→ 同上调用得 `50`

* import 补 `"github.com/glebarez/sqlite"`、`"gorm.io/gorm"`、`"jot/internal/models"`、`"jot/internal/services"`（`services` 若已存在则不重复）

### 7. `frontend/src/main.js` — 术语统一为 SubAgent（P3-4）

* [L3003](frontend/src/main.js) 注释 `// ── 子 Agent 最大运行次数保存 ──` → `// ── SubAgent 最大运行次数保存 ──`

* [L3010](frontend/src/main.js) / [L3015](frontend/src/main.js) / [L3019](frontend/src/main.js) 三处 toast 文案 `子 Agent 运行上限…` → `SubAgent 运行上限…`

* **保留** [index.html L657](frontend/index.html) 描述行的「子 Agent（os\_agent）」中文（有意作为 SubAgent 的解释，且无宽度约束问题）

### 8. `internal/agent/SUBAGENTS.md` — 同步常量/函数改名（P2-3 连带）

* [L83](internal/agent/SUBAGENTS.md) `maxIterations: osSubAgentDefaultMaxIterations,` → `config.AISubAgentMaxIterationsDefault`

* [L94](internal/agent/SUBAGENTS.md) 说明改为：上限由设置项 `ai_sub_agent_max_iterations` 提供（默认 50、范围 1–200，默认值与范围取自 `internal/config` 常量），装配时经 `iterationLimitFromSetting(setting, key, def, max)` 读取，未注入/非法时回退默认值

* [L103](internal/agent/SUBAGENTS.md) 示例体 `cfg.maxIterations = subAgentMaxIterations(setting)` → `cfg.maxIterations = iterationLimitFromSetting(setting, "ai_sub_agent_max_iterations", config.AISubAgentMaxIterationsDefault, config.AISubAgentMaxIterationsMax)`

### 9. `AGENTS.md` — 补记滞后内容（P3-5）+ 常量改名连带

* 临时记忆 5「前端」段补：标签文案改为「**SubAgent 运行上限**」（标题列宽限制）；`.ai-setting-label` 宽度 **112px → 136px**（全页 34 个设置项共用，原宽度放不下新标签导致与描述粘连）；toast/注释统一为 SubAgent 口径

* 临时记忆 5「后端」段中的 `osSubAgentDefaultMaxIterations` / `subAgentMaxIterations(setting)` 更新为 `config.AISubAgentMaxIterationsDefault` / `iterationLimitFromSetting`，并补一句「迭代上限默认值与取值范围集中在 `internal/config` 常量，种子/设置校验/运行时装配共用」

* 临时记忆仍保持 5 条、新条目在末尾（本次为就地更新既有第 5 条，**不触发移位**，因仍属同一批「迭代上限」工作）

### 10. `playground/agent-demo/main.go` — 悬空引用（常量被删的连带）

[L57](playground/agent-demo/main.go) 注释引用 `internal/agent.DefaultMaxIterations`（该常量将被删除）→ 注释改引 `internal/config.AIAgentMaxIterationsDefault`（值不变，仍为 100）

## 假设与决策

* **P2-2 取「两侧都运行时钳制」**：与保存侧 `SaveAllSettings` 形成纵深防御，可挡住「手工改库 / 旧数据写入越界值」；且两处共用 `iterationLimitFromSetting` 后语义天然一致。副作用：主 Agent 若库中被写入 >500，运行时会被压到 500（符合直觉）。

* **删除** **`agent.DefaultMaxIterations`** **导出常量而非保留别名**：避免同一数值两个名字；已核实无代码依赖（仅 `playground` 注释与历史 plan 文档提及），历史文档 `.trae/documents/add-deep-research-skill.md` 属存档不改。

* **不改前端硬编码数值**：前端无法引用 Go 常量；HTML `min/max/value` 与 main.js 校验值保持字面量，spec 已明确三处口径需人工保持一致。

* **不重构** **`tools.getIntSetting`**：它服务 `ai_http_max_chars`/`ai_read_url_max_chars` 等无关设置项，本次不扩大范围（评审中的「提升到公共位置」列为后续可选）。

* **P3-7（CSS 列宽目视验证）不在本次可执行范围**：需 `wails build` 出二进制后人工逐 Tab 检查，作为交付后待办列出。

## 验证步骤

1. `gofmt -l internal/` 无输出
2. `go build ./...` 通过（验证 import 增删正确，尤其 subagent\_os.go 删除 `strconv` 后无未使用导入）
3. `go vet ./...` 通过
4. `go test ./internal/agent/... ./internal/services/... ./internal/database/...` 全绿（含新增 3 个测试）
5. `go test ./internal/agent/ -run 'IterationLimit|SubAgentMaxIterations' -v` 逐个用例通过（覆盖 4 条分支 + nil + 端到端接线）
6. `golangci-lint run ./...` 输出 0 issues
7. `npm run build` 通过（仅既有 chunk 体积警告）
8. 静态核查：

   * `grep -rn "\bosSubAgentDefaultMaxIterations\b\|\bsubAgentMaxIterations\b\|agent\.DefaultMaxIterations" internal/ playground/` → **0 命中**（旧常量名与旧函数名已彻底移除）

   * `grep -rn "AIAgentMaxIterationsDefault\|AISubAgentMaxIterationsDefault\|AIAgentMaxIterationsMax\|AISubAgentMaxIterationsMax" internal/` → 确认 `config`（定义）、`db.go`、`types.go`、`agent.go`、`subagent_os.go` 均已引用

   * 确认 `GetAllSettings` 默认值与 `db.go` 种子值同源（均取自 config 常量）→ 结构上不可能不一致
9. 人工待办（需 `wails build`）：设置页逐 Tab 目视确认无新出现的描述省略号；两个输入的默认值/范围显示正确；改值保存后提示文案为 SubAgent 口径

