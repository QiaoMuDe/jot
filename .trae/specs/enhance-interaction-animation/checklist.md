# Checklist - 交互体验与动画增强

## 动画基础（Task 0）
- [x] CSS 动画变量已定义（duration/easing 统一管理）
- [x] 通用动画工具类已实现（fade-in, slide-up, scale-in）
- [x] `prefers-reduced-motion` 降级已添加
- [x] 主题切换 `transition` 已添加到 CSS 变量选择器

## 视图切换（Task 1）
- [x] 所有主视图（网格/数据/回收站/设置）切换使用 opacity + transform 动画
- [x] 进入 250ms / 退出 150ms
- [x] 切换流畅无闪烁

## 所有笔记/卡片网格（Task 2）
- [x] 卡片交错入场（40ms 间隔，opacity + translateY + scale）
- [x] 最多 12 张卡片有延迟动画
- [x] 卡片 hover 阴影过渡 + scale(1.02)
- [x] 卡片点击 scale(0.98) 反馈
- [x] 置顶按钮旋转+缩放弹跳动画
- [x] 加载更多新卡片底部淡入
- [x] 刷新数据时动画重置

## 数据管理页面（Task 3）
- [x] 统计卡片交错缩放淡入（80ms 间隔）
- [x] 数字 count-up 动画
- [x] 操作按钮 hover/active 反馈

## 回收站页面（Task 4）
- [x] 列表项交错入场（translateY + opacity）
- [x] 单条恢复收缩淡出
- [x] 单条永久删除带红色闪烁
- [x] 全部恢复/清空依次收缩淡出

## 设置页面（Task 5）
- [x] toggle-switch 滑块+背景色过渡（200ms）
- [x] 字体下拉菜单展开/收起动画

## 帮助页面（Task 6）
- [x] 遮罩淡入（200ms）
- [x] 卡片缩放淡入（300ms，弹簧缓动）
- [x] 快捷键列表交错入场（30ms）
- [x] 关闭动画正常

## 新建/编辑笔记（Task 7）
- [x] 模态缩放淡入（300ms）
- [x] 遮罩淡入（200ms）
- [x] 内容区域延迟入场
- [x] 顶部品牌色条展开动画（editorBarEnter，scaleX 0→1，300ms）
- [x] 关闭缩放淡出（180ms）
- [x] 自动保存指示器脉冲
- [x] 标签 chip 选择脉冲动画
- [x] 颜色选择扩散动画 — ⏭️ 不适用，编辑器内无颜色选择器组件

## 查看笔记只读模式（Task 8）
- [x] 开关动画复用编辑器
- [x] Markdown 渲染区域淡入（200ms）
- [x] 代码块着色后淡入

## 关于页面（Task 9）
- [x] 遮罩+卡片缩放动画复用帮助页面
- [x] 品牌 Logo 弹性缩放（300ms）
- [x] 版本号延迟淡入
- [x] 项目链接 hover 反馈

## 笔记全屏/快速笔记（Task 10）
- [x] 进入全屏展开动画（300ms）
- [x] 退出全屏收缩动画（200ms）

## 批量模式（Task 11）
- [x] 批处理栏滑入（translateY + opacity，250ms）
- [x] 批处理栏滑出（150ms）
- [x] 复选框交错淡入
- [x] topbar 按钮淡入/淡出（opacity transition 0.15s + setTimeout 延迟隐藏）

## Toast（Task 12）
- [x] 入场底部滑入+淡入（250ms）
- [x] 离场滑出+淡出（200ms）
- [x] 多条堆叠上移

## 右键菜单（Task 13）
- [x] 打开缩放+淡入（150ms）
- [x] 关闭缩放+淡出（100ms）

## 更多菜单（Task 14）
- [x] 打开从触发位置缩放展开（150ms，transform-origin: top right）
- [x] 关闭缩放收起

## 主题切换（Task 15）
- [x] 所有受影响属性平滑过渡（300ms）
- [x] 无闪烁或跳变

## 综合（Task 16）
- [x] 所有主视图切换流畅
- [x] 卡片网格动画正常
- [x] 数据管理页面动画正常
- [x] 回收站操作动画正常
- [x] 设置页面动画正常
- [x] 帮助/关于模态动画正常
- [x] 编辑器新建/编辑/查看动画正常
- [x] 全屏模式过渡正常
- [x] 批量模式过渡正常
- [x] Toast/右键菜单/更多菜单动画正常
- [x] 主题切换平滑
- [x] `prefers-reduced-motion` 降级正常
- [x] 无重排/重绘性能问题（动画均使用 opacity/transform，仅触发 composite）
- [x] `npx vite build` 通过
