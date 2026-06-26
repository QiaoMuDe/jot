# 重新设计 MD 语法手册页面

## Why

当前 MD 语法页面设计平庸：简单垂直堆叠的卡片 + 两栏分割面板，缺乏视觉层次和设计个性。作为应用中展示 Markdown 排版能力的"旗舰页面"，它本应体现 Jot 对美感和排版质量的追求，但目前的设计与这一目标不匹配。

## 设计方向

**设计风格**："Developer Notebook"——将 MD 语法页面设计为一种「开发者笔记杂志」风格的参考页面。每张卡片是一则"语法故事"，具备独立的视觉身份。

**关键词**：编辑式/杂志感排版、代码即视觉元素、精致克制、层次分明

### 视觉语言

| 维度 | 选择 | 原因 |
|------|------|------|
| 布局 | **Bento Grid**（响应式 CSS Grid，卡片变宽变高） | 打破单调的垂直堆叠，不同语法类别有不同的视觉权重 |
| 源码面板 | **代码编辑器窗口**风格（macOS 交通灯圆点 + 标题栏 + 文件名标签） | 让源码块本身成为视觉亮点，暗示"这是可编辑的" |
| 预览面板 | **文档卡片**风格（浅色背景、微投影、文字排版精致） | 模拟渲染后的真实文档效果 |
| 分类标签 | **图标+文字**组合，每类不同配色 | 提高可扫描性，视觉差异化 |
| 动效 | **IntersectionObserver 触发**交错入场 + hover 微动效 | 滚动才触发，避免一次性全部涌入的性能和视觉负担 |
| 「试试」按钮 | **内联融合式**按钮（微箭头动画 + 悬停填充效果） | 比现有的独立 outline 按钮更精致 |

## What Changes

### HTML 结构变更（`index.html`）

- **整体替换** `#viewMdRef` 内部结构：
  - 保留 `view-header`（返回按钮 + 标题「MD 语法」）
  - 移除旧的 `.md-ref-content` 中简单垂直卡片布局
  - 新的 `.md-ref-content` 使用 CSS Grid `grid-template-columns: repeat(auto-fill, minmax(480px, 1fr))` + 部分卡片 `grid-column: span 2`
- **每张卡片的 HTML 结构**改为：
  ```
  .md-ref-card
    .md-ref-card-header
      .md-ref-badge         ← 图标 + 文字（如 "⌗ 标题" "𝐁 文本样式"）
    .md-ref-card-body
      .md-ref-source-panel  ← 编辑器窗口风格
        .md-ref-editor-bar  ← 交通灯圆点 + 文件名
        pre > code
          button.md-ref-copy-btn
      .md-ref-preview-panel ← 文档卡片风格
        .md-ref-preview
    .md-ref-card-footer
      .md-ref-card-footnote ← 脚注说明
      button.md-ref-try-btn ← 重新设计的按钮
  ```
- **10 张卡片的内容**保留原有的 10 个语法类别，但 HTML 结构完全替换

### CSS 变更（`style.css`）

- **移除**所有旧的 `.md-ref-*` 样式（约 300 行）
- **新增**完整的新样式体系（估计 ~500 行）：
  - Bento grid 布局样式
  - 编辑器窗口风格的源码面板（交通灯圆点、标题栏、阴影）
  - 文档卡片风格的预览面板
  - 新的卡片入场动画（IntersectionObserver + CSS transition）
  - 新的 hover 微动效
  - 重新设计的 badge 样式
  - 重新设计的复制按钮（集成到标题栏）
  - 重新设计的「试试」按钮
- **全部使用 CSS 变量**，确保 6 套主题兼容

### JavaScript 变更（`main.js`）

- **替换** `renderMdRefCards()` 函数：
  - 不再使用 `_mdRefRendered` 标志（之前只渲染一次，改为每次进入都重新渲染）
  - 新增 `IntersectionObserver` 实现卡片滚动入场的交错动画
  - 保留 `marked.parse()` + `hljs.highlightElement()` 渲染逻辑
- **调整** `setupRefCopyButtons()`：适配新的编辑器窗口结构（复制按钮在标题栏中）
- **调整** `setupMdRefTryButtons()`：适配新的卡片结构
- **保留** `openMdRefTryEditor()` 函数不改

