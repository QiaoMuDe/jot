# TOC 侧栏重新设计方案

## 摘要

重新设计 Markdown 预览的大纲侧栏（TOC Sidebar），使其视觉风格与应用的暖色调、纸张质感美学一致，并修复 CSS `!important` 规则导致的潜在显示问题。所有功能逻辑不变。

## 当前问题

1. **视觉平庸**：侧栏样式缺乏设计感，与整体应用的精致调性不匹配

   * 颜色平淡，缺乏层次

   * 层级缩进仅用 `padding-left`，视觉区分度不够

   * 活跃态指示条简单，缺少细节

   * 折叠态交互粗糙
2. **CSS 隐患**：`.editor-overlay[data-mode="preview"] .toc-sidebar { display: flex !important; }` 会覆盖 JS 的 `display:none` 设置，非 `.md` 文件在异常路径下可能错误显示 TOC
3. **滚动高亮不够直观**：当前高亮项滚动到可视区域的逻辑有缺陷

## 设计方案

### 设计方向：暖调纸张质感 + 精致细节

* **色调**：融入现有暖色调（`#F7F5F0` 底色、`#D97706` 琥珀强调色），侧栏背景使用比现有 `--bg-secondary` 略暖的色调

* **字体**：使用与正文一致的字体家族，标题项字号微调，层级用灰色渐变区分

* **层级视觉**：H1 用左侧竖线 + 加粗，H2 缩进 + 常规，H3+ 逐级缩进 + 更浅色，形成清晰的视觉阶梯

* **活跃态**：左侧 3px 圆角指示条 + 背景柔和高亮，带微过渡动画

* **折叠态**：重新设计折叠/展开交互，折叠后显示竖排"大纲"文字 + 展开箭头

* **空状态**：半透明斜体提示"暂无标题"，带小图标（SVG）

* **滚动条**：与 TOC 容器融合，极窄设计

### 改动范围

**只修改 CSS**，不修改 JS 逻辑和 HTML 结构。具体的：

1. **`frontend/src/css/components/editor.css`**

   * 重写全部 TOC 相关样式（第 697-869 行）

   * 删除 `!important` 规则，改用正确的 CSS 特异性控制显示

   * 新增视觉细节：指示条动画、渐变背景、精致边框

   * 优化折叠态样式

2. **`frontend/src/main.js`**（如有必要）

   * 移除 `_integrateToc()` 中对 `tocSidebar.style.display` 的显式设置

   * 改为依赖 CSS class 控制显隐，避免与 `!important` 冲突

## 具体改动

### 1. 修复 CSS 显示逻辑

删除：`.editor-overlay[data-mode="preview"] .toc-sidebar { display: flex !important; }`
新增控制逻辑：

* TOC 侧栏默认 `display: none`

* 通过 JS 添加/移除 `.toc-visible` class 来控制显示

* `.toc-sidebar.toc-visible { display: flex; }`

* 在 `_integrateToc()` 和 `updatePreview()` 中切换 `.toc-visible` class 而非内联 `style.display`

这消除了 `!important` 隐患。

### 2. 重新设计侧栏视觉

#### `.toc-sidebar` 主体

* 宽度 220px（比当前 200px 稍宽，更舒适）

* 背景：`var(--card-bg)` 而非 `var(--bg-secondary)`，更干净

* 右侧分割线：渐变（从实色到透明）

* 顶部与 `md-rendered` 对齐（去掉 `border-bottom` 略显突兀的问题）

#### `.toc-header`

* 添加微妙的底部阴影代替直接 border-bottom

* 标题"大纲"前加一个小图标（SVG 列表符号）

* 字体略大、字距更舒适

#### `.toc-item` 层级系统

* **depth-1（H1）**：左侧 3px 竖线（`--accent` 30% 透明度），加粗，无缩进

* **depth-2（H2）**：左侧 2px 竖线（`--accent` 20% 透明度），常规字重，24px 缩进

* **depth-3（H3）**：左侧 1px 竖线（`--accent` 10% 透明度），小字号，36px 缩进

* **depth-4+（H4+）**：无竖线，更小字号，48px 缩进

* hover 态：背景渐入（`--hover-bg`），文字颜色平滑过渡（0.15s）

* active 态：左侧活跃指示条变为 `--accent` 实色 3px，背景 `color-mix(in srgb, var(--accent) 10%, transparent)`

#### `.toc-item` 文字溢出

* 保持 `text-overflow: ellipsis`，但添加 `title` 属性（JS 生成时已带完整文本）

#### 折叠态

* 折叠后宽度从 220px → 36px

* 折叠按钮文本改为竖排"大纲" + "›" 图标

* 展开动画：`transition: width 0.3s cubic-bezier(0.16, 1, 0.3, 1)`（匹配编辑器的 spring 曲线）

#### 空状态

* 灰色小图标（heading SVG）+ 文字"暂无标题"

* 半透明，居中对齐

#### 滚动条

* 极窄 3px

* 悬停时才显示（auto-hide）

* 使用 `var(--scrollbar-thumb)` 配色

### 3. 更新 JS（最小修改）

* `_integrateToc()`：替换 `els.tocSidebar.style.display = ''` 为 `els.tocSidebar.classList.add('toc-visible')`，替换 `els.tocSidebar.style.display = 'none'` 为 `els.tocSidebar.classList.remove('toc-visible')`

* `updatePreview()` 空内容路径：同样替换为 class 控制

* `closeEditor()`：替换为 class 控制

* `switchEditorReadOnly()` 非 .md 路径：替换为 class 控制

### 不影响的功能

* 滚动高亮逻辑（`_updateTocScrollHighlight`）不变

* 折叠/展开逻辑（`_initTocCollapse`）不变

* 标题提取逻辑（Worker/主线程）不变

* 点击跳转逻辑（`_renderToc`）不变

* 所有功能集成点（`openEditor`, `switchEditorMode`, `switchEditorReadOnly`, `updatePreview`）不变

## 验证清单

* [ ] TOC 侧栏在预览模式下正常显示，样式美观

* [ ] 深色/浅色所有 12 个主题下颜色正确

* [ ] 非 `.md` 文件在预览模式下绝不显示 TOC

* [ ] 层级缩进视觉清晰（H1\~H6 可区分）

* [ ] hover/active 状态过渡平滑

* [ ] 折叠/展开动画流畅

* [ ] 空文档显示正确的空状态提示

* [ ] 滚动高亮正常

* [ ] 点击跳转正常

* [ ] 全屏模式下正常工作

* [ ] 无 `!important` 规则残留

