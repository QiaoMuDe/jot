# 搜索弹窗 UI 与交互动画精修 Spec

## Why

`move-search-to-modal` spec 已将笔记搜索迁移到居中弹窗，但当前实现停留在「能用的基础版」：视觉层次平淡、缺少品牌质感、过滤器和结果项视觉一致性强但缺乏温度、交互动画只是 0.2s 的 scale+fade，与全站「温暖极简」的设计语言脱节。需要一次以弹窗为单元的精修，让搜索从「可用」升级为「愉悦」。

## Design Direction: 温暖极简 · 浮层叙事

延续 `redesign-ui` 的设计系统（暖白基调 + 琥珀色点缀 + DM Sans），并叠加 `enhance-interaction-animation` 的节奏规范。本次精修以「浮层叙事」为主线：

- **浮层定位**：弹窗从屏幕顶部 12vh 处浮起，带轻微的 scale + translateY，进入有「轻盈落地」的弹簧感
- **焦点引导**：弹窗打开时输入框自动聚焦，光标带琥珀色；过滤器、结果项的入场按 30-40ms 间隔交错
- **微交互密度提升**：每一类元素（结果项 hover/select、过滤器 chip 切换、关键字高亮闪烁）都补足反馈
- **可关闭提示**：键盘提示文字在用户开始输入后淡出，避免视觉噪音

### 与既有系统的关系

| 维度 | 复用 | 新增/调整 |
|------|------|-----------|
| 颜色 | 全部使用 `:root` 中 `--accent` / `--accent-light` / `--card-bg` / `--hover-bg` / `--border` 等 token | 顶部增加 2px 琥珀色 brand-line（编辑器模态同款） |
| 字体 | 继续使用 DM Sans 体系（输入框 0.938rem，标题 0.875rem） | 关键字高亮使用 `--accent` 文字 + `--accent-light` 背景（更醒目） |
| 圆角 | 沿用 `--radius-2xl`（弹窗容器）、`--radius-md`（结果项）、`--radius-sm`（chip） | 输入容器改为左内嵌圆角 + 整体圆角 |
| 阴影 | `--shadow-xl`（弹窗容器） | 结果项 hover 阴影从 `none` 提升为 `--shadow-sm` |
| 动画 | 复用 `cubic-bezier(0.16, 1, 0.3, 1)` 弹簧曲线 | 进入时长提升到 280ms；退出保持 180ms ease-in；增加结果项 30ms 错峰 |

---

## What Changes

### 1. 弹窗容器精修
- 宽度从固定 `520px` 改为 `min(560px, calc(100vw - 48px))`，更大屏体验更舒展
- 顶部增加 2px 琥珀色装饰条（与编辑模态一致，强化品牌识别）
- 容器圆角从 `--radius-2xl`（推测 20px）调整为 `20px`，与编辑器模态对齐
- 弹窗遮罩（mask）从 `rgba(0,0,0,0.4)` 微调为 `rgba(45, 42, 36, 0.32)`（暖色调遮罩，与全站暖色基调一致）
- 遮罩 `backdrop-filter: blur(4px)` 保留，但增加 `-webkit-backdrop-filter` 已存在，确保 Wails WebView 兼容

### 2. 头部（header）重设计
- 图标从「线性放大镜」改为「圆形浅色背景 + 线性放大镜」组合（搜索图标 16px 居中于 28×28 圆角矩形，`--accent-light` 背景 + `--accent` 描边色）
- 输入框 `font-size` 从 `0.938rem` 提升至 `1rem`，字重 500；placeholder 文字保持暖灰但更精致（`--text-muted`）
- 右侧「Esc 关闭 · ↑↓ 选择 · Enter 打开」改为三枚 chip 形式（`<kbd>` 风格小标签），使用 `--input-bg` 背景 + `--text-muted` 文字，间距 6px
- 当输入框聚焦或已有内容时，提示 chip 淡出（opacity 0.3），避免拥挤
- header 与内容区的分割线保留 `1px solid var(--border)`

