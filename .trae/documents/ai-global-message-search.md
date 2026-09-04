# AI 助手全局消息检索功能实施计划

## 概要

将 AI 助手侧栏的内联会话搜索框（仅过滤会话标题）改造为**按钮触发的全局搜索弹窗**，参考笔记全局搜索弹窗（`#searchModal`）的交互模式与视觉设计，支持检索**所有历史会话的标题与消息内容**。点击结果跳转并定位到目标消息（自动加载更早历史 + 滚动 + 高亮闪烁）。

## 已确认的决策

| 决策点    | 结论                                                  |
| ------ | --------------------------------------------------- |
| 点击结果行为 | 跳转会话并定位到该条消息（scrollIntoView + 高亮闪烁）；标题命中仅跳转会话       |
| 摘要长度   | SQL 层固定截取：关键词首次出现位置前 40 + 后 80（共约 120 字符），不做可配置     |
| 检索范围   | 会话标题 + 消息内容（排除 `role='system'` 消息；GORM 软删除自动排除已删会话） |
| 原内联搜索框 | 完全移除（HTML/JS/CSS 三处）                                |
| Ctrl+F | 保持现状不改（用户未要求，避免扩大范围）                                |
| 过滤器    | 不加笔记本/标签/日期过滤器，保持简洁                                 |

## 现状分析（基于实际代码）

* 当前侧栏搜索：[index.html#L1107-1109](d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html) 内联输入框 `#aiSessionSearch`，[ai-chat.js](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js) 中 `sessionSearchQuery` 客户端过滤已加载会话标题（`renderSessionList` 内）

* 笔记全局搜索参考：弹窗结构在 index.html `#searchModal`（约 L2449-2513），样式 [search-modal.css](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/search-modal.css) 为**纯 class 选择器**（`.search-modal` 等），新弹窗可直接复用类名；逻辑在 [main.js](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js) `openSearchModal` / `handleSearchModalInput`（200ms 防抖 + `_searchModalLoadSeq` 防竞态）/ `searchModalLoadPage`（分页）/ `renderSearchModalResults`（escapeHtml + 高亮 `<mark class="ai-search-highlight">`）/ 滚动触底加载更多 / Esc 与遮罩关闭

* 后端参考：[note\_service.go](d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go) `Search`（LIKE + `ESCAPE '\'` + `escapeLike`）与 `noteThinSelect`（`INSTR + SUBSTR` SQL 层摘要截取）

* 数据模型：`AISession`（Title + DeletedAt 软删除）、`AIMessage`（SessionID/Role/Content/CreatedAt）

* 消息 DOM 已带 `data-msg-id`（`addMessage` 中 `el.dataset.msgId = msgId`），可直接定位

* 会话跳转已有 `switchSession(id)`（加载最近 6 条 + 滚动底部），滚动上滑加载更早消息逻辑在其 scroll handler 中（`LoadAISessionMessagesPaginated(sessionID, limit, beforeID)` 游标分页）

* 高亮样式 `.ai-search-highlight` 已存在于 [ai-chat.css](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)（模型下拉搜索在用），可直接复用

## 变更内容

### 1. 后端 — internal/services/ai\_service.go（新增）

新增结果类型：

```go
// AISearchSessionHit 会话标题命中项
type AISearchSessionHit struct {
    ID        uint   `json:"id"`
    Title     string `json:"title"`
    UpdatedAt string `json:"updated_at"` // "2006-01-02 15:04"，与 GetAISessions 一致
}

// AISearchMessageHit 消息内容命中项
type AISearchMessageHit struct {
    SessionID    uint   `json:"session_id"`
    SessionTitle string `json:"session_title"`
    MessageID    uint   `json:"message_id"`
    Role         string `json:"role"` // user / assistant
    Snippet      string `json:"snippet"`
    CreatedAt    string `json:"created_at"`
}

// AISearchResult 全局搜索结果（标题命中 + 消息命中分页）
type AISearchResult struct {
    Sessions     []AISearchSessionHit `json:"sessions"`
    Messages     []AISearchMessageHit `json:"messages"`
    TitleTotal   int64                `json:"title_total"`
    MessageTotal int64                `json:"message_total"`
}
```

新增 `func (a *AIService) SearchAIChat(keyword string, page, pageSize int) (*AISearchResult, error)`：

* keyword 去首尾空白后为空 → 返回空结果（err = nil）

* 标题命中：`Model(&models.AISession{})`（自动过滤软删除）`title LIKE ? ESCAPE '\'`，`Order("updated_at DESC").Limit(20)`，返回所有命中（上限 20）

* 消息命中：`Table("ai_messages m").Joins("JOIN ai_sessions s ON s.id = m.session_id AND s.deleted_at IS NULL")`，`m.role != 'system'`，`m.content LIKE ? ESCAPE '\'`

  * 先 `Count` 总数，再分页查询，`Order("m.created_at DESC")`，`Offset/Limit`

  * SQL 层摘要截取（照抄 `noteThinSelect` 模式，关键词内单引号转义 `''`）：
    `SUBSTR(m.content, MAX(1, INSTR(m.content, '<kw>') - 40), 120) AS snippet`（INSTR=0 时 `MAX(1,-39)=1` 自然退化为取前 120 字符，与笔记搜索行为一致）

* 复用包内已有 `escapeLike`（note\_service.go L214，同包可直接调用）

* 时间格式化：`Format("2006-01-02 15:04")`

### 2. 后端绑定 — app.go（新增 1 个方法）

在 `GetAISessions`（约 L2804）附近新增：

```go
// SearchAIChat 全局搜索 AI 会话标题与消息内容（分页）
func (a *App) SearchAIChat(keyword string, page, pageSize int) (*services.AISearchResult, error) {
    // 日志（Debugw 入参 / Errorw 失败 / Infow 成功，模式同 SearchNotes）
    return a.aiService.SearchAIChat(keyword, page, pageSize)
}
```

Wails 运行时绑定 `window.go.main.App.SearchAIChat` 在 `wails build` 时自动生成，前端直接调用即可（与现有调用方式一致）。

### 3. 前端 HTML — frontend/index.html（2 处）

**a) 侧栏改造**（L1107-1109）：保留 `.ai-session-search-wrap` 容器，内部 input 替换为按钮：

