# 回放工具记录显示动作文案与结果悬停卡（方案 A+B，纯前端）

## 摘要

回放时所有成功记录都显示固定的「：已完成」，无信息量。经核实**数据其实都在**（`tool_calls` 落库含 `action_text` 与 `result`），只是 `ok` 分支渲染时未使用。

本次按方案 A+B 改造**纯前端**：

* **A**：`ok` 行内文案改用 `action_text`（如「读取文件：src/main.go」「执行命令：go build ./...」），为空时回退「：已完成」

* **B**：`ok` 行的 `result`（工具真实输出 / os\_agent 子 Agent 摘要）挂到既有悬停卡机制上

不改后端、不落库新字段（耗时属方案 E，本次不做）。

## 现状分析

### 渲染层：`ok` 分支丢弃全部信息

[itemEl](frontend/src/js/ai-chat.js) 的 `ok` 分支：

```javascript
} else if (rec.status === 'ok') {
    cls = 'is-done'; icon = 'check'; text = '：已完成';
}
```

* 不用 `rec.action_text`（同一函数 `auto` 分支却用了）

* 不用 `rec.result`（`error`/`partial` 分支却用了并挂 `dataset.tipText`）

成因：早期 `ok` 走聚合行（`名 ×N : 已完成`），聚合行确实只能写"已完成"；改为逐行渲染后文案未跟进。

### 数据层：所需字段均可用

