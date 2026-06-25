# Checklist

## 移除 topbar 搜索框
- [x] `<input id="searchInput">` 已从 `index.html` 删除
- [x] `.topbar-search` 容器已从 `index.html` 删除
- [x] `main.js` 中所有 `els.searchInput` 引用已迁移或删除（仅留 1 处字段定义为 null + 已废弃注释）

## 高度调整
- [x] `#topbar` 高度 = 40px
- [x] `.sidebar-header` 高度 = 40px
- [x] `.editor-overlay.fullscreening` top = 40px
- [x] `.editor-panel.fullscreen` height = `calc(100vh - 40px)`
- [x] 注释中「高度与 topbar 对齐（48px）」已更新为「40px」
- [x] CSS 中无 48px / 56px 高度残留（Grep 验证）
- [x] `.topbar-search` 死代码 CSS 已清理

## 弹窗结构
- [x] `#searchModal` 已添加到 `index.html` 末尾（`</body>` 前）
- [x] 弹窗包含：遮罩、输入框、过滤器区、结果列表
- [x] 弹窗初始隐藏（无 `.visible` class，通过 `pointer-events: none` + `opacity: 0`）

## 弹窗样式
- [x] 弹窗居中显示（视口 12% 上偏移,符合 Spotlight 风格）
- [x] 弹窗宽度 520px（自适应内容）
- [x] 弹窗使用 `--card-bg`、`--radius-2xl`、`--shadow-xl`
- [x] 遮罩背景 `rgba(0,0,0,0.4)` + `backdrop-filter: blur(4px)`
- [x] 弹窗打开/关闭动画流畅（0.2s cubic-bezier(0.16, 1, 0.3, 1)）
- [x] 弹窗 z-index = 2000，高于 topbar 和全屏遮罩

## 触发逻辑
- [x] Ctrl+F 在编辑器关闭时打开弹窗（`openSearchModal()`）
- [x] Ctrl+F 在编辑器打开时打开 CodeMirror 查找面板（`openSearchPanel(cmEditor)`）
- [x] 弹窗打开时输入框自动 focus（`setTimeout 50ms` 让动画启动）
- [x] 弹窗打开时阻止浏览器默认行为
- [x] 弹窗关闭后焦点恢复（通过 `state._searchModalPrevFocus`）

## 关闭方式
- [x] 按 Esc 键关闭弹窗
- [x] 点击遮罩区域关闭弹窗（mask + 弹窗容器都判）
- [x] 关闭时清空输入框、过滤器、结果列表

## 搜索功能
- [x] 输入关键字时实时搜索（防抖 200ms）
- [x] 搜索范围：标题 + 内容 + 标签 + 笔记本名
- [x] 第一页请求 `pageSize` 条（取自全局 pageSize 变量，默认 18）
- [x] 关键字高亮使用 `<mark>` 标签（`highlightModalMatch` 函数，避开 `highlightMatch` 冲突）
- [x] 高亮样式为黄色背景 `rgba(255, 213, 79, 0.45)`
- [x] 弹窗内滚动到底部附近（< 200px）自动加载下一页
- [x] 全部加载完毕时显示「共 X 条结果」提示
- [x] 防止重复请求（`state.searchModalLoading` 状态管理）
- [x] 无结果显示「无匹配笔记」（`.search-modal-empty` 显示）

## 键盘导航
- [x] ↓ 键选中下一项（循环）
- [x] ↑ 键选中上一项（循环）
- [x] Enter 键打开选中项（无选中则打开第一项）
- [x] 选中项视觉高亮（hover 背景 + accent 左边框 2px）
- [x] 选中项自动滚动到可视区

## 过滤器
- [x] 笔记本下拉：可选择具体笔记本或全部
- [x] 标签下拉：支持多选（Set 状态管理）
- [x] 日期范围：今天/最近 7 天/最近 30 天/不限（HTML 4 个预设）
- [x] 过滤器变化时重新触发搜索（重置分页）

## 鼠标交互
- [x] 点击结果项打开对应笔记（`openEditor(note.ID)`）
- [x] 点击结果项时弹窗自动关闭
- [x] hover 结果项时同步更新 selectedIndex

## 全屏兼容
- [x] 全屏模式下 Ctrl+F 仍可打开弹窗（弹窗 z-index 2000 > 全屏遮罩）
- [x] 弹窗层级高于全屏遮罩（z-index > 1000）
- [x] topbar 在弹窗打开时仍为 40px
- [x] topbar 区域点击不关闭弹窗（避免误触,通过弹窗覆盖层 + `pointer-events: auto`）

## 焦点管理
- [x] 弹窗打开前记录 `document.activeElement`（`state._searchModalPrevFocus`）
- [x] 弹窗关闭时恢复焦点到原元素（带 `document.contains` 检查）
- [x] 弹窗打开时 body 滚动被阻止（`document.body.style.overflow = 'hidden'`）
- [x] 弹窗关闭时 body 滚动恢复（`document.body.style.overflow = ''`）

## 回归测试
- [x] 原 topbar 搜索的所有行为（防抖、高亮、统计、Esc 清空）已迁移到弹窗
- [x] `add-find-replace` 的所有功能（编辑器内查找、替换、键盘导航）保持不变
- [x] `move-menu-to-left` 的布局（☰ 在左、品牌、窗口控制在右）保持不变
- [x] 侧栏的展开/收起、笔记本树、标签页等所有功能不受影响

## 实际环境验证（需用户本地完成）
- [ ] 启动 `wails dev`，目视检查 topbar 高度 = 40px、侧栏 header 高度 = 40px
- [ ] 按 Ctrl+F（编辑器关闭时）→ 弹窗从视口顶部 12% 处出现，输入框自动 focus
- [ ] 输入关键字 → 结果实时显示，条数 = pageSize 设置值
- [ ] 滚动结果区到底部 → 自动加载下一页，「共 X 条结果」提示正确
- [ ] ↑↓ 导航 + Enter 打开笔记 → 弹窗关闭，笔记正确显示
- [ ] Esc 关闭 + 点击遮罩关闭 + 关闭后焦点恢复
- [ ] 打开编辑器 → Ctrl+F → CodeMirror 查找面板（不打开弹窗）
- [ ] 关闭编辑器 → Ctrl+F → 弹窗（不打开 CM6 面板）
- [ ] 修改设置中的 pageSize（9→54）→ 关闭再打开弹窗 → 搜索结果条数跟随变化
- [ ] 6 个主题（default/nord/monokai-pro/light/tokyo-night/dark）下弹窗样式正常