### 3. 过滤器（filters）重设计
- 过滤器整行容器背景从透明改为 `--input-bg`（极淡暖白），与 header 形成层次
- 「笔记本:」「标签:」「时间:」三个 label 字号从 `0.75rem` 提升到 `0.75rem`（保持），字重从 400 提升到 500，颜色从 `--text-muted` 调整为 `--text-secondary`（更可读）
- filter 按钮：未激活态使用 `--card-bg` 背景 + `--border` 边框；激活态使用 `--accent-light` 背景 + `--accent` 文字 + `--accent` 边框（带轻微 scale 0.98 点击反馈）
- filter 按钮右侧加 chevron-down 图标（4px 大小 SVG），激活后旋转 180°（200ms）
- 下拉菜单：进入动画从 `display: block` 改为 opacity + translateY(-4px) → 0，160ms 弹簧；菜单项 hover 背景从 `--hover-bg` 调整为暖色更显眼的 `--accent-light`（带 100ms 过渡）
- 已选项使用 ✓ 图标 + `--accent` 文字双重指示（不仅靠颜色）

### 4. 结果列表（results）重设计
- 列表区增加 `padding: 8px 12px` 让结果项之间有呼吸感
- 结果项布局：标题行（左侧标题 + 右侧时间戳 12px 暖灰），第二行摘要，第三行 meta（笔记本名 · 标签 chip）
- 结果项 hover：背景从 `--hover-bg` 调整为更柔和的渐变（`linear-gradient(90deg, var(--hover-bg), transparent)` 200ms 过渡），左边框从 2px transparent 变为 2px `--accent`（200ms 渐显）
- 结果项 selected：左边框琥珀色 + 背景 `--accent-light`（30% 透明度）+ 标题字重从 500 提升到 600，200ms 过渡
- 结果项交错入场：每项延迟 30ms，opacity + translateY(8px) → 0，单项 200ms cubic-bezier
- 笔记本名前加 SVG 笔记本小图标（10px），标签 chip 保留但增加细微边框（`1px solid currentColor` opacity 0.2）
- 关键字高亮（`<mark>`）从 `rgba(255, 213, 79, 0.45)` 调整为更精致的 `--accent-light` 实色背景 + 文字下方 1px `--accent` 边线，padding 0 3px，圆角 2px

### 5. 空状态（empty）重设计
- 增加圆形背景图标（`--accent-light` 圆 + `--accent` 搜索图标），64px 直径
- 主标题「开始搜索你的笔记」+ 副标题「输入关键字搜索标题、内容或标签」（字号 0.875rem + 0.75rem）
- 无结果时主标题改为「没有找到匹配的笔记」+ 副标题「试试调整过滤器或换个关键词」

### 6. 底部（footer）精修
- footer 从纯文字「共 N 条结果」改为「共 N 条 · ⏎ 打开」组合（右侧键盘提示）
- 当结果数 > 0 时显示，否则隐藏
- footer 顶部增加极细分割线（`1px solid var(--border)` 50% 透明度）

### 7. 交互动画系统化
- **打开弹窗**（用户触发 Ctrl+F）：
  - 容器：`scale(0.96) → scale(1)` + `translateY(-8px → 0)` + `opacity 0 → 1`，280ms `cubic-bezier(0.16, 1, 0.3, 1)`
  - 头部元素依次淡入（图标 → 输入框 → 提示），每项 30ms 错峰
  - 过滤器 chip 依次淡入，每项 40ms 错峰
- **关闭弹窗**：
  - 容器：`scale(1) → scale(0.97)` + `translateY(0 → -4px)` + `opacity 1 → 0`，180ms `ease-in`
  - 内部元素不单独做退出动画（避免视觉噪音）
- **结果项入场**（搜索完成时）：
  - 第一页 18 条以内：每项 30ms 错峰，opacity + translateY(8px → 0)
  - 加载下一页：仅新追加的项做入场动画（前次已加载的不重复）
- **过滤器下拉打开**：opacity 0 → 1 + translateY(-4px → 0)，160ms
- **过滤器下拉关闭**：opacity 1 → 0，120ms
- **键盘导航**：selectedIndex 切换时，结果项滚动到可视区域（如果被遮挡），并使用 `scrollIntoView({ block: 'nearest' })` 不强制居中
- **`prefers-reduced-motion: reduce`** 媒体查询：禁用所有 transform 动画，仅保留 opacity 渐变

### 8. 焦点与可访问性
- 弹窗打开后 `aria-modal="true"` 已有，补充 `aria-labelledby` 关联标题（如果无标题，aria-label 已存在）
- 输入框聚焦时无可见 focus ring（因整个弹窗视觉上就是焦点），但增加输入框下方 1px 琥珀色下划线（`border-bottom: 1px solid var(--accent)`），250ms 渐显
- 过滤器按钮、结果项都需 `cursor: pointer`
- 键盘 Esc 关闭行为不变（已有）
- 弹窗打开时记录 `document.activeElement`，关闭时恢复（已有 `_searchModalPrevFocus`）

