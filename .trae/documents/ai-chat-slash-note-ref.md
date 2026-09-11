# AI 输入框斜杠搜索笔记并入引用栏

## 摘要
在 AI 助手输入框（`#aiChatInput`，textarea）内，输入 `/关键词` 时于输入区上方弹出一个约 8 条的小菜单，实时搜索匹配的笔记；通过 ↑/↓ 选择、Enter 确认后，将该笔记按现有「引用笔记」链路并入引用栏（chips），并清除输入框里的 `/关键词`。纯 `/`、`/\s`（后无词）不触发，保留未来 AI 斜杠命令的扩展空间。

用户已确认的三个决策：
1. 触发语法：**纯 `/关键词`**。
2. 选中后：**清除输入框里的 `/关键词` 文字**。
3. 菜单条数：**约 8 条**。

## 现状分析
- 输入框为普通 `<textarea id="aiChatInput">`，位于 `.ai-chat-composer`（`position: relative`）内（[index.html L1179-1188](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1179-L1188)）。键盘处理走 `onInputKeydown`（[ai-chat.js L2634](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2634)），输入监听、发送按钮状态、长度截断均在 L901-929 注册。
- 引用栏链路完整可用：
  - `referencedNotes` 数组（`{id,title,notebook_name,truncated}`，[L93](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L93)）。
  - `confirmNoteSelection` 里 `App.GetNoteRefContext(ids)` 返回 `{notes:[...]}` → 合并去重 → `updateRefChips()` 渲染 → `saveCurrentSessionConfig()` 持久化（[L6891-6917](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6891-L6917)）。
  - `updateRefChips()`（[L6934](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6934)）、`removeRefNote`（[L7000](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L7000)）、去重同步 `roleplayNotes`。
- 搜索数据源现成：`App.SearchNotes(query, page, pageSize, notebookId, 'updated_at', '', '', tagIds)`，已被笔记引用浮层 debounce(200ms) 调用（[loadNoteList L6572](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L6572)）。
- ESC 处理链已验证：文档级 ESC 监听只在对应浮层可见时才关闭它，不会与斜杠菜单冲突；在 `onInputKeydown` 内处理 Esc 即可，无需动全局处理。
- 样式规范：`.ai-chat-composer` 为 `position:relative`，现有 `--card-bg`、`--bg-secondary`、`--shadow-dropdown`、`--accent` 等变量可复用（[ai-chat.css L1143-1156](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1143-L1156)）。菜单可绝对定位在 composer 上方。

## 变更方案

### 1. HTML — [frontend/index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html)
在 `.ai-chat-composer` 内、`.ai-chat-composer-main` 之前插入斜杠菜单容器：
```html
<div id="aiChatSlashMenu" class="ai-chat-slash-menu" style="display:none;"></div>
```
菜单挂在 composer（relative）下，绝对定位于 composer 上方（`bottom: calc(100% + 8px)`），天然浮在引用栏/输入区之上，不需要改布局抵消逻辑。

### 2. CSS — [frontend/src/css/components/ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)
在输入区样式附近新增 `.ai-chat-slash-menu` 块：
- 绝对定位，`left: 50% / transform: translateX(-50%)` 对齐 composer；`bottom: calc(100% + 8px)`；`width: min(460px, calc(100% - 32px))`。
- 背景 `var(--card-bg)`、边框、圆角、`box-shadow` 复用 `--shadow-dropdown`；`z-index` 高于输入区（≥ 6）。
- 顶栏小字操作提示（"↑↓ 选择 · Enter 引用 · Esc 关闭"）。
- 列表项：标题 + 笔记本小字；`.active` 高亮用 `color-mix(in srgb, var(--accent) 12%, var(--bg))` 背景 + accent 左边条；hover 同高亮。
- 空态："未找到笔记"。
- 轻量入场/回弹动效，遵循项目既有 `cubic-bezier(0.34, 1.56, 0.64, 1)` 风格（可选，保持克制）。

### 3. JS — [frontend/src/js/ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)
**状态变量（模块顶部 NOTES 引用区附近）**：
```js
let slashMenuEl = null;      // #aiChatSlashMenu
let slashOpen = false;       // 菜单是否显示
let slashQuery = '';         // 当前搜索词
let slashResults = [];       // [{id,title,notebook_name}]
let slashActiveIdx = -1;     // 高亮索引
let slashTimer = null;       // 搜索 debounce 定时器
```

**DOM 初始化**（在既有的 `inputEl` 相关初始化块 L338 附近补充）：
- 获取 `slashMenuEl = document.getElementById('aiChatSlashMenu')`。
- 添加一次性 `document` 的 `pointerdown` 监听：点击在菜单与输入框之外时关闭菜单（捕获阶段，允许先于输入框 blur 处理）。

**触发检测** — 新增 `handleSlashInput()`，挂到既有第一个 input 监听（L904 截断那个）之后、以 `appendEventListener` 追加独立监听，避免改动既有逻辑：
- 取 `caret = inputEl.selectionStart`、`prefix = value.slice(0, caret)`；`tail = prefix.split('\n').pop()`。
- 若 `tail` 形如 `/开头` 且长度 >1 且第 2 个字符非空白 → `openSlashMenu(tail.slice(1))`。
- 否则 → `closeSlashMenu()`。
- 追加时 `isStreaming` 则直接 `closeSlashMenu()`（不新开）。

