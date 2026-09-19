# os\_agent 子 Agent 工具调用记录折叠展开功能

## 摘要

为 AI 聊天消息的工具调用记录添加 os\_agent 子 Agent 级别的折叠/展开功能:

* **实时(Agent 工作中)**:所有子步骤默认展开,方便观察执行进度

* **回放/流结束后**:子 Agent 记录默认折叠,只显示 os\_agent 主行,点击展开查看细节

## 现状分析

### 当前渲染方式

`buildToolStatusRows`([L6119](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6119))逐条渲染所有记录,子步骤通过 `.is-substep` 类缩进显示。无折叠逻辑。

### 关键代码位置

* 记录构建:实时 `unsubToolStatus`([L3658](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3658))、回放 `buildToolRecords`([L6011](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6011))

* 渲染入口:实时 `refreshToolStatus`([L3620](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3620))、回放 `renderToolCalls`([L6233](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6233))

* 流结束: `ai:stream-done`([L3913](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3913)),此时折叠整个工具摘要条

* CSS: `.is-os-agent`([L725](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L725))、`.is-substep`([L748](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L748))

* 流级变量: `toolRecords`/`toolSeqMap`/`osAgentStack`([L3572-3577](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3572-L3577))

### DOM 重建机制

`buildToolStatusRows` 每次调用都 `innerHTML = ''` 全量重建。需要跨重建持久的折叠状态。

## 修改方案(仅改 ai-chat.js + ai-chat.css)

### 1. 折叠状态存储

在 `buildToolRecords` 函数之前(L6009)新增模块级变量:

```javascript
var collapsedOsAgents = new Set(); // 跨 DOM 重建持久：记录已折叠的 os_agent 记录 ID
```

每条记录在构建时赋予唯一 `rid`:

* 实时路径: `rec.id = nextToolId++`(已有自增 ID 机制,复用即可)

* 回放路径: `buildToolRecords` 内新增 `var nextRid = 1; rec.rid = nextRid++`

折叠状态以 `rec.rid` 为 key 存入 `collapsedOsAgents`。

### 2. 回放初始化:全部折叠

`renderToolCalls`([L6233](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6233))调用 `updateToolSummary` 之前,遍历 records 将所有 os\_agent 记录的 rid 加入 `collapsedOsAgents`:

```javascript
records.forEach(function(r) { if (r.isAgentGroup) collapsedOsAgents.add(r.rid); });
updateToolSummary(summary, records, false);
```

### 3. 实时初始化:全部展开

`startStreaming`([L3568](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3568))的变量声明区域,清空折叠集合:

```javascript
collapsedOsAgents.clear();
```

### 4. 流结束后:折叠

`ai:stream-done`([L3913](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3913))处理中,在折叠整个工具摘要条之前,将当前流的所有 os\_agent 记录加入折叠集合:

```javascript
toolRecords.forEach(function(r) { if (r.isAgentGroup) collapsedOsAgents.add(r.rid); });
```

### 5. `itemEl`:os\_agent 行追加折叠箭头

`buildToolStatusRows` 的 `itemEl` 函数([L6136](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6136))中,当 `rec.isAgentGroup` 为 true 时:

* 在 `nameEl` 前插入折叠箭头 `<span class="ai-tool-collapse-icon">`

* 箭头图标:折叠态 `chevron-right`、展开态 `chevron-down`(项目已有 `svgIcon` 函数和 chevron 图标)

* 点击箭头(或整行) → toggle `collapsedOsAgents` 中该 rec.rid → 调用 `refreshToolStatus()`(实时)或手动重建列表(回放)

* 需要区分实时/回放:传入 `isLive` 参数判断,实时用 `refreshToolStatus`,回放用 `updateToolSummary`

### 6. `itemEl`:substep 行读取折叠状态

当 `rec.substep` 为 true 时,检查 `collapsedOsAgents.has(rec.rid)`:

* 命中 → `item.classList.add('is-collapsed')`

### 7. CSS:折叠样式

```css
/* 子 Agent 折叠态：隐藏子步骤行 */
.ai-tool-status-item.is-collapsed {
    display: none;
}

/* os_agent 行折叠箭头 */
.ai-tool-collapse-icon {
    display: inline-flex;
    align-items: center;
    margin-right: 4px;
    transition: transform 0.15s ease;
    cursor: pointer;
}
.ai-tool-collapse-icon svg {
    width: 12px;
    height: 12px;
}
/* 展开态箭头旋转 */
.ai-tool-status-item.is-os-agent .ai-tool-collapse-icon {
    transform: rotate(0deg);
}
.ai-tool-status-item.is-os-agent.is-expanded .ai-tool-collapse-icon {
    transform: rotate(90deg);
}
```

### 8. 回放路径 click 事件绑定

`renderToolCalls` 中,在 `updateToolSummary` 之后,为所有 os\_agent 行绑定 click:

```javascript
// 用事件委托：list 上统一监听 click，判断目标是否为 os_agent 行
list.addEventListener('click', function(e) {
    var row = e.target.closest('.ai-tool-status-item.is-os-agent');
    if (!row) return;
    var rid = parseInt(row.dataset.rid, 10);
    if (collapsedOsAgents.has(rid)) collapsedOsAgents.delete(rid);
    else collapsedOsAgents.add(rid);
    updateToolSummary(summary, records, false);
});
```

### 9. 实时路径 click 事件绑定

`ensureToolSummary`([L3579](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3579))中,为 summary body 的 list 绑定事件委托(只需绑定一次):

```javascript
list.addEventListener('click', function(e) {
    var row = e.target.closest('.ai-tool-status-item.is-os-agent');
    if (!row) return;
    var rid = parseInt(row.dataset.rid, 10);
    if (collapsedOsAgents.has(rid)) collapsedOsAgents.delete(rid);
    else collapsedOsAgents.add(rid);
    refreshToolStatus();
});
```

## 假设与决策

* **折叠粒度**:按 os\_agent 粒度(整个子 Agent 一起折叠),不支持单个工具折叠——粒度过细会增加复杂度,且子 Agent 内工具通常较少

* **成功/失败一致折叠**:无论 os\_agent 成功或失败都可折叠,失败时外层仍显示红色 × 标记

* **流结束时自动折叠**:stream-done 后整个工具摘要条收起(已有逻辑),用户手动展开后看到的是折叠态的 os\_agent 子步骤——符合「历史看结果为主」的预期

* **实时不自动折叠**:Agent 工作中保持展开,方便观察进度;流结束后才折叠

## 验证步骤

1. `npm run build` 通过
2. 实时验证:AI 对话触发 os\_agent 调用 → 工具摘要条内 os\_agent 子步骤展开显示,可点击折叠/展开
3. 流结束后:工具摘要条自动收起,手动展开后 os\_agent 子步骤默认折叠,显示箭头,点击展开
4. 历史消息:点开工具调用记录 → os\_agent 子步骤默认折叠,点击展开
5. 多个 os\_agent:每个独立折叠/展开,互不影响
6. `prefers-reduced-motion`:箭头旋转动画 respect reduced-motion

