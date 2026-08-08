# 落地页新增「异构文件清洗转换」模块计划

## Summary

在 `landing/` 落地页中：

1. **核心特性（`#features`）** 新增第 9 张特性卡片，介绍异构文件清洗转换为 Markdown 的能力；
2. **新增** **`#file-import`** **模块**（位于 `#features` 之后），介绍异构文件清洗转换：含区块标题/描述、支持格式徽章、以及一张**内联 SVG 动效图**展示「异构文件 → 清洗转换引擎 → Markdown 笔记 / AI 助手」的完整流水线；
3. **导航栏** 增加「文件导入」链接。

## Current State Analysis

* 当前落地页为**浅色蓝主题**：`--primary: #2563EB`、`--accent: #F97316`、`--bg: #F8FAFC`，所有 section 背景统一 `var(--bg)`，卡片为白色毛玻璃（`rgba(255,255,255,0.75)`）。

* 区块顺序（`#stats` 项目亮点已移除后）：`#features` → `#screenshots` → `#videos` → `#tech-stack` → `#cta`。

* `js/main.js` 的 `.reveal` + IntersectionObserver 在 DOM 解析后统一观察所有 `.reveal` 元素，**新增静态 HTML 区块无需改 JS 即可自动获得入场动画**。

* 特性卡片现有 8 张，图标共 8 个颜色变体（默认蓝/orange/green/purple/pink/teal/cyan/red），无 amber 变体。

