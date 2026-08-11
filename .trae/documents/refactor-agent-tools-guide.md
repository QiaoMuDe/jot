# TOOLS.md 转为纯工具开发规范指南（移除具体工具清单）

## Summary

将 `internal/agent/TOOLS.md` 从"工具清单 + 开发规范"混合文档，改造为**纯开发维护规范指南**：移除 §6 具体工具清单表格，明确"工具清单以 `tools/doc.go` 与 `registry.go` 为权威"，并同步调整 §2/§3 中涉及"更新 TOOLS.md"的流程描述，使新增工具时不再需要改动本文件。§1 架构树、§4 编写规范、§8 动作文案中的具体工具（web_search / manage_note / current_time）保留为**通用示例**（符合"提供通用示例"的意图）。

## Current State Analysis

现状（探索确认）：

- **TOOLS.md 共 8 节**，其中"具体工具信息"集中在：
  - §6「现有工具清单」表格（9 行：工具名/文件/依赖/功能/结构化收集）——**核心删除对象**
  - §3 第 86 行"含全部 9 个工具"——具体数量表述
  - §2 第 4 步（L63-67）「更新工具清单文档」——流程指向 doc.go×2（保留，但需强调 doc.go 是唯一权威）
  - §1 架构树已瘦身（仅 current_time.go / manage_note.go 作参考示例）、§4.4/§4.6/§8.3 的具体工具引用——均为**示例性质**，符合"通用示例"意图，保留
- **权威清单已存在**：[tools/doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go) 完整列出 9 工具 + 9 构造器（可作为唯一权威）；[registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go#L19-L31) 是注册与依赖注入的代码真相
- **[agent/doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/doc.go#L5-L6) 有小瑕疵**：只读工具分类列表（web_search / recall_notes / refine_search_query / get_stats）漏了 `get_current_time`，TOOLS.md 移除清单后 doc.go 权威性上升，建议顺带补齐
- 引用方：AGENTS.md 记忆点与历史 spec（.trae/specs/*）提及"TOOLS.md 表格同步"——均为**历史记录，不改**（用户已确认不追加 AGENTS.md 记忆点）

## Proposed Changes

### 1. `internal/agent/TOOLS.md`

1. **§2 第 4 步「更新工具清单文档」**（L63-67）改为：
   - 强调 Go 包文档是工具清单的**唯一权威来源**：`tools/doc.go` 工具列表与构造器名（必须）、`agent/doc.go` 结构说明（涉及新依赖时同步 `Deps` 说明）
   - 追加一句：**本指南（TOOLS.md）不维护具体工具清单，新增工具无需更新本文件**
2. **§3 第 86 行**：`（完整清单以 registry.go 为准，含全部 9 个工具）` → `（完整清单以 registry.go 的 buildTools 为准）`——移除具体数量
3. **§6「现有工具清单」整节**（L288-301 表格）替换为「工具清单（权威来源）」指引节：
   ```markdown
   ## 6. 工具清单（权威来源）

   本指南不维护具体工具清单。现有工具与构造器以以下代码真相为准：
   - [tools/doc.go](internal/agent/tools/doc.go)：工具列表与导出构造器名（权威清单）
   - [registry.go](internal/agent/registry.go#L19-L31)：注册顺序与依赖注入
   新增/删除工具时仅需同步上述 Go 文档，**无需更新本文件**。
   ```
   （保留 §6 编号，避免 §7/§8 重排导致文档内大量"见 §8/§8.3"交叉引用失效）
4. **其余章节不动**：§1 架构树（参考示例）、§2 第 5 步/§4/§5/§8（均为规范与示例表述）

### 2. `internal/agent/doc.go`（可选，建议一并做）

- L5-6 只读工具列表补 `get_current_time`：`只读：web_search / recall_notes / refine_search_query / get_stats` → `只读：web_search / recall_notes / refine_search_query / get_stats / get_current_time`
- 理由：TOOLS.md 移除清单后 doc.go 是唯一权威清单，此瑕疵会误导维护者

### 3. 不改动的文件

- `AGENTS.md`（用户确认不追加记忆点；历史记忆点保留为记录）
- `.trae/specs/*`、`.trae/documents/*`（历史文档）
- `tools/doc.go`（已完整，无需改动）
- 任何前端 / Go 业务代码

## Assumptions & Decisions

- §6 用"指引节"替换表格而非直接删除小节，保持 §7/§8 编号稳定、减少交叉引用破坏
- 具体工具示例（web_search / manage_note / current_time 等）保留在 §1/§4/§8——它们服务"通用示例"目的，不属于"具体工具清单"
- 权威清单收敛为：代码真相 `registry.go` + Go 文档 `tools/doc.go`（后者每次新增工具仍需同步，但这是 Go 包文档，与"维护指南文件"性质不同，符合用户意图）
- 用户已确认：不追加 AGENTS.md 记忆点，仅改 TOOLS.md（doc.go 补 `get_current_time` 为建议项，若严格限定范围可跳过）

## Verification

1. 通读 TOOLS.md，确认无残留具体工具清单表格、无"含全部 N 个工具"类数量表述
2. `Grep` 确认 TOOLS.md 中 §6 引用已被指引节替代，无失效链接（§2/§3/§5 引用的 registry.go / doc.go 链接仍有效）
3. 若执行可选改动：`go build ./...`（doc.go 注释改动不影响编译，确认即可）
4. 向用户确认改造后"新增工具流程"文档步骤（§2）描述一致