```html
<div class="ai-session-search-wrap">
    <button type="button" id="aiChatSearchBtn" class="ai-session-search-btn">
        [放大镜 SVG，同 search-modal-icon]
        <span>搜索会话与消息...</span>
    </button>
</div>
```

**b) 新增弹窗**（L2513 `#searchModal` 结束后、`#presetModalOverlay` 之前）：

```html
<!-- AI 全局搜索弹窗（侧栏搜索按钮触发） -->
<div id="aiSearchModal" class="search-modal" role="dialog" aria-modal="true" aria-label="搜索 AI 会话与消息">
    <div class="search-modal-mask"></div>
    <div class="search-modal-content">
        <div class="search-modal-header">
            <span class="search-modal-icon-wrap">[放大镜 SVG]</span>
            <input type="text" id="aiSearchModalInput" class="search-modal-input" placeholder="搜索会话与消息..." autocomplete="off" spellcheck="false" />
        </div>
        <div class="search-modal-results" id="aiSearchModalResults"></div>
        <div class="search-modal-empty" id="aiSearchModalEmpty" style="display:none">
            [空态放大镜 SVG]
            <p class="search-modal-empty-title" id="aiSearchModalEmptyTitle">开始搜索你的 AI 对话</p>
            <p class="search-modal-empty-desc" id="aiSearchModalEmptyDesc">输入关键字搜索会话标题或消息内容</p>
        </div>
        <div class="search-modal-footer" id="aiSearchModalFooter" style="display:none">
            <span id="aiSearchModalCount">共 0 条结果</span>
            <span>·</span><kbd>⏎</kbd><span>跳转</span>
        </div>
    </div>
</div>
```