`tool_calls` 落库为 `[]tools.Record`（[context.go](internal/agent/tools/context.go#L62-L69)）：`action`/`name`/`call_id`/`args`/`result`/`action_text`。

回放路径 [buildToolRecords](frontend/src/js/ai-chat.js) 已提取 `action_text` 与 `result` 到记录（`{name, seqName, status, action_text, result}`），**无需改动**。

### `action_text` 长度已由后端收敛

各工具的 `ActionText` 内部已截断，无需前端额外截断即可安全展示：

| 工具                      | 文案示例                  | 截断 |
| ----------------------- | --------------------- | -- |
| read\_file              | `读取文件：src/main.go`    | 30 |
| run\_command            | `执行命令：go build ./...` | 30 |
| ls\_dir                 | `列出目录：pkg/plugins`    | 30 |
| copy\_item / move\_item | `复制项：x` / `移动项：x`     | 30 |
| os\_agent（子 Agent 主行）   | `执行操作系统任务：克隆仓库并分析结构`  | 40 |

`result` 上限为 `MaxResultLen = 500` runes（[context.go](internal/agent/tools/context.go#L23)）。

### 悬停卡机制已通用，仅缺 `ok` 标题分支

* 触发：`messagesEl` 上 `mouseover` 委托，命中 `.ai-tool-status-item` 且存在 `dataset.tipText`（延迟 `HOVER_DELAY = 300ms`）

* 填充：[fillToolTip](frontend/src/js/ai-chat.js) 依据 `dataset.tipStatus` 决定标题——**目前只认 error/partial/auto，其余一律回退「调用失败」**，故 `ok` 必须补分支，否则会误显示"调用失败"

* 卡片结构（[index.html](frontend/index.html) `data-tip="tool-record"`）：标题 + `工具：<名>` + 正文（`.ai-tip-fulltext` 已 `white-space: pre-wrap; word-break: break-word`）

* 标题默认色即 `var(--accent)`（[ai-chat.css](frontend/src/css/components/ai-chat.css#L1630-L1636)），与行内 `is-done` 图标色一致 → **`ok`** **无需新增 CSS**

## 修改方案（仅 `frontend/src/js/ai-chat.js`）

### 修改 1：`itemEl` 的 `ok` 分支改用 `action_text` + 挂 `result`

`itemEl` 内 `ok` 分支改为：

```javascript
} else if (rec.status === 'ok') {
    cls = 'is-done'; icon = 'check';
    // 成功行展示动作文案（读哪个文件/跑什么命令/列哪个目录），无文案时回退通用文案
    var okText = rec.action_text || '';
    text = okText ? '：' + (okText.length > 56 ? okText.slice(0, 56) + '…' : okText) : '：已完成';
    // 成功结果（工具输出 / os_agent 子 Agent 摘要）挂悬停卡；空结果不弹卡
    if (rec.result) hasTip = true;
}
```

要点：

* 56 字上限与 `auto` 分支保持一致（后端已截 30\~40，此为防御性上限）

* 空 `action_text`（如 `buildToolRecords` 的「补行」分支只带 status/result、无 start 配对）回退「：已完成」，避免出现空冒号

### 修改 2：`hasReason` 语义改名 `hasTip`

`itemEl` 内 `hasReason` 现仅控制 `dataset.tip*` 的挂载。新增 `ok` 用法后它会对成功行也为 true，名字失真。在同函数内改名为 `hasTip`（含变量声明与 `if (hasTip) { ... }` 块，共 3 处），块内注释同步说明「失败原因 / 成功结果」。

### 修改 3：`fillToolTip` 补 `ok` 标题分支

```javascript
const fillToolTip = (trigger) => {
    const st = trigger.dataset.tipStatus || '';
    let title = '调用失败', cls = 'is-error';
    if (st === 'ok') { title = '执行结果'; cls = ''; }          // 成功：输出结果（中性 accent 色）
    else if (st === 'partial') { title = '部分来源失败'; cls = 'is-warning'; }
    else if (st === 'auto') { title = '完全访问自动放行'; cls = 'is-warning'; }
    toolEls.title.textContent = title;
    toolEls.title.className = ('ai-mode-tip-title ' + cls).trim();  // trim 防 ok 时尾随空格
    toolEls.name.textContent = trigger.dataset.tipTool || '—';
    toolEls.reason.textContent = trigger.dataset.tipText || '';
};
```

## 假设与决策

* **实时与回放同步生效**：`itemEl` 为两路径共用（符合「渲染单一事实源 / 所见即所存」），本次改动同时让实时行也可悬停查看结果，不区分路径。

* **行内只放** **`action_text`，`result`** **只进悬停卡**：`result` 最长 500 runes 且常为多行（目录列表、文件内容），行内展示会破坏行密度；悬停卡已支持 `pre-wrap` 多行显示。

* **不新增 CSS**：`ok` 标题用 `.ai-mode-tip-title` 默认 accent 色，与行内 `is-done` 一致；卡片宽度 `.ai-mode-tip.wide` = 340px 且已折行。

* **不改后端 / 不加耗时**：耗时（方案 E，需 `Record` 加 `duration_ms`）本次不做，回放仍无耗时。

* **已知限制（不修）**：`os_agent` 主行带 `user-select: none`（防连击选中整片记录），故其悬停卡中的子 Agent 摘要无法选中复制；如需复制需另开入口，超出本次范围。

* **`args`** **仍不展示**：方案 D 未纳入本次范围。

## 验证步骤

1. `npm run build` 通过（仅既有 chunk 体积警告）
2. 静态检查：`itemEl` 内 `ok` 分支已使用 `action_text`/`result`；`fillToolTip` 含 `st === 'ok'` 分支；`hasReason` 无残留引用
3. 手动验证（需 `wails build` 出新二进制）：

   * 回放历史消息：成功行显示具体动作（如 `run_command ×3 ：执行命令：go build ./...`），不再是「已完成」

   * 悬停成功行 → 卡片标题「执行结果」+ `工具：<名>` + 完整 `result`（多行正常折行）

   * 悬停失败/部分失败/auto 行 → 标题与配色保持原样（回归不破坏）

   * `os_agent` 主行悬停 → 显示子 Agent 结构化摘要

   * 空 `action_text` 的行（补行场景）仍显示「：已完成」；`result` 为空的行不弹卡

   * 实时流期间行为与回放一致（共用 `itemEl`）