### 9. 兼容性
- 保留所有现有 DOM id（`#searchModal`、`.search-modal-content`、`.search-modal-header` 等），不破坏 main.js 选择器
- 保留所有现有 JS 行为（防抖 200ms、过滤、分页、键盘导航、Esc 关闭等），仅微调时序
- 新增 CSS 变量与现有 token 复用，不引入新 SCSS/CSS 预处理器
- 适配 `prefers-color-scheme: dark`（在 dark theme 下已通过 CSS 变量自动适配，无需额外处理）

---

## Impact

- **Affected specs**:
  - `move-search-to-modal`：本 spec 是其精修版，不破坏基础能力
  - `redesign-ui`：本 spec 完全沿用其设计 token，不修改基础变量
  - `enhance-interaction-animation`：复用其动画时长/缓动规范
  - `add-search-filters`：过滤器视觉精修，行为不变
- **Affected code**:
  - `frontend/src/style.css` — 仅修改 `.search-modal*` 相关样式（约 250 行），不触碰其他选择器
  - `frontend/index.html` — 仅在 `<div class="search-modal-header">` 内部追加 1-2 个 chip 元素（kbd 提示）；不修改其他结构
  - `frontend/src/main.js` — 仅在 `openSearchModal` / `closeSearchModal` 中追加 chip 显隐逻辑（1-2 行），不修改主流程
  - 不涉及：后端 Go、状态管理、API 调用

---

## ADDED Requirements

### Requirement: 弹窗品牌色装饰条
弹窗 SHALL 在容器顶部展示 2px 琥珀色 brand-line，强化品牌识别。

#### Scenario: 弹窗打开
- **GIVEN** 用户触发搜索弹窗
- **WHEN** 弹窗从隐藏到可见
- **THEN** 顶部 2px 琥珀色装饰条在容器内显示（与编辑模态一致）
- **AND** 装饰条不参与悬停/点击事件（`pointer-events: none`）

### Requirement: 输入框焦点视觉
弹窗 SHALL 在输入框聚焦时显示琥珀色底部下划线。

#### Scenario: 输入框聚焦
- **GIVEN** 弹窗已打开且输入框为空
- **WHEN** 用户点击输入框或自动聚焦
- **THEN** 输入框底部 1px 琥珀色下划线在 250ms 内渐显
- **AND** 失焦时下划线渐隐回透明

### Requirement: 过滤器激活态
弹窗 SHALL 用「背景 + 文字 + 边框 + chevron 旋转 + ✓ 图标」多重视觉指示激活过滤器。

#### Scenario: 过滤器激活
- **GIVEN** 弹窗已打开
- **WHEN** 用户选择某个过滤器选项
- **THEN** 过滤器按钮背景变为 `--accent-light`
- **AND** 文字颜色变为 `--accent`
- **AND** 边框变为 `--accent`
- **AND** chevron 图标旋转 180°

### Requirement: 结果项键盘选中反馈
弹窗 SHALL 在键盘导航时高亮当前 selectedIndex 项目，并滚动到可视区域。

#### Scenario: 按下 ↓
- **GIVEN** 弹窗已打开且结果列表有 ≥1 项
- **WHEN** 用户按下 ↓ 键
- **THEN** 当前 selectedIndex 项目获得选中态样式（背景 `--accent-light` + 左边框 `--accent`）
- **AND** 之前选中的项目移除选中态
- **AND** 新选中项目使用 `scrollIntoView({ block: 'nearest', behavior: 'smooth' })` 滚动到可视区域
- **AND** 如已在可视区域内，不滚动（避免抖动）

### Requirement: 结果项交错入场
弹窗 SHALL 在结果首次加载或加载更多时为新项目做交错入场动画。

#### Scenario: 首次搜索完成
- **GIVEN** 用户在弹窗中输入关键字
- **WHEN** 防抖后第一页结果到达
- **THEN** 每个结果项按 30ms 错峰执行 `opacity 0 → 1` + `translateY(8px → 0)`，单项 200ms `cubic-bezier(0.16, 1, 0.3, 1)`
- **AND** 第一页最多 18 项参与错峰，超出项同时入场（避免动画时间过长）