无过滤器区（不需要）。

### 4. 前端 CSS — ai-chat.css 与 search-modal.css

**a)** **[ai-chat.css](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)**：

* 删除 `.ai-session-search` / `.ai-session-search:focus` / `.ai-session-search::placeholder`（约 L2783-2802）

* 新增 `.ai-session-search-btn`：复用原 input 的视觉规格（全宽、圆角、border、muted 文字、hover 高亮 border、左侧图标 + 文字），适配 button 元素（cursor、背景透明）

* 新增跳转闪烁动画 `.ai-msg-jump-flash`（背景短暂高亮渐隐约 1.6s keyframes，含 `prefers-reduced-motion` 降级）

**b)** **[search-modal.css](d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/search-modal.css)**（文件末尾追加 AI 弹窗专属样式，作用域限定 `#aiSearchModal`）：

* `.ai-search-item-top`：条目首行 flex（会话标题 + 角色 badge + 时间，间距对齐）

* `.ai-search-item-session`：会话标题（带小对话气泡 SVG，muted 色，单行 ellipsis）

* `.ai-search-item-role`：角色徽标（user=「提问」/ assistant=「回答」/ title 命中=「会话」，小圆角 chip，三种配色用主题变量）

* `.ai-search-item-time`：时间（muted、字号小、不收缩）

* 复用现有 `.search-modal-item` / `.search-modal-item-snippet` / `.search-modal-item.selected` / `.ai-search-highlight` 样式

### 5. 前端逻辑 — frontend/src/js/ai-chat.js

**a) 移除旧内联搜索**：

* 删除 `sessionSearchQuery`、`sessionSearchEl` 变量（L41-42）与 `getElementById` 赋值（L351）

* 删除 input 事件绑定块（约 L980-986）

* `renderSessionList` 中删除 `sessionSearchQuery` 过滤与标题高亮逻辑

**b) 新增弹窗模块**（照搬笔记弹窗模式，全部局部实现，文件内新增一组函数）：

* 元素引用 + 状态变量：`_aiSearchKeyword/_aiSearchPage/_aiSearchMessageTotal/_aiSearchHasMore/_aiSearchLoading/_aiSearchSeq/_aiSearchSelectedIdx`，pageSize = 30

* `openAiSearchModal()`：body 滚动锁定、`visible` 类、重置状态与 UI、`transitionend` + 500ms 兜底聚焦输入框

* `closeAiSearchModal()`：解锁滚动、`closing` 类 + 150ms 后移除（同笔记弹窗）

* 输入处理：200ms 防抖 + `_aiSearchSeq` 序号丢弃过期响应

* `aiSearchLoadPage(page, append)`：调用 `window.go.main.App.SearchAIChat(kw, page, 30)`；`append=false` 清空重渲染，`append=true` 追加；根据 `page*30 < messageTotal` 计算 `hasMore`；渲染空态/底部统计（`共 X 个会话 · Y 条消息`）；`console.warn('SearchAIChat 未绑定')` 兜底

* 结果渲染（`_aiEscapeHtml` + `_aiEscapeRegExp` + `_aiHighlight` 三个局部 helper，匹配处包 `<mark class="ai-search-highlight">`）：

  * 标题命中项：首行 = 会话标题（高亮）+ 「会话」徽标 + updated\_at；`data-session-id`，点击 → `jumpToSession(id)`（即 `switchSession(id)` + 关弹窗）

  * 消息命中项：首行 = 会话标题（不高亮，muted + 气泡图标）+ 角色「提问/回答」徽标 + created\_at；摘要行 = snippet（高亮）；`data-session-id` + `data-msg-id`，点击 → `jumpToMessage(sessionId, msgId)`

  * 顺序：标题命中组在前，消息命中组在后；条目 `data-idx` 供键盘导航

