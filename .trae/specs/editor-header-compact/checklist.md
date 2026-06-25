# Checklist

## Header padding
- [x] `.editor-header` 的 `padding-top` = 4px（之前 12px）
- [x] 编辑/新建/查看三种模式下打开笔记,header 按钮离顶部 4px
- [x] fullscreen 全屏模式下也生效

## 标题 padding
- [x] `.editor-input` 的 `padding` = `2px 0`（之前 `8px 0`）
- [x] 标题字号 1.5rem / 字重 600 不变
- [x] 标题 focus 时的 `border-bottom: 1px solid var(--accent)` 高亮正常显示
- [x] 标题底部与标签之间的距离明显收紧

## 标签区 margin
- [x] `.editor-section` 的 `margin-bottom` = 6px（之前 24px）
- [x] 标签底部与正文（CodeMirror / 预览区）间距 = 6px
- [x] 标签 chip 自身样式不变（padding / border / 颜色）

## 内容区效果
- [x] CodeMirror 编辑器在固定窗口高度下可见行数 +1
- [x] Markdown 预览模式首屏内容增加
- [x] 3 种模式（编辑/新建/查看）都生效

## 回归测试
- [x] 标题输入正常,无卡顿
- [x] 标签 chip 增/删/点击交互不受影响
- [x] 6 个主题（default/nord/monokai-pro/light/tokyo-night/dark）下样式正常
- [x] fullscreen 全屏模式下视觉一致
- [x] 标题 hover 状态、标签 hover 状态、按钮 hover 状态都正常
- [x] 没有视觉错位、遮挡、对齐问题