#### Scenario: 加载更多
- **GIVEN** 用户滚动到结果列表底部附近
- **WHEN** 加载下一页结果到达
- **THEN** 仅新追加的项做入场动画（每项 30ms 错峰，单项 180ms）
- **AND** 已存在的项保持原样，无动画

### Requirement: 键盘提示 chip 自动隐藏
弹窗 SHALL 在用户开始输入后淡化键盘提示 chip。

#### Scenario: 输入框无内容
- **GIVEN** 弹窗刚打开
- **WHEN** 输入框为空
- **THEN** 头部右侧三个键盘提示 chip 完全可见（opacity 1）

#### Scenario: 输入框有内容
- **GIVEN** 弹窗已打开
- **WHEN** 输入框 value 不为空
- **THEN** 头部三个 chip 淡出到 opacity 0.3（200ms）
- **AND** 输入框清空后 chip 恢复 opacity 1

### Requirement: 减动效偏好支持
弹窗 SHALL 尊重 `prefers-reduced-motion: reduce` 媒体查询。

#### Scenario: 用户启用减动效
- **GIVEN** 操作系统/浏览器启用了减动效偏好
- **WHEN** 弹窗打开/关闭或结果入场
- **THEN** 所有 transform 动画被替换为 opacity 渐变
- **AND** 错峰动画被禁用（所有项同时显示）
- **AND** 切换时间缩短到 100ms

---

## MODIFIED Requirements

### Requirement: 弹窗容器尺寸（原 move-search-to-modal）
- **旧**：`width: 520px`
- **新**：`width: min(560px, calc(100vw - 48px))`

### Requirement: 弹窗遮罩颜色（原 move-search-to-modal）
- **旧**：`background: rgba(0, 0, 0, 0.4)`
- **新**：`background: rgba(45, 42, 36, 0.32)`（暖色基调）

### Requirement: 弹窗打开动画（原 move-search-to-modal）
- **旧**：容器 `transform: scale(0.96) translateY(-8px) → scale(1) translateY(0)`，200ms `cubic-bezier(0.16, 1, 0.3, 1)`
- **新**：同上曲线，时长提升到 **280ms**；内部元素（header 部件、过滤器 chip）按 30-40ms 错峰淡入

### Requirement: 弹窗关闭动画（原 move-search-to-modal）
- **旧**：容器 `opacity 1 → 0` + 容器 transform 反向，200ms
- **新**：同上，**时长缩短到 180ms**（退出更快原则），`ease-in` 缓动；内部元素不做单独动画

### Requirement: 头部输入框字号（原 move-search-to-modal）
- **旧**：`font-size: 0.938rem`，字重 inherit（400）
- **新**：`font-size: 1rem`，字重 500

### Requirement: 头部右侧提示（原 move-search-to-modal）
- **旧**：`<span class="search-modal-hint">Esc 关闭 · ↑↓ 选择 · Enter 打开</span>`，纯文字 0.75rem
- **新**：改为 3 枚 chip 形式（`<kbd>` 风格），使用 `<span class="search-modal-kbd">` 包裹
  - chip 样式：`background: var(--input-bg); color: var(--text-muted); border: 1px solid var(--border); border-radius: 4px; padding: 1px 6px; font-size: 0.688rem; font-family: ui-monospace, monospace;`
  - chip 间用 `·` 分隔（仍为文本节点，不参与 chip 样式）

### Requirement: 关键字高亮（原 move-search-to-modal）
- **旧**：`mark { background: rgba(255, 213, 79, 0.45); color: inherit; padding: 0 1px; border-radius: 2px; }`
- **新**：`mark { background: var(--accent-light); color: var(--accent); font-weight: 600; padding: 0 3px; border-radius: 2px; border-bottom: 1px solid var(--accent); }`

### Requirement: 过滤器按钮激活态（原 move-search-to-modal / add-search-filters）
- **旧**：激活态无明显视觉指示（仅靠 dropdown 内部 `selected` 类）
- **新**：filter 按钮激活后使用「背景 + 文字 + 边框 + chevron 旋转」多重视觉指示
  - 背景：`--accent-light`
  - 文字：`--accent`，字重 500
  - 边框：`--accent`（替换 `--border`）
  - chevron：旋转 180°（200ms 过渡）

---

## REMOVED Requirements

无功能被移除。所有改动均为视觉与动画精修，行为 API 保持向后兼容。
