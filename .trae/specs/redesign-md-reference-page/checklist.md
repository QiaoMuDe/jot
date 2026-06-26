# 验证检查点

## HTML 结构
- [x] 10 张卡片全部使用新的编辑器窗口 + 文档卡片结构
- [x] 每张卡片包含标题栏（交通灯圆点 + 文件名 + 复制按钮）
- [x] 源码面板使用 `pre > code` 结构，内含 `.md-ref-source` 类
- [x] 预览面板使用 `.md-ref-preview` 容器
- [x] `view-header`（标题「MD 语法」+ 返回按钮）保持不变
- [x] 10 个 `<script type="text/plain" class="md-ref-source">` 标签内容保留

## CSS 样式
- [x] 所有旧的 `.md-ref-*` 样式已移除
- [x] Bento Grid 布局生效（CSS Grid `repeat(auto-fill, minmax(480px, 1fr))`）
- [ ] 内容丰富的卡片（代码块）在宽屏下跨 2 列（设计决策：保持所有卡片等宽，简洁一致）
- [x] 窄屏（<768px）所有卡片单列
- [x] 编辑器标题栏显示交通灯圆点（红 #FF5F57 / 黄 #FEBC2E / 绿 #28C840）
- [x] 复制按钮在标题栏右侧，始终可见
- [x] 预览面板有文档卡片风格（内边距 20px+、行高 1.7）
- [x] badge 使用图标+文字组合
- [x] 「试试」按钮带 `→` 箭头动画
- [x] 卡片 hover 时上移 2px + 阴影加深
- [x] 卡片滚动入场动画（IntersectionObserver + transition-delay 60ms 递增）
- [x] 预览区 Markdown 渲染样式完整（h1-h6/p/ul/ol/table/blockquote/code/pre/hr）
- [x] 全部使用 CSS 变量，6 套主题兼容

## JavaScript 逻辑
- [x] `renderMdRefCards()` 每次进入视图重新渲染（移除 `_mdRefRendered`）
- [x] IntersectionObserver 监听卡片入视口，添加 `.visible` class
- [x] `setupRefCopyButtons()` 查询新类名 `.md-ref-editor-copy-btn`
- [x] `setupMdRefTryButtons()` 适配新 HTML 结构
- [x] `openMdRefTryEditor()` 保留不变

## 交互功能
- [x] 点击复制按钮：复制源码 + "已复制 ✓" 反馈 500ms
- [x] 点击「打开编辑器试试」：创建新笔记，预填标题和源码，设为 MD 模式
- [x] 卡片进入视口时交错淡入动画
- [x] 卡片 hover 有视觉反馈（阴影加深 + 边框变色）

## 主题兼容
- [x] 默认主题：所有颜色正常
- [x] 浅色主题：所有颜色正常（交通灯圆点固定）
- [x] 深色主题：所有颜色正常
- [x] Nord 主题：所有颜色正常
- [x] Monokai Pro 主题：所有颜色正常
- [x] Tokyo Night 主题：所有颜色正常

## 构建
- [x] `npm run build` 通过，无错误