* 后端能力（已确认，文案以此为准）：

  * [converter.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/converter/converter.go#L24-L43) 定义办公文件扩展名：`.docx/.xlsx/.xls/.pptx/.pdf/.epub`（`.zip` 已弃用，不展示）；纯文本（`.txt/.md/.html/.csv/.json`）走二进制检测兜底。

  * markitdown 内置转换器还包括：`csv / rss(xml) / ipynb / html / plaintext`（[markitdown.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/markitdown/markitdown.go#L202-L218)）。

  * [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L3398-L3537) `ImportFiles` 批量导入拖拽文件为笔记；`ConvertToMarkdown` 转换后可存为笔记，也可供 AI 助手使用（双通道输出）。

* 历史经验（记忆）：动效图用**单一内联 SVG + CSS/SMIL 动画**，流动效果用 `stroke-dashoffset` 可靠；避免 `offset-path`（跨浏览器不可靠）；SVG 元素 transform 动画需 `transform-box: fill-box` 保证绕自身中心变换。

## Proposed Changes

### 1. `landing/index.html`

**a. 导航栏**：`nav-links` 中「特性」之后插入 `<a href="#file-import">文件导入</a>`。

**b.** **`#features`** **网格末尾**新增第 9 张特性卡片（带 `reveal` 类与递增的 transition-delay）：

* 图标：新增 `.feature-card-icon.amber` 琥珀色变体，SVG 用「文件 + 进入箭头」的导入语义图标（stroke 风格与现有卡片一致，stroke-width 1.5）；

* 标题：**异构文件导入**；

* 描述：拖入 PDF、Word、Excel、PPT 等办公文件，内置 markitdown 引擎自动清洗转换为 Markdown，一键存为笔记，或直接交给 AI 助手分析。

**c. 新增** **`#file-import`** **section**（放在 `#features` 与 `#screenshots` 之间）：

```
<section id="file-import">
  <div class="section-header reveal">
    <span class="section-label">File Import</span>
    <h2 class="section-title">异构文件清洗转换</h2>
    <p class="section-desc">…介绍文案…</p>
  </div>

  <div class="format-badges reveal">
    PDF / DOCX / XLSX / XLS / PPTX / CSV / EPUB / IPYNB / HTML / RSS / TXT·MD
  </div>

  <div class="workflow-card reveal">
    <!-- 内联 SVG 动效图 -->
  </div>

  <div class="import-points">
    3 个小特性点：一键导入 / 智能清洗 / 双通道输出
  </div>
</section>
```

**SVG 动效图设计**（`viewBox="0 0 920 340"`，`width:100%`，浅色系配色与页面一致）：

* **左（输入）**：圆角矩形虚线框内叠放 4 张小文件卡（PDF/DOCX/XLSX/PPTX），带轻微旋转扇形堆叠，框上标注「异构文件」；

* **中（引擎）**：六边形引擎（`polygon`，蓝色渐变填充）+ 外部旋转虚线环（`ellipse` stroke-dasharray + 旋转动画）+ 脉冲光晕（`circle` opacity 呼吸），下方标注「Markitdown 清洗引擎」；

* **右（输出）**：两个输出节点卡片——

  * 上：「Markdown 笔记」，3 条文本线（`rect` 用 `transform: scaleX` + `transform-box: fill-box` 伸缩生长动画）；

  * 下：「AI 助手」，3 个圆点（弹跳动画模拟输入中）；

* **连线**：左→中主管道、中→右上/右下两条分叉管道，均为贝塞尔曲线：

  * 虚线流动：`stroke-dasharray` + `stroke-dashoffset` CSS 动画；

  * 光点沿路径移动：SVG `<animateMotion>`（`mpath` 引用路径）——原生 SVG 动画，跨浏览器可靠；

* **动效数量克制**：每类动画 1-2 个关键元素（符合动画规范），全部可被现有 `prefers-reduced-motion` 全局规则关闭（CSS 动画部分）；对 SMIL 光点，在 reduced-motion 媒体查询下 `display: none` 隐藏光点，保留静态图。

**d. 新增 3 个小特性点**（`import-points`）：一键导入（拖拽批量导入，自动识别格式）/ 智能清洗（标题层级、表格、图片自动还原为 Markdown）/ 双通道输出（存为笔记进知识库 · 直接发给 AI 助手分析）。

### 2. `landing/css/style.css`

追加 `/* ========== File Import Section ========== */` 注释区块（放在 Features 区块之后）：

* `.feature-card-icon.amber` 琥珀色变体（`#F59E0B` 系渐变 + 文字色）；

* `#file-import` section：`background: var(--bg)`、`padding-top: 120px`，与其它区块一致；

* `.format-badges`：徽章行样式（复用 `.tech-badge` 视觉语言：浅色底 + 主色边框 + 圆角），支持换行居中；

* `.workflow-card`：毛玻璃卡片容器（复用 `.feature-card` 视觉：白底/圆角/阴影），内部 SVG 自适应宽度；

* SVG 内动画 keyframes：

  * `dash-flow`（stroke-dashoffset 从正值到 0 循环，虚线流动）；

  * `ring-spin`（旋转虚线环，`transform-origin` 指向环中心——用 `transform-box: fill-box` + 圆心对齐，或直接 `transform-origin: 50% 50%` 配合环的几何中心）；

  * `pulse-glow`（opacity 呼吸）；

  * `text-line-grow`（`scaleX` 生长）；

  * `dot-bounce`（AI 输出圆点弹跳）；

* 动画时长统一 2-3s 循环、`ease-in-out`，CSS 动画属性仅用 transform/opacity/stroke-dashoffset（避免重排）；

* 响应式：`@media (max-width: 768px)` 时 SVG 高度自适应（`height:auto`），徽章字号/间距缩小，`import-points` 降为单列；

* **不加** section `::after` 过渡层（当前页面已无过渡层模式，避免泛白光问题）。

### 3. `landing/js/main.js`

**无需修改**。新 `.reveal` 元素会被现有 IntersectionObserver 自动覆盖；SVG 动画为纯 CSS/SMIL。

### 4. 文档

**不更新** `LANDING_README.md` / `展示内容规范.md`（这两份文档已滞后于页面现状——`#stats` 已删但仍被记录；本次聚焦页面实现本身，避免扩大范围）。

## Assumptions & Decisions

* 新模块 id 定为 `#file-import`，位置放在 `#features` 之后（沿用此前版本「特性 → 文件导入 → 界面预览」的区块顺序，符合浏览逻辑：先讲特性总览，再深入介绍单一能力）。

* 导航栏新增「文件导入」链接（与其它区块保持一致）。

* 支持格式徽章**不包含 ZIP**（项目已弃用 ZIP 支持）。

* 动效图采用**单一内联 SVG + CSS/SMIL 动画**（沿用此前验证过的 `stroke-dashoffset` 方案，光点用 `animateMotion`，规避 `offset-path` 兼容问题）。

* 新 section 采用**浅色**（与当前全部浅色区块一致，不再做明暗交替）。

* 特性卡片从 8 张变为 9 张（桌面端 3×3 网格）。

## Verification

1. `GetDiagnostics` 检查 `index.html` / `style.css` 无语法错误；
2. 启动 `go run serve.go`（或 `python serve.py`）预览；
3. 浏览器验证：

   * 导航「文件导入」链接存在且可平滑滚动到 `#file-import`；

   * `#features` 共 9 张卡片，新卡片 hover 浮起、入场动画正常；

   * `#file-import` 标题/描述/格式徽章/动效图全部渲染，SVG 动画流畅（虚线流动、环旋转、光点移动、文本线生长）；

   * 缩放至 375px 无横向滚动，徽章换行正常，SVG 自适应；

   * 控制台无报错；
4. `grep` 新增内容无 `ZIP`/`zip` 字样残留。