### 移除的内容（REMOVED）

### Requirement: 旧卡片布局
**Reason**: 旧的 `.md-ref-card` 使用简单的 flex 垂直排列 + 双栏分割面板，设计平庸
**Migration**: 所有 10 张卡片使用新的 bento grid 布局重新实现

### Requirement: 旧 badge 样式
**Reason**: 旧的 `.md-ref-badge` 只是简单的文本+背景色
**Migration**: 新的 badge 使用图标+文字组合，每类不同配色

### Requirement: 旧复制按钮样式
**Reason**: 旧的复制按钮是独立的浮动按钮，hover 才显示
**Migration**: 复制按钮集成到编辑器标题栏中

### Requirement: 旧「试试」按钮样式  
**Reason**: 旧的按钮是简单的 outline 样式
**Migration**: 重新设计为内联融合样式，带箭头动画

## 设计变更细节

### 1. Bento Grid 布局

```css
.md-ref-content {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(480px, 100%), 1fr));
    gap: 20px;
    padding: 28px 32px 40px;
}

/* 特定卡片跨列（内容较多的语法类别） */
.md-ref-card:nth-child(5)  /* 代码块 */
    grid-column: span 2;
```

布局原则：
- 默认单列宽（~480px），部分内容多的卡片跨 2 列
- 窄屏幕自动降为单列
- 卡片高度由内容决定，不强制统一高度

### 2. 编辑器窗口风格源码面板

源码面板顶部加入「标题栏」：

```
┌──────────────────────────────┐
│  ● ● ●    syntax.md         │ ← 交通灯 + 文件名 + 复制按钮
├──────────────────────────────┤
│  # Heading 1                 │ ← 源码内容
│  ## Heading 2                │
│  ...                         │
└──────────────────────────────┘
```

```css
.md-ref-editor-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--bg-tertiary, var(--bg-secondary));
    border-radius: 8px 8px 0 0;
    border-bottom: 1px solid var(--border);
}

.md-ref-editor-dots {
    display: flex;
    gap: 6px;
}

.md-ref-editor-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
}

.md-ref-editor-dot.red    { background: #FF5F57; }
.md-ref-editor-dot.yellow { background: #FEBC2E; }
.md-ref-editor-dot.green  { background: #28C840; }

.md-ref-editor-filename {
    flex: 1;
    font-size: 12px;
    color: var(--text-muted);
    text-align: center;
}
```

复制按钮放在标题栏右侧：

```css
.md-ref-editor-copy-btn {
    /* 在标题栏右侧，始终可见（不依赖 hover） */
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 12px;
    padding: 2px 6px;
    border-radius: 4px;
    transition: all 0.15s ease;
}
.md-ref-editor-copy-btn:hover {
    background: var(--hover-bg);
    color: var(--text-primary);
}
```

### 3. 文档卡片风格预览面板

预览面板设计为模拟文档卡片：

```css
.md-ref-preview-panel {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 0 0 8px 8px;
    padding: 20px 24px;
    min-height: 100px;
    line-height: 1.7;
}
```

### 4. 分类标签（Badge）

每类 badge 使用不同的配色（基于 CSS 变量，但在不同色相上微调）：

```css
.md-ref-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    padding: 4px 12px;
    border-radius: 6px;
    letter-spacing: 0.3px;
    background: rgba(var(--accent-rgb), 0.1);
    color: var(--accent);
}
```

### 5. 交互动效

- **卡片滚动入场**：IntersectionObserver 检测卡片进入视口，添加 `.visible` class 触发 `opacity: 0→1` + `translateY(20px→0)` 过渡，每张卡片 `transition-delay` 递增 60ms
- **卡片 hover**：`transform: translateY(-2px)` + `box-shadow` 加深，过渡 0.25s
- **按钮 hover**：箭头 `→` 向右移动 3px，背景填充
- **复制反馈**："已复制 ✓" 显示 + 绿色闪烁，500ms 后恢复

### 6. 「打开编辑器试试」按钮（重新设计）