* 键盘导航：弹窗 keydown capture 阶段监听，↑↓ 循环移动 `.selected` + `scrollIntoView({block:'nearest'})`，Enter 打开当前选中项（无选中则第一项）

* Esc 关闭：`document.addEventListener('keydown', ..., true)` capture 阶段拦截（优先于 main.js 全局 Esc 处理），仅当弹窗 `visible` 时 `stopPropagation + preventDefault` 后关闭

* 遮罩点击关闭 + 滚动触底加载更多（`scrollTop + clientHeight >= scrollHeight - 200`，同笔记弹窗）

* 绑定按钮：`aiChatSearchBtn` click → `openAiSearchModal`（在原 sessionSearch 绑定位置）

**c) 新增** **`jumpToMessage(sessionId, msgId)`**（定位核心）：

```
1. closeAiSearchModal()
2. isStreaming → showNotification('回复进行中，暂时无法跳转', 'warning') + return
3. sessionId !== activeSessionId → await switchSession(sessionId)
   （switchSession 自带 isStreaming 守卫；同会话则跳过切换）
4. let target = messagesInnerEl.querySelector('[data-msg-id="<msgId>"]')
5. 未找到 → 循环加载更早消息直到找到或耗尽：
   while (!target && _oldestMsgId > 0):
       older = await LoadAISessionMessagesPaginated(activeSessionId, 50, _oldestMsgId)
       空数组 → _oldestMsgId = 0; break
       渲染并前插（见下）→ target = 重新查询
6. target 存在 → scrollIntoView({ block: 'center', behavior: 'smooth' })
   + 加 .ai-msg-jump-flash 类，动画结束后移除
7. 耗尽未找到 → showNotification('未找到该消息，可能已被删除', 'warning')
```

* 将 switchSession scroll handler 中的「渲染 olderMsgs + 前插 + 滚动位置恢复 + chatHistory.unshift」抽取为 `prependOlderMessages(olderMsgs)` 辅助函数，scroll handler 与 jumpToMessage 共用（避免逻辑重复）

* 跳转批次 50 条/次（正常上滑加载仍为 6 条/次，互不影响）

## 假设与边界

* `DeleteAISession` 会话删除后消息不会成为孤儿参与搜索（JOIN 已过滤；且现有删除逻辑会清理消息）

* 摘要 INSTR 为二进制匹配，英文大小写不同时窗口可能取到消息开头（与笔记搜索行为一致，可接受）

* SQLite LIKE 对 ASCII 大小写不敏感、中文精确匹配，与笔记搜索口径一致

* `reasoning_content` / `meta` / `tool_calls` 不参与检索（仅 `content` 与会话 `title`）

* 构建约束：改动后需 `cd frontend && npm run build` 再 `wails build` 才能生效（项目既定流程）

## 验证步骤

1. **编译**：`cd frontend && npm run build` → `wails build`，无编译错误
2. **功能**（运行应用后逐项验证）：

   * AI 侧栏原搜索框已变为按钮；点击按钮弹窗唤起、自动聚焦；Esc / 点遮罩正常关闭

   * 输入关键词 → 防抖后返回结果：标题命中（「会话」徽标）在前、消息命中（提问/回答徽标 + 摘要）在后，关键词高亮

   * 结果滚动到底部自动加载下一页；底部统计数字正确

   * ↑↓ 键盘导航 + Enter 跳转

   * 点击**其他会话**的消息命中 → 切换会话并定位滚动到该消息、闪烁高亮

   * 点击**当前会话**中已加载的消息 → 直接定位

   * 点击**当前会话**中很早的消息（不在最近 6 条窗口内）→ 自动逐批加载更早消息后定位成功

   * 点击标题命中 → 仅切换会话

   * 流式回复进行中点击结果 → 出现「回复进行中，暂时无法跳转」提示，不跳转

   * 空关键词 / 无结果空态文案正常

   * 回归：侧栏会话列表渲染正常（原过滤逻辑移除无残留报错）；笔记 Ctrl+F 搜索弹窗不受影响

