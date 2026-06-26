# Tasks

## Task 1: 新建目录结构并创建入口文件 ✅
- [x] 创建 `src/css/` 目录
- [x] 创建 `src/css/components/` 目录
- [x] 创建 `src/css/index.css`（入口文件，用 `@import` 按顺序引入所有子文件）

## Task 2: 拆分 app.css ✅
- [x] 提取 `variables.css` — `:root` 设计变量（亮/暗主题）
- [x] 提取 `reset.css` — `*`、`html`、`body`、`#app` 基础重置
- [x] 提取 `scrollbar.css` — 全局 `::-webkit-scrollbar` 系列规则（去重后）
- [x] 提取 `animations.css` — 动画工具类（.anim-fade-in 等）

## Task 3: 拆分 style.css — 基础布局组件 ✅
- [x] 提取 `components/topbar.css` — 顶部栏样式
- [x] 提取 `components/sidebar.css` — 笔记本侧栏
- [x] 提取 `components/main-content.css` — #mainContent、view、search-results、data-content

## Task 4: 拆分 style.css — 编辑器相关 ✅
- [x] 提取 `components/editor.css` — CM6、toolbar、markdown 渲染、代码高亮、预览

## Task 5: 拆分 style.css — 弹窗/模态框 ✅
- [x] 提取 `components/search-modal.css` — Ctrl+F 搜索弹窗
- [x] 提取 `components/modals.css` — batch-tag、move-notebook、shortcuts 等通用弹窗

## Task 6: 拆分 style.css — 其余组件 ✅
- [x] 提取 `components/dropdowns.css` — 下拉菜单、字体选择等
- [x] 提取 `components/data-view.css` — 数据管理视图（标签栏、卡片网格）
- [x] 提取 `components/md-reference.css` — Markdown 引用页面
- [x] 提取 `components/settings-panel.css` — 设置面板

## Task 7: 更新 main.js 并删除旧文件 ✅
- [x] 修改 `src/main.js`：将 `import './app.css'` 和 `import './style.css'` 替换为 `import './css/index.css'`
- [x] 验证构建成功（`npx vite build`）
- [x] 删除 `src/app.css` 和 `src/style.css`

## Task 8: 视觉回归验证
- [x] 构建通过，CSS 产物从 94.16 KiB 降至 91.31 KiB（去重效果）
- [ ] 启动开发服务器，逐一检查各主要页面/组件样式与重构前一致
- [ ] 如发现差异，回退到旧文件排查原因并修复
- [ ] 所有检查点通过后，确认重构完成

# Task Dependencies
- Task 7 依赖 Task 1~6（必须先创建好所有 CSS 文件再修改 import 和删除旧文件）
- Task 4、5、6 可并行执行（彼此无依赖）