```css
.md-ref-try-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-top: 12px;
    padding: 6px 14px;
    background: transparent;
    color: var(--accent);
    border: 1px solid transparent;
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: inherit;
}
.md-ref-try-btn:hover {
    background: rgba(var(--accent-rgb), 0.08);
    border-color: rgba(var(--accent-rgb), 0.25);
}
.md-ref-try-btn::after {
    content: '→';
    transition: transform 0.2s ease;
}
.md-ref-try-btn:hover::after {
    transform: translateX(3px);
}
```

## Impact

- Affected specs: `add-md-reference-page/`（替代原实现）、`add-md-ref-try-button/`（保留逻辑，适配新结构）
- Affected code:
  - `frontend/index.html` — 替换 `#viewMdRef` 内部所有卡片 HTML 结构
  - `frontend/src/style.css` — 替换所有 `.md-ref-*` 样式
  - `frontend/src/main.js` — 重写 `renderMdRefCards()`，调整 `setupRefCopyButtons()` / `setupMdRefTryButtons()`

## ADDED Requirements

### Requirement: Bento Grid 响应式布局
The system SHALL use CSS Grid 实现卡片布局，支持响应式列数变化

#### Scenario: 宽屏显示 (>1200px)
- **WHEN** 页面宽度 > 1200px
- **THEN** 部分内容丰富的卡片（如代码块/表格卡片）跨 2 列显示

#### Scenario: 窄屏显示 (<768px)
- **WHEN** 页面宽度 < 768px
- **THEN** 所有卡片均为单列，取消跨列

### Requirement: 编辑器窗口风格的源码面板
The system SHALL 为每个源码块添加 macOS 风格的窗口标题栏

#### Scenario: 源码展示
- **WHEN** 卡片渲染时
- **THEN** 源码块顶部显示交通灯圆点（红/黄/绿）和文件名标签

### Requirement: IntersectionObserver 滚动动画
The system SHALL 使用 IntersectionObserver 实现卡片的滚动入场动画

#### Scenario: 滚动入视口
- **WHEN** 卡片进入视口 100px 阈值
- **THEN** 卡片以 `opacity: 0→1` + `translateY(20px→0)` 动画入场，后续卡片延迟 60ms 递增

### Requirement: 卡片 hover 微动效
The system SHALL 为卡片添加 hover 上浮效果

#### Scenario: 鼠标悬停
- **WHEN** 鼠标悬停在卡片上
- **THEN** 卡片上移 2px，阴影加深

### Requirement: 重新设计的「试试」按钮
The system SHALL 使用带箭头动画的新按钮样式

#### Scenario: 按钮交互
- **WHEN** 鼠标悬停在「打开编辑器试试」按钮上
- **THEN** → 箭头右移 3px，按钮背景轻微填充

### Requirement: 复制按钮集成到标题栏
The system SHALL 将复制按钮从浮动位置移到编辑器标题栏右侧

#### Scenario: 复制操作
- **WHEN** 用户点击标题栏的复制按钮
- **THEN** 复制源码文本到剪贴板，按钮显示"已复制 ✓"反馈 500ms

## MODIFIED Requirements

### Requirement: 卡片渲染逻辑
`renderMdRefCards()` 从一次渲染改为每次进入视图重新渲染，移除 `_mdRefRendered` 标志。保留 `marked.parse()` + `hljs.highlightElement()` 核心逻辑。

### Requirement: 10 个语法类别
保留原有 10 个语法类别的内容：（1）标题、（2）文本样式、（3）链接与图片、（4）列表、（5）代码块、（6）引用、（7）表格、（8）任务列表、（9）分割线、（10）转义字符。源码内容和预览渲染不变。

### Requirement: 6 套主题兼容
所有新样式继续使用 CSS 变量（`var(--accent)`、`var(--bg-secondary)`、`var(--border)`、`var(--card-bg)`、`var(--font-family)` 等），确保 6 套主题自动适配。交通灯圆点颜色固定（红 #FF5F57 / 黄 #FEBC2E / 绿 #28C840），不跟随主题。

### Requirement: 「打开编辑器试试」功能
保留 `openMdRefTryEditor()` 的 async/await 逻辑和 HTML 实体解码不变。仅调整 CSS 类名以匹配新结构。
