# Tasks - 交互体验与动画增强

## Task Dependencies
- [Task 0] 必须先完成（CSS 动画变量和工具类基础）
- [Task 1-15] 可在 Task 0 之后独立并行
- [Task 16] 依赖所有其他 Task（综合验证）

---

## Tasks

- [ ] **Task 0**: 动画基础建设 — CSS 动画变量 + 工具类 + prefers-reduced-motion 降级
  - [ ] 0.1 在 `app.css` 的 `:root` 中添加动画相关 CSS 变量（`--anim-duration-fast: 150ms`, `--anim-duration-normal: 250ms`, `--anim-duration-slow: 300ms`, `--anim-easing-spring: cubic-bezier(0.16, 1, 0.3, 1)`, `--anim-easing-out: ease-out`, `--anim-easing-in: ease-in`）
  - [ ] 0.2 创建通用动画工具类（`.anim-fade-in`, `.anim-slide-up`, `.anim-scale-in`）
  - [ ] 0.3 在 `app.css` 添加 `prefers-reduced-motion: reduce` 媒体查询，统一禁用动画
  - [ ] 0.4 在 `:root` 和 `[data-theme]` 选择器上添加 `transition` 属性（background-color, color, border-color, box-shadow），为主题切换过渡做准备
  - [ ] 0.5 构建验证（`npx vite build`）

- [x] **Task 1**: 视图切换过渡动画（所有主视图）
  - [x] 1.1 在 `style.css` 中创建视图切换动画 class（`.view-enter`, `.view-exit`）和 keyframes ✓（Task 0 已在 app.css 中完成）
  - [x] 1.2 重构 `main.js` 中的 `switchView()` 函数：用 class 切换替代直接 display 控制
  - [x] 1.3 进入视图 animation: fadeIn + slideUp（250ms, `viewEnter`）
  - [x] 1.4 退出视图 animation: fadeOut + slideDown（150ms, `viewExit`）
  - [x] 1.5 构建验证

- [x] **Task 2**: 所有笔记页面（卡片网格）动画
  - [x] 2.1 创建卡片入场关键帧动画（`@keyframes cardEnter`：opacity + translateY + scale）✓（Task 0 已在 app.css 中完成）
  - [x] 2.2 修改 `renderCardGrid()`：渲染后为每张卡片设置 `animation-delay`（40ms * index），最多 12 项
  - [x] 2.3 卡片 hover 增强：阴影过渡 + scale(1.01) 微缩放
  - [x] 2.4 卡片点击 active 反馈：scale(0.98)
  - [x] 2.5 置顶按钮 ☆/★ 切换：旋转动画（`rotateIn` keyframe）
  - [x] 2.6 加载更多：新卡片从底部淡入（`cardEnter` + `append` 模式）
  - [x] 2.7 数据刷新时动画重置
  - [x] 2.8 构建验证

- [x] **Task 3**: 数据管理页面动画
  - [x] 3.1 统计卡片交错入场（3 张，80ms 间隔，scale + opacity）
  - [x] 3.2 统计数字 count-up 动画（0 → 实际值，300ms）
  - [x] 3.3 操作按钮 hover/active 缩放反馈
  - [x] 3.4 构建验证

- [x] **Task 4**: 回收站页面动画
  - [x] 4.1 列表项交错入场（30ms 间隔，translateX + opacity）
  - [x] 4.2 单条恢复：收缩淡出动画
  - [x] 4.3 单条永久删除：收缩淡出 + 红色闪烁
  - [x] 4.4 全部恢复：所有项依次收缩淡出（30ms 间隔）
  - [x] 4.5 全部清空：所有项依次收缩淡出 + 红色闪烁
  - [x] 4.6 构建验证

- [x] **Task 5**: 设置页面动画
  - [x] 5.1 toggle-switch 开关动画（滑块移动 + 背景色过渡，200ms）
  - [x] 5.2 字体族下拉菜单展开/收起动画（max-height + opacity）
  - [x] 5.3 构建验证

- [x] **Task 6**: 帮助页面（快捷键说明）动画
  - [x] 6.1 遮罩淡入动画（200ms）
  - [x] 6.2 卡片缩放淡入（scale(0.9→1) + opacity，300ms，弹簧缓动）
  - [x] 6.3 快捷键列表项交错入场（30ms 间隔）
  - [x] 6.4 关闭动画（scale(0.95) + opacity，150ms）
  - [x] 6.5 构建验证

