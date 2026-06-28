# Checklist

## 依赖 & 构建
- [x] 6 个新依赖安装成功，`package.json` 和 `package-lock.json` 已更新
- [x] `npx vite build` 通过，零 warning/error

## cm6-syntax-highlight.js 模块
- [x] `mdHighlightStyle` 保留原 16 个 MD 专有 tag，内容完整
- [x] `codeHighlightStyle` 定义 ~25 个编程通用 tag，颜色引用 CSS 变量
- [x] 原生语言解析器：`javascript`（含 ts/jsx/tsx）、`css`、`html`、`json`、`python` 已导入
- [x] legacy-modes 兜底语言：go/sql/sh/yaml/xml/rust/ruby/swift/perl/lua/haskell/powershell/dockerfile/cpp/java/csharp/scala/dart 已导入
- [x] `langMap` 映射表完整，覆盖约 30 种扩展名
- [x] `getHighlightExtension(fileExt)` 工厂函数正确返回三种情况（MD 高亮 / 代码高亮 / 空数组）
- [x] 未知扩展名返回空数组，不报错

## main.js 修改
- [x] import 已更新为 `{ jotTheme, getHighlightExtension }`
- [x] `useMdHighlight` → `useSyntaxHighlight`
- [x] 逻辑从 `.md 始终高亮` 简化为 toggle 全局控制
- [x] 存储键 `md_highlight_plain` → `cm_syntax_highlight`
- [x] 扩展数组调用使用 `getHighlightExtension(fileExt)`
- [x] `initCodeMirror()` 签名参数名已更新 + 新增 `fileExt` 参数

## 设置页面
- [x] 标签文案已更新为「启用 CM6 语法高亮」
- [x] 提示文案已更新为「关闭后所有笔记不再显示代码语法高亮」

## 运行时验证
- [ ] `.md` 笔记：标题/加粗/斜体/链接/代码块/引用/列表/水平线 高亮正常
- [ ] `.js` 笔记：关键字/字符串/数字/函数名/注释 高亮正常
- [ ] `.py` 笔记：def/class/import/字符串/注释 高亮正常
- [ ] `.go` 笔记（legacy-modes 兜底）：关键字/字符串/注释 高亮正常
- [ ] 未知扩展名：无高亮，纯文本
- [ ] Toggle 关闭 → 任何笔记无高亮；Toggle 开启 → 恢复高亮
- [ ] 切换 6 套主题 → 所有高亮颜色跟随 CSS 变量变化
- [ ] Vite dev 无 console 错误