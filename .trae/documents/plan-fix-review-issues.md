# 修复代码审查发现的全部问题

## 摘要

针对上一轮全面审查报告中的问题,按建议顺序逐项修复:

1. 删除 `buildToolStatusRows` 死代码
2. 折叠状态存储改造(消除内存无界增长 + 折叠态跨重渲染丢失)
3. 同步 5 处陈旧注释
4. 参数清理、折叠时收 tooltip、下标校验、reflow 优化
5. 风格统一(Go 文案分号、测试子名称)
6. 更新 AGENTS.md 长期/临时记忆

## 现状分析(问题定位)

| #     | 问题                                                                | 位置                                                        |
| ----- | ----------------------------------------------------------------- | --------------------------------------------------------- |
| P1-3  | `errs/parts/oks/runs/autos` 五数组构建后从未使用(聚合逻辑删除后的残留)                | [ai-chat.js#L6175-L6182](frontend/src/js/ai-chat.js)      |
| P1-1  | `collapsedOsAgents` 模块级 Set 永不清理;回放每次渲染都分配新 rid 并 add → 无界增长      | [ai-chat.js#L6025-L6033](frontend/src/js/ai-chat.js)      |
| P1-2  | rid 每次渲染重新生成,旧 rid 变孤儿,用户手动展开态被静默丢弃                               | 同上                                                        |
| P2-4  | `itemEl(rec, isLive, collapsed)` 的 `isLive` 参数**完全未使用**,且遮蔽外层同名参数 | [ai-chat.js#L6191](frontend/src/js/ai-chat.js)            |
| P2-5  | 5 处注释仍描述已删除的「聚合」行为                                                | L5972 / L5980 / L6170-6171 / L3626 / L6289                |
| P2-6  | 局部折叠时子步骤行被 `display:none`,不触发 mouseleave → 悬停原因卡残留                | [ai-chat.js#L6035](frontend/src/js/ai-chat.js)            |
| P2-7  | `parseInt(row.dataset.rid, 10)` 无校验;缺失时多个 NaN 行共享同一 key           | [ai-chat.js#L6040](frontend/src/js/ai-chat.js)            |
| P2-8  | 展开时循环内逐行 `void sib.offsetWidth` 强制同步布局                            | [ai-chat.js#L6050-L6054](frontend/src/js/ai-chat.js)      |
| P3-9  | Go 错误文案括号全角、分号半角                                                  | [config.go#L80](internal/config/config.go)                |
| P3-10 | 测试子名称含 `/` `\`,Go 会按 `/` 分层显示                                     | [config\_test.go#L66-L71](internal/config/config_test.go) |
| P3-11 | AGENTS.md 长期记忆 20 的「`hasAgentGroup` 降级逐条渲染」与代码不符;本次变更未记录          | AGENTS.md L324 / 七、临时记忆                                   |

### 关键设计依据:折叠 key 为何能改为「列表元素 + 下标」

* `renderToolCalls(el, toolCalls)` 的调用点只有两处([L4055](frontend/src/js/ai-chat.js) 流结束兜底、[L4536](frontend/src/js/ai-chat.js) 历史消息渲染),两者都在**新建消息 DOM 时**调用一次 → 同一条消息只要 DOM 存活,其 `records` 数组下标即稳定。

* 实时路径的 list 由 `ensureToolSummary` 创建一次并被复用,`buildToolStatusRows` 只 `innerHTML=''` 重建子节点 → **listEl 本身稳定**,以它为 WeakMap key 可在重建后恢复折叠态。

* 消息 DOM 整体销毁时(切换会话/重载历史),新 DOM 本就是全新视觉,回到默认折叠态符合预期。

## 修改方案

### A. ai-chat.js — 折叠状态存储改造(P1-1 / P1-2 / P2-7 / P2-8)

**A1. 替换全局变量为按列表隔离的 WeakMap**([L6025-L6059](frontend/src/js/ai-chat.js))

删除 `var _toolRidSeq = 0;` 与 `var collapsedOsAgents = new Set();`,替换为:

```javascript
/** 折叠状态存储：按「明细列表元素」隔离，值为该列表内已折叠的 os_agent 记录下标集合。
    用 WeakMap 而非模块级 Set 的原因：
    1) 列表 DOM 销毁后条目自动回收，不随消息反复渲染无限增长；
    2) key 为列表元素本身，同一列表被 innerHTML 重建（实时刷新）时状态仍可恢复；
    3) 下标在「同一条消息的 records 数组」内稳定，不依赖全局递增序列。 */
var collapsedOsAgentMap = new WeakMap();

/** 取某明细列表的折叠下标集合（惰性创建）。 */
function collapsedSetFor(listEl) {
    var s = collapsedOsAgentMap.get(listEl);
    if (!s) { s = new Set(); collapsedOsAgentMap.set(listEl, s); }
    return s;
}
```

**A2. 移除 6 处** **`rid: ++_toolRidSeq`** **赋值**

* 实时 3 处:[L3678](frontend/src/js/ai-chat.js)、[L3708](frontend/src/js/ai-chat.js)、[L3735](frontend/src/js/ai-chat.js)

* 回放 3 处:[L6076](frontend/src/js/ai-chat.js)、[L6102](frontend/src/js/ai-chat.js)、[L6130](frontend/src/js/ai-chat.js)

**A3.** **`itemEl`** **签名精简 + 下标落 dataset**([L6189-L6215](frontend/src/js/ai-chat.js))

* 签名改为 `function itemEl(rec, collapsed, idx)`(删掉完全未使用的 `isLive`,顺带消除遮蔽)

* `item.dataset.rid = rec.rid;` → `item.dataset.ridx = idx;`

* 同步更新函数上方的 JSDoc(collapsed 语义保留,补 idx 说明)

**A4.** **`buildToolStatusRows`** **删死代码 + 改用下标折叠集**([L6170-L6275](frontend/src/js/ai-chat.js))

* 删除 `errs/parts/oks/runs/autos` 声明与分类遍历(P1-3)

* 渲染循环改为:

```javascript
    var collapsed = collapsedSetFor(listEl);
    var groupCollapsed = false;
    records.forEach(function(r, idx) {
        // os_agent 主行先更新折叠态（决定自身箭头方向），其后子步骤沿用该组状态
        if (r.isAgentGroup) groupCollapsed = collapsed.has(idx);
        listEl.appendChild(itemEl(r, groupCollapsed, idx));
    });
```

**A5.** **`toggleOsAgentGroup`** **重写**([L6032-L6059](frontend/src/js/ai-chat.js))

统一解决 P2-6 / P2-7 / P2-8:

```javascript
function toggleOsAgentGroup(row) {
    var listEl = row.parentElement;
    if (!listEl) return;
    var idx = parseInt(row.dataset.ridx, 10);
    if (isNaN(idx)) return; // 防御：无下标的行不参与折叠（避免多个缺值行共享同一 NaN key）
    // 折叠/展开属交互行为，清除浏览器可能已建立的文本选区
    var sel = window.getSelection && window.getSelection();
    if (sel && sel.rangeCount > 0) sel.removeAllRanges();
    var collapsed = collapsedSetFor(listEl);
    var collapse = !collapsed.has(idx);
    if (collapse) collapsed.add(idx);
    else collapsed.delete(idx);
    _hideToolReasonTip?.(); // 折叠会隐藏行，先收起可能残留的悬停原因卡（mouseleave 不触发）
    row.classList.toggle('is-expanded', !collapse);
    // 仅遍历本组区间：主行之后、下一个 os_agent 主行之前
    var toAnimate = [];
    var sib = row.nextElementSibling;
    while (sib) {
        if (sib.classList.contains('is-os-agent')) break;
        if (sib.classList.contains('is-substep')) {
            sib.classList.toggle('is-collapsed', collapse);
            if (!collapse) { sib.style.animation = 'none'; toAnimate.push(sib); }
        }
        sib = sib.nextElementSibling;
    }
    if (toAnimate.length) {
        void row.offsetWidth; // 单次强制 reflow：批量重启入场动画（循环内逐行 reflow 代价高）
        toAnimate.forEach(function(el) { el.style.animation = ''; });
    }
}
```

**A6. stream-done 折叠改用列表集合**([L3927-L3933](frontend/src/js/ai-chat.js))

```javascript
        if (toolSummaryEl) {
            var stListEl = toolSummaryEl.querySelector('.ai-tool-status-list');
            if (stListEl) {
                var stCollapsed = collapsedSetFor(stListEl);
                toolRecords.forEach(function(rec, idx) { if (rec.isAgentGroup) stCollapsed.add(idx); });
            }
            if (toolStatusTimer) { clearInterval(toolStatusTimer); toolStatusTimer = null; }
            updateToolSummary(toolSummaryEl, toolRecords, true);
        }
```

**A7. 回放初始折叠改用列表集合**([L6329-L6330](frontend/src/js/ai-chat.js))

```javascript
    // 回放初始态：所有 os_agent 组默认折叠（只显示主行，点击展开查看组内子步骤明细）
    var initCollapsed = collapsedSetFor(list);
    records.forEach(function(r, idx) { if (r.isAgentGroup) initCollapsed.add(idx); });
```

### B. ai-chat.js — 注释同步(P2-5)

| 位置          | 改法                                                                |
| ----------- | ----------------------------------------------------------------- |
| L5972       | 「成功按工具聚合「名 ×N」」→「全部按原始顺序逐行渲染,不聚合;os\_agent 组可折叠」                  |
| L5980       | 栈元素 `{ callId, rec }` → `{ callId, rec, seqMap }`(seqMap 为组内独立计数) |
| L6170-L6171 | 删「失败的置前…成功聚合置后…顺序即优先级」,改为「按 records 原始顺序逐行渲染,折叠态按组下标记」            |
| L3626       | 去掉「成功按名聚合」,改为「逐行渲染」                                               |
| L6289       | 去掉「成功按工具聚合」,改为「逐行渲染,os\_agent 组默认折叠」                              |
| L6140       | 补一句：徽标计数按记录总数(含折叠组内记录),**不等于**可见行数                                |

### C. config.go — 文案分号统一(P3-9)

[L80](internal/config/config.go):`;请使用相对路径` → `;请使用相对路径`(半角改全角,与同句全角括号一致)

### D. config\_test.go — 子测试名清洗(P3-10)

* import 增加 `"strings"`

* [L66-L71](internal/config/config_test.go):子测试名对 `/` `\` 替换为 `_`,避免 Go 按 `/` 分层展示

### E. AGENTS.md — 文档同步(P3-11)

**E1. 长期记忆 20**([L324](AGENTS.md)):末句「前端实时与回放共用 `osAgentGroup*` 栈函数 + isAgentGroup/substep 标记 + `hasAgentGroup` 降级逐条渲染 + `.is-os-agent`/`.is-substep` 样式。」替换为:

> 前端实时与回放共用 `osAgentGroup*` 栈函数 + `nextToolSeq`（组内独立计数，os\_agent 主行走全局，子步骤走栈顶组内 seqMap）+ isAgentGroup/substep 标记；明细**按原始顺序逐行渲染不聚合**（失败行自带原因）；os\_agent 组支持折叠/展开——实时默认展开、回放与流结束后默认折叠，点击主行**局部切换**该组子步骤显隐（不整表重建）；折叠态存于模块级 `collapsedOsAgentMap`（WeakMap：listEl → 已折叠下标 Set），随列表 DOM 回收，避免无界增长。

**E2. 临时记忆**:按维护规范三步执行——删除 `临时记忆 1`,原 2→1、3→2、4→3、5→4,末尾追加新条目为 `临时记忆 5`,内容覆盖本次两类改动:

* **工作区路径失败治理**:`ls_dir`/`delete_item` 的 `fs.ReadDir(root.FS(), filepath.ToSlash(rel))`(Windows 反斜杠多层 rel 在 io/fs 适配器报 invalid argument);`SandboxFilePath` 前置拦截外来/畸形绝对路径;os\_agent 提示词禁猜 `~` 展开路径

* **os\_agent 记录渲染与折叠**:逐行渲染不聚合、组内独立计数、折叠交互与 WeakMap 折叠态存储

* 验证结果与「需 wails build」提示

**注意**:新条目内文件引用必须用项目相对路径(如 `frontend/src/js/ai-chat.js`),禁止绝对路径(维护规范第 4 条)。

## 假设与决策

* **不保留全局 rid**:确认 `renderToolCalls` 只在消息 DOM 新建时调用一次,下标在同消息内稳定,故以「listEl + 下标」作 key 足够;消息 DOM 整体重建时回到默认折叠属预期(视觉全新)。

* **`itemEl`** **的** **`isLive`** **直接删除**而非改名:实测函数体完全未引用该参数(`timeText` 闭包用外层 `isLive`),删除比改名更彻底。

* **reflow 批量化**:改为「先统一禁动画 → 单次 reflow → 统一恢复」,保持展开动画效果的同时把 N 次强制布局降为 1 次。

* **CSS 不改动**:折叠样式(`.is-collapsed` / `.ai-tool-collapse-icon`)与 `dataset` 改名无关,无需调整。

* **不改 header 计数语义**:折叠组内记录仍计入「已调用 N 次 / N 失败」徽标(折叠不应漏统计),仅更新注释说明。

* AGENTS.md 只动长期记忆 20 与临时记忆区,不动第一\~九章其它内容(结构未变)。

## 验证步骤

1. `go build ./...` / `go vet ./...` 通过
2. `go test ./internal/config/ ./internal/agent/tools/` 全绿(子测试名已清洗,输出可读)
3. `golangci-lint run ./...` 0 issues
4. `npm run build` 通过(仅既有 chunk 体积警告)
5. 静态审查:全文 grep `_toolRidSeq` / `collapsedOsAgents` / `aggItemEl` / `hasAgentGroup` / `dataset.rid` 应为 0 命中(仅 `collapsedOsAgentMap` / `dataset.ridx` 存在)
6. 手动验证(需 `wails build` 出新二进制):

   * 实时:os\_agent 组默认展开;点击主行局部折叠,其余行无重绘闪烁

   * 流结束后:组回到折叠;手动展开后**切换会话再回来**,新 DOM 为默认折叠(预期)

   * 历史消息:默认折叠,点击展开正常

   * 折叠一个组不影响其它组;无悬浮提示卡残留

   * 长时间使用后 `collapsedOsAgentMap` 无累积(WeakMap + DOM 回收)

