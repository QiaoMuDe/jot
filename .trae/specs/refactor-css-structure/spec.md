# CSS 文件结构重构 Spec

## Why

当前 `app.css` 和 `style.css` 两个文件都放在 `src/` 根目录下，其中 `style.css` 体积接近 5100 行，所有样式按功能混杂在一起，导致：
- 维护困难：修改某个功能需要在大文件中反复搜索定位
- 重复冗余：多个容器重复定义 `::-webkit-scrollbar`、`scrollbar-button` 等样式
- 加载顺序耦合：`style.css` 和 `app.css` 靠 `main.js` 中的 import 顺序决定优先级，不直观

## What Changes

1. **新建 `src/css/` 目录**，将所有 CSS 源文件移入
2. **将 `app.css` 拆分为设计系统文件**（变量、基础重置、全局滚动条、动画工具类）
3. **将 `style.css` 按功能拆分为多个文件**，每个文件负责一个功能模块
4. **合并重复的滚动条样式**，提取统一的滚动条 mixin/class
5. **更新 `main.js`** 中的 import 路径

## Impact

- Affected specs: 无（纯重构，不影响功能）
- Affected code: `main.js`（import 路径变更），新增 `src/css/` 目录及其下多个文件，删除 `src/app.css`、`src/style.css`

## 拆分方案

### 文件结构

```
src/css/
├── variables.css          # app.css 中的 :root 设计变量
├── reset.css              # app.css 中的 * / body / html / #app 基础重置
├── scrollbar.css          # 全局滚动条样式（统一管理，去重）
├── animations.css         # 动画工具类（anim-fade-in 等）
├── components/
│   ├── topbar.css         # 顶部栏样式
│   ├── sidebar.css        # 侧栏、笔记本列表
│   ├── main-content.css   # #mainContent、view、search-results 等
│   ├── search-modal.css   # Ctrl+F 搜索弹窗
│   ├── editor.css         # 编辑器（CM6、toolbar、markdown 渲染）
│   ├── dropdowns.css      # 下拉菜单、字体选择等
│   ├── modals.css         # 模态弹窗（batch-tag、move-notebook、shortcuts 等）
│   ├── data-view.css      # 数据管理视图
│   ├── md-reference.css   # Markdown 引用页面
│   └── settings-panel.css # 设置面板
└── index.css              # 入口文件，@import 所有以上文件（控制加载顺序）
```

### 具体拆分映射

| 原文件 | 目标文件 | 内容说明 |
|--------|----------|----------|
| app.css:1-388 | variables.css | `:root` 设计变量（含亮/暗主题） |
| app.css:389-437 | reset.css | `*`, `html`, `body`, `#app` 基础重置 |
| app.css:438-505 | scrollbar.css | 全局 `::-webkit-scrollbar` 系列规则 |
| app.css:507-524 | animations.css | 动画工具类 |
| style.css 顶部 ~130 行 | components/topbar.css | 顶部栏（topbar、search-box 等） |
| style.css Section B | components/main-content.css | #mainContent、view、search-results、data-content |
| style.css 标签/卡片/数据 | components/data-view.css | .tag-bar、.data-content、卡片网格 |
| style.css Section X | components/search-modal.css | Ctrl+F 搜索弹窗全部样式 |
| style.css 编辑器相关 | components/editor.css | CM6、toolbar、markdown 渲染、代码高亮 |
| style.css 弹窗相关 | components/modals.css | batch-tag、move-notebook、shortcuts |
| style.css 侧栏相关 | components/sidebar.css | 笔记本侧栏 |
| style.css 字体设置 | components/dropdowns.css | font-family、字体大小等 |
| style.css md-ref | components/md-reference.css | Markdown 引用页面 |
| style.css 其余 | components/settings-panel.css | 设置面板 |

### 去重策略

同一组件多次出现的 `::-webkit-scrollbar` 规则全部合并到 `scrollbar.css` 中。对于需要自定义滚动条的容器，统一用一个 data 属性 `[data-scrollbar]` 或类名 `.custom-scrollbar` 来应用样式，避免每个容器重复写 6-8 行滚动条样式。

## ADDED Requirements

### Requirement: CSS 文件结构重组

The system SHALL restructure CSS files into a dedicated `src/css/` directory with functional splitting.

#### Scenario: 构建验证
- **WHEN** 构建项目时
- **THEN** 新 CSS 结构不应产生构建错误，且最终产物内容与原结构一致

#### Scenario: 功能无退化（视觉回归验证）
- **WHEN** 页面渲染时
- **THEN** 所有样式应与重构前完全一致，无视觉差异
- **验证方法**：在重构前后分别截取各主要页面/组件的截图进行对比，包括：
  - 首页笔记列表（亮色/暗色主题）
  - 编辑器页面（编辑模式、预览模式）
  - 搜索弹窗
  - 设置面板
  - 数据管理视图
  - 侧栏各种状态（展开/折叠）
  - Markdown 引用页
  - 各类弹窗（batch-tag、shortcuts、move-notebook）
- **WHEN** 发现视觉差异时
- **THEN** 逐一排查差异原因并修复，确保最终产物与重构前一致

## REMOVED Requirements

### Requirement: src/app.css、src/style.css 两个单文件

**Reason**: 统一移到 `src/css/` 目录并按功能拆分，提高可维护性
**Migration**: `main.js` 中的 `import './app.css'` 和 `import './style.css'` 替换为 `import './css/index.css'`