- [x] **Task 7**: 新建/编辑笔记（编辑器模态框）动画
  - [x] 7.1 模态打开：scale(0.85→1) + opacity(0→1)，300ms
  - [x] 7.2 遮罩淡入：200ms
  - [x] 7.3 内容区域延迟入场：translateY(16px→0) + opacity，200ms
  - [ ] 7.4 顶部品牌色条展开：scaleX(0→1)，300ms，transform-origin: center
  - [x] 7.5 关闭动画：scale(0.95) + opacity，180ms
  - [x] 7.6 自动保存指示器脉冲动画（绿色圆点，500ms）
  - [x] 7.7 标签 chip 选择脉冲缩放动画
  - [ ] 7.8 颜色选择扩散动画
  - [x] 7.9 构建验证

- [x] **Task 8**: 查看笔记（只读模式）动画
  - [x] 8.1 复用编辑器模态打开/关闭动画
  - [x] 8.2 Markdown 渲染区域淡入（200ms）
  - [x] 8.3 代码块着色后淡入（100ms）
  - [x] 8.4 构建验证

- [x] **Task 9**: 关于页面动画
  - [x] 9.1 复用帮助页面的遮罩淡入 + 卡片缩放动画
  - [x] 9.2 品牌 Logo 弹性缩放（scale(0.8→1)，300ms）
  - [x] 9.3 版本号延迟淡入（100ms 延迟）
  - [x] 9.4 项目链接 hover 上移 + 颜色过渡
  - [x] 9.5 关闭动画
  - [x] 9.6 构建验证

- [x] **Task 10**: 笔记全屏动画（快速笔记模式）
  - [x] 10.1 进入全屏：编辑器面板展开动画（300ms）
  - [x] 10.2 退出全屏：编辑器面板收缩动画（200ms）
  - [x] 10.3 集成到现有 toggleEditorFullscreen 函数
  - [x] 10.4 构建验证

- [x] **Task 11**: 批量模式过渡动画
  - [x] 11.1 批处理栏滑入（translateY(100%→0) + opacity，250ms）
  - [x] 11.2 批处理栏滑出（translateY(0→100%)，150ms）
  - [x] 11.3 卡片复选框交错淡入（20ms 间隔）
  - [x] 11.4 topbar 批量按钮淡入/淡出
  - [x] 11.5 构建验证

- [x] **Task 12**: Toast 通知动画增强
  - [x] 12.1 入场：translateY(24px→0) + opacity(0→1)，250ms
  - [x] 12.2 离场：translateY(0→-12px) + opacity(1→0)，200ms
  - [x] 12.3 多条堆叠上移动画（translateY，200ms）
  - [x] 12.4 撤销按钮 hover 缩放
  - [x] 12.5 构建验证

- [x] **Task 13**: 右键菜单动画
  - [x] 13.1 打开：scale(0.9→1) + opacity(0→1)，150ms
  - [x] 13.2 关闭：scale(1→0.95) + opacity(1→0)，100ms
  - [x] 13.3 构建验证

- [x] **Task 14**: 更多菜单动画
  - [x] 14.1 打开：从触发位置缩放展开（scale(0.85→1) + opacity，150ms，transform-origin: top right）
  - [x] 14.2 关闭：缩放收起，100ms
  - [x] 14.3 构建验证

- [x] **Task 15**: 主题切换过渡
  - [x] 15.1 确保所有受影响 CSS 属性有 transition（background-color, color, border-color, box-shadow）
  - [x] 15.2 验证切换时平滑过渡，无闪烁
  - [x] 15.3 处理性能：对高频元素使用 `will-change`
  - [x] 15.4 构建验证

- [ ] **Task 16**: 综合验证与修复
  - [ ] 16.1 验证所有主视图切换流畅
  - [ ] 16.2 验证卡片网格入场 + hover + 点击反馈
  - [ ] 16.3 验证数据管理页面动画
  - [ ] 16.4 验证回收站操作动画
  - [ ] 16.5 验证设置页开关/下拉动画
  - [ ] 16.6 验证帮助/关于模态动画
  - [ ] 16.7 验证编辑器新建/编辑/查看动画
  - [ ] 16.8 验证全屏模式过渡
  - [ ] 16.9 验证批量模式过渡
  - [ ] 16.10 验证 Toast、右键菜单、更多菜单
  - [ ] 16.11 验证主题切换平滑
  - [ ] 16.12 验证 `prefers-reduced-motion` 降级正常
  - [ ] 16.13 确认无重排/重绘性能问题
  - [ ] 16.14 最终构建验证（`npx vite build` 通过）
