# Jot 项目介绍落地页 - 实施计划

## 一、概述

在项目根目录创建 `landing/` 目录，编写一个静态落地页（`index.html`），用于对外介绍 Jot 卡片式笔记桌面应用。页面需精美 UI 设计 + 丝滑交互动画。

## 二、项目现状分析

* **项目名称**: Jot — 卡片式笔记桌面应用

* **技术栈**: Go 1.26+ / Wails v2 / SQLite / 原生 HTML+CSS+JS / Vite

* **核心功能**: 笔记管理（卡片式网格）、AI 助手（双 Provider + 联网搜索 + 卡片召回）、Markdown 编辑器（CodeMirror 6）、标签系统、多主题（14 套）、搜索、备份还原、回收站、Mermaid 图表等

* **前端现状**: 纯手写 CSS，无 UI 框架，CSS 变量主题系统，过渡动画

* **设计系统建议** (来自 ui-ux-pro-max):

  * 风格: Glassmorphism（毛玻璃）

  * 颜色: Primary `#2563EB`, Secondary `#3B82F6`, CTA `#F97316`, Background `#F8FAFC`

  * 排版: Inter 字体

  * 效果: 毛玻璃模糊（10-20px）、微妙边框、光影深度

## 三、实施计划

### 3.1 创建目录结构

```
landing/
  index.html          # 主页面（包含所有 HTML/CSS/JS，单文件自包含）
```

使用单文件 `index.html` 内嵌 CSS 和 JS，方便部署和分享，无需额外构建工具。

### 3.2 页面内容结构

**Hero 区域** — 全屏毛玻璃 Hero

* 背景使用动态渐变网格 + 光晕粒子动画

* 项目 Logo / 名称 "Jot" + 标语 "卡片式笔记桌面应用"

* 简短描述（一句话概括）

* 两个 CTA 按钮：Gitee Star / 查看特性

* 向下滚动提示动画

**特性展示区域** — 卡片网格布局

* 6-8 张特性卡片，展示核心功能：

  1. 卡片式笔记网格 — 三步交互范式
  2. AI 智能助手 — 双 Provider 流式输出
  3. Markdown 编辑器 — CodeMirror 6 专业体验
  4. 多主题系统 — 14 套主题自由切换
  5. 标签系统 — 自定义标签 + 筛选
  6. 联网搜索 — Tavily / 知乎 / 全网搜索
  7. 数据管理 — 备份还原 / 回收站 / 统计
  8. 本地存储 — SQLite 私有本地数据

* 每张卡片 hover 时浮起 + 毛玻璃效果

* 卡片进入视口时 staggered 动画

**技术栈展示** — 图标网格

* 展示关键技术：Go, Wails, SQLite, GORM, CodeMirror, Marked, highlight.js, Mermaid 等

* 使用徽章样式呈现

**数据统计/亮点区域**

* 数字亮点：如 "14 套主题"、"12 项 AI 技能"、"纯手写 CSS" 等

* 动画计数器滚动效果

**CTA 底部区域**

* 号召性用语 + 按钮

* 页脚：Gitee 链接、许可证信息

### 3.3 交互与动画设计

| 元素         | 动画效果                         | 实现方式                                                      |
| ---------- | ---------------------------- | --------------------------------------------------------- |
| Hero 背景    | 渐变网格 + 光晕粒子缓慢浮动              | CSS @keyframes + 伪元素                                      |
| 标题文字       | 分段淡入 + 向上位移（stagger）         | Intersection Observer + CSS transition                    |
| 特性卡片       | 进入视口时 staggered 缩放入场 + 弹性缓动  | Intersection Observer + cubic-bezier(0.34, 1.56, 0.64, 1) |
| 特性卡片 hover | 向上浮起 translateY(-4px) + 阴影加深 | CSS transition                                            |
| 数字统计       | 滚动到视口时数字递增动画                 | requestAnimationFrame 计数器                                 |
| CTA 按钮     | hover 时弹跳 + 阴影变化             | CSS transition + transform                                |
| 滚动指示器      | 上下弹跳箭头                       | CSS @keyframes                                            |
| 导航栏        | 页面滚动时背景从不透明→毛玻璃              | scroll 事件 + class 切换                                      |

动画原则（遵循 ui-ux-pro-max 指南）：

* 微交互 150-300ms，复杂过渡 ≤400ms

* 仅使用 transform/opacity，避免动画 width/height

* 使用弹性缓动 cubic-bezier(0.34, 1.56, 0.64, 1)

* 尊重 prefers-reduced-motion

### 3.4 视觉设计

* **风格**: Glassmorphism（毛玻璃）+ 现代极简

* **配色**:

  * 主色: `#2563EB` (蓝色)

  * 辅色: `#3B82F6`

  * CTA: `#F97316` (橙色)

  * 背景: `#F8FAFC` → 深色渐变

  * 文字: `#1E293B`

* **字体**: 使用 Inter 字体（Google Fonts CDN）

* **毛玻璃效果**: backdrop-filter: blur(12-20px), 半透明背景, 1px 白色半透明边框

* **响应式**: 适配 375px / 768px / 1024px / 1440px

### 3.5 文件结构

单文件 `landing/index.html` 包含：

* `<head>`: meta, 字体 CDN, 内联 CSS

* `<body>`: 所有 HTML 结构

* `<script>` 底部: 所有 JS 逻辑（Intersection Observer, 计数器动画, 滚动导航等）

### 3.6 验证方式

* 直接在浏览器中打开 `landing/index.html` 即可预览

* 检查响应式布局（Chrome DevTools 设备模拟）

* 检查动画流畅度（60fps）

* 检查 prefers-reduced-motion 媒体查询

## 四、无需修改的现有文件

本次任务仅新建 `landing/` 目录和 `landing/index.html` 文件，不修改项目中的任何现有文件。

## 五、依赖

* Google Fonts (Inter) — 通过 CDN 加载

* 无其他外部依赖，纯原生 HTML/CSS/JS

