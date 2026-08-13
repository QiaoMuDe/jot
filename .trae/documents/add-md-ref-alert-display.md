# MD 语法页补充 Alert 引用块（引用扩展语法）展示

## 摘要

MD 语法帮助页（`#viewMdRef`）目前只展示了基础引用语法（`> 文本`、嵌套引用），缺少对 GitHub 风格 Alert 引用块扩展语法（`> [!NOTE]` / `> [!TIP]` / `> [!IMPORTANT]` / `> [!WARNING]` / `> [!CAUTION]`）的展示。该扩展语法在编辑器插入操作（[md-syntax.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions/md-syntax.js) 中的 NOTE/TIP/IMPORTANT/WARNING/CAUTION）和渲染管线（[preview-worker.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/preview-worker.js) 与 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 均注册了 `marked-alert` 插件）中已支持，帮助页预览区样式（[md-reference.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/md-reference.css) 中 `.md-ref-preview .markdown-alert` 系列）也已就绪，只缺一张展示卡片。

## 现状分析

* 帮助页卡片定义位于 [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1511-L1923)，每张卡片结构：`.md-ref-card`（含 `id`）→ `.md-ref-badge` 标题 → 源码面板（`<pre><code>` 内为 HTML 转义后的源码）→ 预览面板（`.md-ref-preview` 带 `data-ref` 序号）→ 底部说明 + 「打开编辑器试试」按钮 → 隐藏的 `<script type="text/plain" class="md-ref-source">`（原始 Markdown）。

* 引用卡片 `#md-ref-card--blockquote` 位于 [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1734-L1774)，展示基础引用 + 嵌套 + 引用内行内语法。

* 渲染逻辑 [renderMdRefCards()](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L7024-L7090) 遍历所有 `.md-ref-card`，用 `marked.parse`（已注册 `marked-alert`）渲染 `.md-ref-source` 到预览区；复制按钮、尝试按钮均通过 `querySelectorAll` 自动绑定，无需额外 JS 改动。

* 预览区 Alert 样式已在 [md-reference.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/md-reference.css#L300-L362) 完整定义。

* 卡片入场动画延迟按 `.md-ref-card:nth-child(N)` 递增（当前到 nth-child(10)），新增卡片后末尾卡片会无延迟，需补两条规则。

## 变更方案

### 1. `frontend/index.html` — 新增 Alert 引用块卡片

**TOC 项**：在 `💬 引用` 的 TOC 项（第 1524 行）后新增：

```html
<a class="md-ref-toc-item" href="#md-ref-card--alerts">💡 Alert 引用块</a>
```

**新卡片**：在引用卡片 `#md-ref-card--blockquote`（第 1774 行 `</div>`）之后、表格卡片之前插入一张新卡片，结构与现有卡片完全一致：

```html
<!-- 卡片 7: Alert 引用块 -->
<div class="md-ref-card" id="md-ref-card--alerts">
    <div class="md-ref-card-header">
        <span class="md-ref-badge">💡 Alert 引用块</span>
    </div>
    <div class="md-ref-card-body">
        <div class="md-ref-source-panel">
            <div class="md-ref-editor-bar">
                <div class="md-ref-editor-dots">
                    <span class="md-ref-editor-dot red"></span>
                    <span class="md-ref-editor-dot yellow"></span>
                    <span class="md-ref-editor-dot green"></span>
                </div>
                <span class="md-ref-editor-filename">alerts.md</span>
                <button class="md-ref-editor-copy-btn" title="复制源码">复制</button>
            </div>
            <pre><code>&gt; [!NOTE]
&gt; 提示信息

&gt; [!TIP]
&gt; 小技巧

&gt; [!IMPORTANT]
&gt; 重要提醒

&gt; [!WARNING]
&gt; 警告内容

&gt; [!CAUTION]
&gt; 小心操作</code></pre>
        </div>
        <div class="md-ref-preview-panel">
            <div class="md-ref-preview" data-ref="6"></div>
        </div>
    </div>
    <div class="md-ref-card-footer">
        <div class="md-ref-card-footnote">在引用首行添加 <kbd>[!类型]</kbd> 标记生成彩色提示块，支持 5 种类型</div>
        <button class="md-ref-try-btn">打开编辑器试试</button>
    </div>
    <script type="text/plain" class="md-ref-source">> [!NOTE]
> 提示信息

> [!TIP]
> 小技巧

> [!IMPORTANT]
> 重要提醒

> [!WARNING]
> 警告内容

> [!CAUTION]
> 小心操作</script>
</div>
```

说明：

* 源码示例内容与 [md-syntax.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions/md-syntax.js#L224-L294) 中五个 Alert 插入操作的默认模板保持一致。

* `<pre><code>` 中 `>` 需写为 `&gt;`（HTML 转义），隐藏源码脚本中保持原始 `>`。

* `data-ref` 仅为历史遗留的静态序号（JS 中未使用，已确认），为保持连续性将插入位置之后的卡片序号顺延。

**data-ref 序号顺延**（新卡片为 6，后续依次 +1）：

* 表格卡片 `data-ref="6"` → `"7"`（第 1799 行）

* 任务列表卡片 `data-ref="7"` → `"8"`（第 1835 行）

* 分割线卡片 `data-ref="8"` → `"9"`（第 1871 行）

* 转义卡片 `data-ref="9"` → `"10"`（第 1907 行）

### 2. `frontend/src/css/components/md-reference.css` — 补充入场延迟规则

新增卡片后 `.md-ref-content` 子元素变为 12 个（TOC + 11 张卡片），当前延迟规则只到 `nth-child(10)`，分割线与转义卡片将无延迟。在 [md-reference.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/md-reference.css#L70-L71) 的 nth-child 规则后追加：

```css
.md-ref-card:nth-child(11) { animation-delay: 0.60s; }
.md-ref-card:nth-child(12) { animation-delay: 0.66s; }
```

### 无需改动的部分

* JS 渲染 / 复制 / 尝试按钮绑定：均按 `.md-ref-card` 自动遍历，新卡片自动生效。

* Alert 预览样式：`.md-ref-preview .markdown-alert` 系列样式已存在。

## 假设与决策

* 采用「新增独立卡片」而非「扩展现有引用卡片」，与编辑器插入菜单中「引用」和五个 Alert 操作分项展示的既有组织方式一致。

* 五种 Alert 类型全部展示，与 [md-syntax.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions/md-syntax.js) 支持的类型一一对应。

* 卡片插入位置紧邻「💬 引用」卡片之后，语义归类相邻；TOC 相应加一项。

## 验证

1. 启动应用进入「更多菜单 → MD 语法」或 Ctrl+P 启动器 → MD 语法。
2. 检查 TOC 出现「💡 Alert 引用块」且点击可平滑滚动到新卡片。
3. 新卡片源码面板显示 `> [!NOTE]` 等源码（含转义还原正确），右侧预览正确渲染 5 种带图标的彩色提示块（蓝/绿/主题色/橙/红）。
4. 点击「复制」按钮复制内容为原始 `> [!NOTE]...` 文本；点击「打开编辑器试试」后编辑器预填对应内容。
5. 所有卡片入场动画仍依次错峰播放（含新增与末尾卡片）。

