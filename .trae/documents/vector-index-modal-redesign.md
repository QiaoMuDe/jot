# 笔记量化弹窗 UI 重构计划

## Summary

重构数据管理模块「笔记量化」（向量索引）弹窗的视觉样式与交互动画：保留全部 JS 依赖的 ID、DOM 结构与事件绑定，重写 CSS 样式层、微调 HTML 结构与少量 JS 动画增强，完全融入 Jot 现有 14 主题设计体系（CSS 变量 + token，不引入外部字体/新色板）。

## 现状分析

**HTML**（[index.html L1951-L2025](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1951-L2025)）：`#vectorIndexModal` → overlay + panel；panel = header（标题+关闭）+ 两个视图（`#vectorIndexSelectView` 选择 / `#vectorIndexProgressView` 进度）。

**JS**（[data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L517-L1022)）：open/close/setVectorIndexView（display 硬切）/switchVectorIndexScope/懒绑定事件/render 列表/startVectorIndex（后端三方法按范围分发）/updateVectorIndexProgress（双进度条+250ms 块级延迟清零）/showSummary/showError。

**CSS**（[data-view.css L363-L715](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/data-view.css#L363-L715)）：43 条 vector-index 规则，基础可用但无个性。

**现有问题（本次要解决的）**：

1. 头部纯文字标题，无视觉锚点
2. 范围分段控件：无滑动指示条，active 只换色/背景
3. checkbox 用原生 `accent-color`，朴素
4. 列表项 hover 只有背景色，勾选态无强调
5. select↔progress 视图切换 `display:none` 硬切，无过渡
6. 进度条纯色渐变填充，无动效
7. summary/error 直接显示，无入场动画

## 变更内容

### 1. index.html — 结构微调（保留全部 ID）

* header 中标题前加装饰 SVG（向量/连接点语义，16px，`--accent` 色，非 emoji）

* 两个 view 容器保持不变（视图切换动画由 CSS + JS class 驱动，不改 display 逻辑）

* **不删除、不重命名任何现有 ID/class 依赖**（JS 引用清单：vectorIndexModal/Overlay/Close/DoneBtn/SelectView/ProgressView/ScopeSeg 下全部 .scope-btn/NotebookPicker/NotePicker/NotebookSearch/NoteSearch/NotebookSelectAll/NoteSelectAll/NotebookList/NoteList/Count/StartBtn/ProgressFill/ProgressPercent/ProgressStage/ChunkFill/ChunkPercent/ChunkStage/CurrentTitle/Summary/Error）

### 2. data-view\.css — 重写 vector-index 区块（核心）

全部沿用现有 token：`--card-bg`/`--radius-*`/`--shadow-*`/`--accent`/`--border`/`--divider`/`--hover-bg`/`--text-*`/`--success-*`/`--danger-*`/`--transition`/`--anim-easing-spring`。

* **面板入场**：`modalEnter` 增强为 scale(0.96)+translateY(12px)+fade，spring 曲线

* **头部**：图标 + 标题排版，`border-bottom` 改为 `linear-gradient` 细线；关闭按钮 hover 态

* **分段控件**：新增 `.vector-index-scope-indicator` 滑动指示条（绝对定位，JS 计算 left/width 并过渡移动），active 按钮文字色 accent + 指示条高亮；整体改为内凹容器

* **自定义 checkbox**：原生 input `appearance:none` 掩藏，伪元素画边框，勾选时 `stroke-dashoffset` 对勾描边动画 + 背景填充 accent；indeterminate 半选态画横线

* **列表项**：hover 左缘 2px accent 指示 + 背景；勾选态背景 tint（`color-mix` accent 8%）；标题区 + 计数徽标

* **视图切换动画**：`.vector-index-view` 进出场 class（`view-leave`/`view-enter`），transform: translateX(±14px) + opacity，200ms ease-out；JS 在 setVectorIndexView 中先加 leave class、`transitionend`/定时后切 display、再加 enter class

* **进度条**：填充色 accent 渐变 + `background-size` 条纹流光（`background-position` 位移，仅 width>0 时动画）；百分比 `font-variant-numeric: tabular-nums`

* **完成态**：进度条 fill 切成功色 + 一次脉冲（scaleY 闪烁）；summary 卡片入场上浮动画 + 左侧成功图标

* **错误态**：error 卡片入场 + 轻微 shake；保留 `--danger-*` 语义色

* **计数徽标**：已选 N 篇/个笔记本做成 pill 徽标

* **`prefers-reduced-motion: reduce`**：关闭全部动画（入场/指示条/进度流光/切换/脉冲）

### 3. data-management.js — 最小增强（不动业务逻辑）

* 新增 `repositionVectorIndexScopeIndicator()`：按 active 按钮计算 left/width，`openVectorIndexModal` 打开时与 `switchVectorIndexScope` 切换后调用（含 `requestAnimationFrame` 保证布局稳定）

* `setVectorIndexView(view)`：改为动画切换——旧视图加 `view-leave`，约 180ms 后切 display 并给新视图加 `view-enter`（清理动画 class 防泄漏）；逻辑等价，仅增强过渡

* 其余函数（渲染/计数/进度/summary/error/懒绑定）**零改动**——所有动画均通过 class 与 CSS 驱动

### 不动项

* 后端（app.go 三方法、事件 payload 结构）零改动

* 事件名 `vector:index-progress/done/error`、`cleanupVectorIndexEvents` 零改动

* `startVectorIndex` 的禁用态/错误提示逻辑零改动

## 假设与决策

* **风格方向**：精致扁平 + 细腻微动效（ui-ux-pro-max Flat Design 建议 + 项目现有设计语言），**不引入外部字体/新色板**，14 主题通过 token 自动适配

* **JS 改动最小化**：仅新增指示条定位函数与视图切换动画 class，不触碰事件绑定与业务逻辑

* **checkbox 自定义**：需兼顾 14 主题对比度（用语义 token + color-mix，不用硬编码色）

* 动画时长遵守项目规范（入场 ≤300ms、退场更短、transform/opacity 驱动、尊重 reduced-motion）

## 验证步骤

1. `npm run build` 通过（前端构建）
2. `wails dev` 手动验证：

   * 弹窗入场动画、头部图标

   * 范围分段滑动指示条（三档切换）

   * 笔记本/笔记列表勾选对勾动画、全选半选态

   * select→progress 视图切换滑动过渡

   * 量化过程中双进度条流光动画、块级 250ms 延迟清零仍正常

   * 完成摘要入场 + 进度条成功色脉冲；错误态 shake

   * 量化进行中关闭被拦截（`vectorIndexRunning` 逻辑不受影响）
3. 抽查浅色/深色主题各 2-3 套：对比度、选中态、指示条可见性
4. 系统开启减少动态效果后：全部动画关闭，功能可用