**openSlashMenu(query)**：
- 设 `slashOpen=true`、`slashQuery=query`、显示容器。
- 默认高亮第 0 项：`slashActiveIdx = 0`（有结果后应用；空结果时无高亮）。
- 清空旧定时器后 `slashTimer = setTimeout(async () => { } , 200)` 内：`const res = await App.SearchNotes(query,1,8,0,'updated_at','','',[])`；`slashResults = res?.items||[]`；`renderSlashResults()`。
- 捕获异常时渲染空态。

**renderSlashResults()**：
- 生成项 HTMl：`DOC_ICON` + `title` + 笔记本小字；`data-index`。
- `slashResults.length===0` → 空态文案（仍显示菜单，便于用户看到"未找到"）。
- 应用高亮到 `.active`：`slashActiveIdx` 有效（在结果范围）时高亮对应项，否则不高亮。

**selectSlashNote(idx)**（确认 + 清除 + 并入引用栏）：
```js
const note = slashResults[idx];
// 1) 清除输入框里的 /关键词
const caret = inputEl.selectionStart;
const prefix = inputEl.value.slice(0, caret);
const lineStart = prefix.lastIndexOf('\n') + 1;   // 光标所在行起始
inputEl.value = inputEl.value.slice(0, lineStart) + inputEl.value.slice(caret);
inputEl.selectionStart = inputEl.selectionEnd = lineStart;
// 2) 并入引用栏（复用 confirmNoteSelection 链路）
const refContext = await App.GetNoteRefContext([note.id]);
if (refContext) {
    const ids = refContext.notes.map(n => n.id);
    const keepNotes = referencedNotes.filter(n => !ids.includes(n.id));
    referencedNotes = [...keepNotes, ...refContext.notes];
}
// 3) 关闭菜单、重渲染、持久化
closeSlashMenu();
updateRefChips();
await saveCurrentSessionConfig();
inputEl.focus();
inputEl.dispatchEvent(new Event('input', { bubbles: true })); // 让发送按钮/撑高更新
```

**closeSlashMenu()**：
- 清 debounce 定时器；`slashOpen=false`；隐藏容器；`slashTimer=null`；`slashResults=[]`；`slashActiveIdx=-1`。

**键盘接管** — 修改 `onInputKeydown`（L2634），在既有 Enter 逻辑之前插入：
```js
if (slashOpen) {
    if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
        e.preventDefault();
        if (slashResults.length === 0) return;
        const delta = e.key === 'ArrowDown' ? 1 : -1;
        slashActiveIdx = (slashActiveIdx + delta + slashResults.length) % slashResults.length; // 循环
        renderSlashResults();
        return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        if (slashActiveIdx >= 0 && slashActiveIdx < slashResults.length) selectSlashNote(slashActiveIdx);
        return;
    }
    if (e.key === 'Escape') {
        e.preventDefault();
        closeSlashMenu();
        return;
    }
}
```
若菜单打开时按下与斜杠无关的按键（如普通字符）不强制接管，交给正常输入，由 input 监听自动重估（输入非 `/词` 时会关闭菜单）。

**blur 兜底**：给 `inputEl` 加 `blur` 监听，`setTimeout(closeSlashMenu, 150)`，让菜单内点击有足够时间完成。

## 假设与决策
- 复用后端 `SearchNotes`/`GetNoteRefContext`，不新增 Go 方法与后端改动。
- 触发仅限「光标所在行末尾」以 `/词` 结尾；`/ `（带空格）或 `/` 单独不触发，保留 AI 命令扩展空间。
- 采用「定义无默认高亮，按 ↑/↓ 后才有高亮」的常规斜杠菜单交互；也可直接默认高亮第 0 项，两种均可，实现默认第 0 项高亮更利于快速回车（采用：**默认高亮第 0 项**，命中即高亮，Enter 直接引用）。
- 不与现有 Alt+↑/↓ 跳转用户消息冲突（该处理仅在 TEXTAREA/INPUT 未聚焦时生效，见 [L1455-1458](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1455-L1458)）。
- 流式回复期间禁止触发新的斜杠菜单（`isStreaming` 守卫），关闭按钮逻辑复用 `updateBarsAreaVisibility` 无需改动。

## 验证
1. 输入 `/关键字` → 输入区上方出现菜单，展示标题+笔记本，默认高亮第 0 项。
2. ↑/↓ 循环移动高亮；再次输入细分词会实时刷新（debounce）。
3. Enter 引用 → `/关键字` 被清除、引用 chips 出现、根因持久化（切换会话后仍在）。
4. Esc 关闭菜单且不影响输入框文字；纯 `/` 或 `/ 空格` 不弹菜单。
5. 点击菜单外部/窗口区域关闭；流式回复期间输入 `/` 不弹菜单。
6. 输入框多行时 `/` 位于行首/行尾均正确解析。
7. 运行项目前端 lint（若配置存在）确保无报错；手动回归既有 Enter 发送、Ctrl+Enter 换行、引用浮层功能无回归。