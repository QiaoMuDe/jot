# 修复 Agent 工具调用记录的渲染与计数问题

## 摘要

AI 聊天消息的工具调用记录存在两个与用户期望不符的问题:

1. **渲染规则**:消息中一旦出现 os_agent(子 Agent),`hasAgentGroup` 降级为**全部逐条渲染**,成功记录不再合并「名 ×N :已完成」。
2. **计数规则**:`seqName` 是全局递增序号,子 Agent 内层工具调用与父层/其他 Agent 共用一个计数器,导致「OS Agent 内 run_command 10 次」显示成全局第 11~20 次,子 Agent 内未单独计数。

用户已确认目标:**已完成的记录不保序,直接按工具合并显示;报错的记录单独显示其内容**;子 Agent 内工具调用单独计数,不与全局叠加。

## 现状分析(基于 ai-chat.js 实测)

### 数据流
- 实时路径:startStreaming 内 `unsubToolStatus`(L3658)收集 `ai:tool-status` 事件 → 构建 `toolRecords`(L3572 定义);`toolSeqMap`(L3573)全局计数;`osAgentStack`(L3577)组栈。
- 回放路径:历史落库 `tool_calls` 数组 → `renderToolCalls`(L6238)→ `buildToolRecords`(L5998)局部 `seqByName` 计数 + 局部 `agentStack`。
- 两条路径共用模块级 `osAgentGroupOpen/Close/Active`(L5973-5994)与 `updateToolSummary`→`buildToolStatusRows` 渲染。

### 问题点
| 位置 | 现状 | 问题 |
|------|------|------|
| L6205-6209 `buildToolStatusRows` | `hasAgentGroup` 时 `records.forEach` 全部逐条 | 成功不再聚合,与用户期望不符 |
| L3670 / L3700(实时)`toolSeqMap[name]++` | 组内子步骤也走全局计数 | 子 Agent 内计数与全局叠加 |
| L6009 / L6035(回放)`seqByName[name]++` | 同上 | 同上 |
| L6167 `itemEl` 显示 `名 ×seqName` | 序号是全局第 N 次 | 子 Agent 内显示非组内序号 |
| L6089 header | 「已调用 N 次」= 逐条记录总数 | 属汇总信息,保留(见决策) |

### 设计意图(AGENTS.md 临时记忆 20)
- 「hasAgentGroup 降级逐条渲染」原为保持「os_agent start → 内层子步骤 → os_agent result」组内顺序不被打散。
- 用户已明确放弃该顺序要求(完成的记录直接合并),故可移除降级分支。

## 修改方案(全部在 frontend/src/js/ai-chat.js)

### 1. os_agent 组内独立计数 —— 实时路径

**`startStreaming` 的 `unsubToolStatus`(L3668-3674)**:

- 开组:`osAgentGroupOpen` 压入栈元素时附加**组内独立计数 map**。改模块级函数签名:
  `function osAgentGroupOpen(stack, ev, rec, seqMap)` → 压入 `{ callId, rec, seqMap }`。
- 组内子步骤 `tool_start`(L3670):若 `osAgentGroupActive(osAgentStack)`,取**栈顶元素的 seqMap** 计数(`const seq = (top.seqMap[trName] = (top.seqMap[trName] || 0) + 1)`),组内从 1 开始;否则走原 `toolSeqMap[name]++`(全局)。
- 补行分支(L3700)同步:同样按「在组内用栈顶 seqMap」取序号。
- 关组不变(`osAgentGroupClose` 按 call_id/顺序弹栈,组内 seqMap 随栈元素丢弃,天然隔离)。
- 注意:同一函数内 os_agent 主行自身(`trName === 'os_agent'`)仍走**全局** `toolSeqMap`(它是父层工具),不要用组内 map。

### 2. os_agent 组内独立计数 —— 回放路径

**`buildToolRecords`(L5998-6072)**:

