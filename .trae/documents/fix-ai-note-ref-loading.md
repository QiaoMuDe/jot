# 修复 AI 引用笔记选择器的加载体验

## 摘要

AI 助手中的引用笔记选择器存在两个问题：1) 快速滚动到底部时出现白屏（加载指示器未拦阻）；2) 查询速度明显慢于笔记首页搜索框。本计划从 DOM 结构、加载策略和查询优化三个维度解决。

## 当前状态分析

### DOM 结构

```html
<div class="ai-note-ref-list-wrap" id="aiNoteRefListWrap">   ← 滚动容器 (overflow-y:auto, max-height:360px)
    <div class="ai-note-ref-list" id="aiNoteRefList">          ← flex 容器
        ...条目...
    </div>
    <!-- loader 不在此层级 -->
</div>
```

加载指示器通过 JS 动态创建并 `appendChild` 到 `refList`（内层 flex 容器），而非 `refListWrap`（滚动容器）。

### 问题一：白屏原因

1. **加载器位置**：作为 flex 子项跟在最后一个条目后面，无法在可视区底部"拦截"
2. **DOM 间隙**：`appendToList()` 先 `loader.remove()` 再 `fragment.firstChild` 逐个 append，两者之间有短暂的 DOM 空隙
3. **触发距离**：距底部 200px 才触发，快速滚动时用户已越过加载器位置

### 问题二：查询慢原因

AI 引用浮层每次打开都发起 **3 次 Wails RPC 调用**：
1. `loadRefPageSize()` — 调用 `GetPageSize()`
2. `loadAllNotebooks()` — 调用 `GetAllNotebooks()`
3. `loadNoteList()` — 调用 `SearchNotes()` 或 `GetNotes()`

而搜索弹窗打开时只需 **1 次** RPC 调用（`SearchNotes`）。

SQLite 对并发写入有锁，`loadAllNotebooks()` 和 `loadNoteList()` 并行执行可能产生资源竞争。

### 问题三：加载器闪一下就被替换

加载器被移除后新条目还未插入的间隙，浏览器可能触发渲染导致白块闪烁。

## 改动方案

### 文件清单及改动

#### 1. `frontend/index.html`

**添加**：在 `#aiNoteRefList` 之后、`#aiNoteRefListWrap` 闭合之前，新增一个静态的加载指示器元素：

```html
<div class="ai-note-ref-list-loader" id="aiNoteRefListLoader">
    <span class="loading-spinner"></span> 加载中...
</div>
```

删除 `loadNoteList()` 中动态创建 loader 的代码，改为控制其 `visible` class。

#### 2. `frontend/src/css/components/ai-chat.css`

**修改**：`.ai-note-ref-list-loader` 增加隐藏/显示控制：
- 默认 `display: none`
- `.visible` 时 `display: flex`

**修改**：`.ai-note-ref-list-wrap` 保持 `position: relative`

#### 3. `frontend/src/js/ai-chat.js`

**a) 初始化时缓存**：在 `initAIChat()` 或首次打开浮层时，一次性加载 `_refPageSize` 和笔记本列表，后续打开不再重复请求。

- `loadRefPageSize()` 改为只在 `_refPageSize === null` 时调用，初始缓存值
- `loadAllNotebooks()` 改为只在 `_notebooksCache === null` 时调用，初始缓存值
- 新增 `_pageSizeLoaded = false` 和 `_notebooksCache = null` 状态变量

**b) 加载指示器控制**：
- `loadNoteList(append=true)`: 显示 loader（`classList.add('visible')`）
- `appendToList()`: 先 append 条目，再隐藏 loader（`classList.remove('visible')`）
- `catch` 块：隐藏 loader

**c) 滚动触发距离**：从 200px 改为 **400px**，让加载更早开始

#### 4. `frontend/src/js/ai-chat.js` — scroll 事件微调

保持绑定在 `refListWrap` 不变，仅改阈值。

## 假设与决策

1. **笔记本列表不需要实时刷新**：用户在一个会话中很少创建/删除笔记本，缓存到下次会话或手动刷新即可。如需实时性可以后续加一个弱刷新机制。
2. **分页大小也不需要实时读取**：设置页修改分页大小后，下次打开浮层时才会生效，中间不自动同步。这是可接受的延迟。
3. **加载器采用静态元素而非动态创建**：避免每次创建/销毁的 DOM 操作和 `id` 查找，更稳定可控。

## 验证步骤

1. 打开 AI 助手 → 引用笔记选择器 → 快速滚动到底部
   - 应看到底部的"加载中..."指示器（白屏消失）
   - 数据到达后指示器被新条目替换，无闪烁
2. 打开速度对比
   - 首次打开应比之前略慢（需要缓存加载）
   - 后续打开应显著加快（缓存命中，仅 1 次 RPC）
3. 笔记本下拉框
   - 首次打开填充正常
   - 后续打开下拉选项应立即出现
4. 手动刷新场景（如切换笔记本筛选）仍能正常加载新数据
