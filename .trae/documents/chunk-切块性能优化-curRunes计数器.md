# Chunk 切块性能优化：curRunes 计数器 + runeLen 零分配

## Summary

针对 `ChunkContent` 的两处性能问题做**零行为变化**的优化：

1. 超限判断从 **O(n²) 重复 Join** 改为 **O(1) 累积计数器**（`curRunes`）
2. `runeLen` 从 `len([]rune(s))`（分配切片）改为 `utf8.RuneCountInString`（零分配）

不改切块输出语义，存量向量不受影响（不触发重新嵌入）。

## Current State Analysis

### 现状代码（internal/services/chunk.go）

- 主循环每次向 `cur` 追加行后，在空行分支（L217）与正文分支（L227）都执行
  `runeLen(strings.Join(cur, "\n")) > maxRunes` 判定超限：每加一行就把整个当前块重新 Join 一遍，**O(n²)**。
- `runeLen`（L316-318）实现为 `len([]rune(s))`，每次调用都分配切片。
- `flush()`（L165-189）中 `cur = nil` 重置块累积；块内容拼接、硬切判定、前缀拼接均不受影响。
- `runeLen` 全部调用点（L183 / L217 / L227 / L304）都在 `chunk.go` 内，无包外调用。

### 调用方与测试

- 写路径：`vector_service.go` L187（IndexNotes）、L417（classifyVectorNotes）调用 `ChunkContent`，两处共用 `chunkMaxRunes=600` 口径，输出变化会联动状态比对——**本次必须保证输出逐字节不变**。
- 测试：`chunk_test.go` 共 18 个测试函数，大量精确断言块内容（`HasPrefix` / 全等比较 / 去除前缀后拼接还原），是「输出不变」的天然回归保障。
- `playground/vec-poc/internal/chunk/` 是独立 PoC 副本，**非交付代码，不修改**。

## Proposed Changes

### 文件 1：internal/services/chunk.go

#### 改动 A：引入 `curRunes` 累积计数器（消除 O(n²) Join）

1. `ChunkContent` 中与 `var cur []string` 并列新增 `curRunes := 0`。
   - **语义**：`curRunes` 恒等于 `runeLen(strings.Join(cur, "\n"))`（块内行 rune 总和 + 行间换行符数）。
2. 所有向 `cur` append 的行均同步维护计数，**统一增量规则**：
   - append 前若 `len(cur) > 0`（非首行）则 `curRunes += 1`（分隔换行符，1 rune）；
   - 再 `curRunes += runeLen(line)`。
   - 覆盖 5 个 append 点：围栏开启（L204）、代码块内累积（L197）、标题行（L209）、空行（L216，`runeLen("")==0` 但可能需 +1 换行）、正文行（L225）。
3. 超限判定替换：
   - 空行分支 L217：`if runeLen(strings.Join(cur, "\n")) > maxRunes` → `if curRunes > maxRunes`
   - 正文分支 L227：同上替换。
4. `flush()` 内 `cur = nil` 重置处改为 `cur, curRunes = nil, 0`。
5. `flush()` 中的真实 Join（`strings.Join(cur, "\n")`）、硬切判定、前缀拼接**保持原样**——flush 每块仅一次，无性能问题。

**等价性证明**：`len(strings.Join(cur, "\n"))`（rune 数）= Σ`runeLen(行)` +（`len(cur)-1`）个换行。计数器按上述规则逐行累加，二者严格相等，输出逐字节不变。

#### 改动 B：`runeLen` 零分配

1. 文件 import 新增 `"unicode/utf8"`。
2. `runeLen` 实现改为：

```go
// runeLen 返回字符串的 rune 数量（Unicode 安全）
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
```

   调用点不变，全包受益（含 flush 内硬切判定与 splitWithHeading 预算）。

### 文件 2：internal/services/chunk_test.go（新增回归测试）

新增 `TestChunkMixedLongInput`：构造标题链 + 超 600 rune 长段落 + 代码围栏 + 多空行段落的大型混合输入（覆盖标题/空行/正文/围栏全部 append 点与多次超限 flush），断言：
- 每块 `runeLen ≤ maxRunes`；
- 去除元数据前缀后按块拼接可还原原文（不丢内容、不产生空块）；
- 输入量级（1000+ 行）确保计数器路径无 panic。

同时**不改动任何现有测试**——现有 18 个精确断言测试全绿即证明「输出不变」。

## Assumptions & Decisions

| 决策 | 说明 |
|------|------|
| 不改输出语义 | 计数器严格等价于原 Join 计算，杜绝存量块判「需重新嵌入」 |
| 不修 `playground/vec-poc` | 非交付 PoC（AGENTS.md 明示），不在本次范围 |
| 硬切边界等质量项不做 | 本次仅第 1、2 点性能修复，第 3 点起后续单独评估 |
| 不动 `chunkMaxRunes=600` | 保持写路径与状态比对口径一致 |
| `curRunes` 判定用 `>` 与原 `> maxRunes` 完全同语义 | 不引入边界偏移 |

## Verification

```bash
gofmt -l internal/services/chunk.go        # 无输出
go build ./...                             # 编译通过
go vet ./internal/services/                # 静态检查通过
go test ./internal/services/ -run 'Chunk' -v   # 切块相关测试全绿
go test ./internal/services/               # 全量服务层回归（含 vector_service 依赖切块口径的用例）
```

说明：纯后端改动，需 `wails build` 出新二进制才对前端生效（本次不含打包步骤）。