- `agentStack` 元素同样带独立 seqMap(经改签名后的 `osAgentGroupOpen` 传入 `{}`)。
- 子步骤记录(L6010-6012 与补行 L6036-6039):若在组内,`seq` 用栈顶 seqMap 计数;否则用 `seqByName[name]++`。
- os_agent 主行(`name === 'os_agent'`)仍用全局 `seqByName`。

### 3. 渲染统一为「失败逐条 + 成功聚合」 —— `buildToolStatusRows`(L6106-6222)

移除 L6205-6209 的 `hasAgentGroup` 降级分支,统一渲染(实时/回放共用,行为一致):

1. **running**(实时)逐条置顶(原逻辑)。
2. **失败 / 部分失败**:逐条独立显示,保留 `is-substep` 缩进标记(组内失败仍可见归属),显示各自报错内容(原 `itemEl` 逻辑)。
3. **auto**:逐条(原逻辑)。
4. **成功聚合**,按**分组隔离**统计:
   - key = `name`(普通记录,含 os_agent 主行)→ 聚合行 `名 ×N :已完成`(原 `aggItemEl`,无缩进)。
   - key = `name + ':sub'`(子步骤记录)→ 聚合行 `名 ×N :已完成`,加 `is-substep` 类(缩进/弱化,复用现有样式),计数 = **所有 os_agent 组内该工具的成功总次数**(跨组累计,不跨出子 Agent 域)。
5. 顺序:running → 失败 → 部分 → auto → 成功(子步骤聚合行在前、普通聚合行在后,聚合内部不保序)。

**`aggItemEl`(L6181)**:新增 `substep` 布尔参数,为真时 `item.classList.add('is-substep')`。

**语义保证**:父层 ls_dir 成功 3 次 + 组内 ls_dir 成功 5 次 → 显示「ls_dir ×3」(普通)+「ls_dir ×5」(子步骤缩进),互不叠加,符合用户诉求。

### 4. header(rebuildToolSummaryHeader L6075-6102)保持不变

- 「已调用 N 次 · M 个工具」维持逐条实际调用总数(含子 Agent 内层),它是整条消息的汇总真值,聚合仅影响明细展示;失败/部分徽标逻辑不变。
- 决策理由:若 header 改为聚合后行数,会低估真实调用量,反而误导。

## 假设与决策

- **放弃组内顺序**:按用户明确要求,完成记录不保序,成功聚合;仅失败/部分失败逐条(保留 substep 缩进以便归属)。
- **组内计数跨组累计**:同工具子步骤的成功聚合跨 os_agent 组合计(如 3 次 os_agent 调用累计 run_command 10 次 → 显示 run_command ×10),与「OS Agent 内只记这 10 次、不与全局叠加」一致。
- **失败行序号用组内序号**:组内失败显示该组内第 N 次;聚合行计数用组内累计,两者语义独立(失败行关注报错内容,序号仅辅助定位)。
- 子 Agent 内层**无**失败/部分失败但父层普通工具失败时,混合消息同样走统一聚合逻辑,无特殊分支。
- 仅改 `ai-chat.js` 一个文件;CSS 复用现有 `.is-substep`/`.is-os-agent` 样式,不改样式。

## 验证步骤

1. `npm run build` 通过(前端构建)。
2. 静态审查双路径一致性:实时 `unsubToolStatus` 与回放 `buildToolRecords` 的组内计数、聚合 key 逻辑对称(所见即所存)。
3. 手动验证(需 `wails build` 出新二进制):
   - 无 os_agent 消息:成功聚合「名 ×N :已完成」、失败单独(回归不破坏)。
   - 含 os_agent 消息:成功子步骤聚合(缩进)、os_agent 主行聚合、失败逐条且缩进归属正确。
   - 构造「父层 ls_dir 3 次 + OS Agent 内 ls_dir 5 次」场景:应显示 ls_dir ×3(普通)+ ls_dir ×5(缩进),不叠加。
   - 历史消息回放与实时渲染一致。
