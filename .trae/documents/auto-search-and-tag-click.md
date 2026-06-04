# 自动搜索 + 标签可点击 改造计划

## Summary
三个改动：① 搜索框改为输入时自动搜索（250ms 防抖）；② 笔记卡片上的标签可点击，点击后搜索该标签；③ 后端搜索也匹配标签名称。

## Current State Analysis
- **搜索触发**：目前仅支持按回车或点击"搜索"按钮才触发搜索（`searchNotes()`）
- **标签渲染**：卡片网格中标签以 `.card-tag` 样式渲染，纯展示，不可点击
- **后端搜索**：`note_service.go:Search()` 仅搜索 `title LIKE` 和 `content LIKE`，不搜索标签名
- **标签搜索**：后端有 `GetByTag()` 方法，但前端从未通过标签名搜索过

## Proposed Changes

### 1. 前端：搜索框自动搜索（防抖 250ms）
- **文件**: `frontend/src/main.js`
- **改动**：
  - 新增 `debounce(fn, delay)` 工具函数
  - `initEventListeners()` 中：移除旧的 `keydown` 监听器，改为 `input` 事件 + 防抖（250ms）
  - 保留按 Enter 立即触发搜索（作为即时搜索的快捷方式）
  - 保留"搜索"按钮点击立即触发搜索
- **为什么**：用户只需输入，无需额外操作即可看到搜索结果

### 2. 前端：标签可点击
- **文件**: `frontend/src/main.js`
- **改动**：
  - 在 `renderCardGrid()` 的标签渲染中，给 `.card-tag` 添加 `onclick`，点击时：
    1. 将标签名填入搜索框（`els.searchInput.value = tag.name`）
    2. 调用 `searchNotes(tag.name)` 触发搜索
    3. 使用 `event.stopPropagation()` 阻止事件冒泡到卡片本身的 `onclick`
  - 在搜索列表（`renderSearchResults`）中也添加相同的标签点击逻辑
- **为什么**：用户可点击标签快速筛选相关笔记

### 3. 后端：搜索接口支持标签名称匹配
- **文件**: `services/note_service.go`
- **改动**：
  - 修改 `Search()` 方法，在原有的 `title LIKE` + `content LIKE` 基础上，增加 `OR` 条件：
    ```
    OR id IN (
      SELECT note_id FROM note_tags 
      JOIN tags ON tags.id = note_tags.tag_id 
      WHERE tags.name LIKE '%keyword%'
    )
    ```
  - 使用 GORM 的 `Where()` 链式调用或子查询实现
- **为什么**：用户搜索"工作"时，即使标题和内容不含该词，但标了"工作"标签的笔记也应出现在结果中

### 4. 前端：搜索状态显示优化
- **文件**: `frontend/src/main.js`
- **改动**：
  - 当通过标签点击触发搜索时，在搜索结果页的标题显示 `标签: "xxx"` 而非 `搜索: "xxx"`
  - 在 `switchView('search')` 调用前设置一个标记来判断触发来源

## Assumptions & Decisions
- 防抖时间 250ms：比 200-300ms 取中间值，响应及时且不会因频繁输入产生过多请求
- 标签点击后走搜索流程（而非调用 `GetNotesByTag`）：这样搜索结果页可统一复用，且用户可在此基础上继续输入过滤
- 后端用子查询而非 JOIN：避免因标签匹配导致重复行（一条笔记有多个标签可能匹配）

## Verification
1. 输入框中输入文字，等待片刻（250ms），观察是否自动触发搜索
2. 点击卡片上的标签，观察是否自动填充搜索框并显示搜索结果
3. 创建一个标签名为"工作"的笔记，然后在搜索框输入"工作"，检查该笔记是否出现在搜索结果中（即使标题/内容不含"工作"）
4. `npm run build` 构建验证
