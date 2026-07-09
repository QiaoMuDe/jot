# Checklist

## 后端
- [x] `AIChatFileResult` 结构体定义了 Path/Name/Content/Size/Truncated/Error 字段，JSON tag 正确
- [x] `SelectAIChatFiles()` 使用 `runtime.OpenMultipleFilesDialog` 打开多选对话框（title: "选择要上传的文本文件"）
- [x] 用户取消对话框时返回空数组（非 nil）
- [x] 目录被拒绝设 error
- [x] 文件大小 > 10MB 时 error = "文件过大（超过 10MB），请选择小于 10MB 的文件"
- [x] `fs.IsBinaryPath()` 检测二进制文件，error = "不支持二进制文件，请选择文本文件"
- [x] 每次读取文件时调用 `GetAIRefMaxChars()`（实时查 DB 不缓存）获取截断阈值
- [x] 内容超过阈值时截断并追加 `...(内容已截断，完整文件共 X 字)`，Truncated = true
- [x] 正常文件返回完整内容，Truncated = false

## 前端 HTML
- [x] 工具栏 `#aiChatRefBtn` 后新增了 `#aiChatFileBtn` 上传按钮（`.ai-chat-toolbar-btn` 类）
- [x] 上传按钮内含设计好的自定义上传 SVG 图标和 `<span>上传</span>` 文字
- [x] 技能指示条 `#aiChatSkillBar` 后、输入行前新增了 `#aiChatFileBar`（初始 `display:none`）和 `#aiChatFileChips`

## 前端 CSS
- [x] `.ai-chat-file-bar` 样式与 `.ai-chat-ref-bar` 一致
- [x] `.ai-chat-file-chip` 样式与 `.ai-chat-ref-chip` 一致（max-width: 260px）
- [x] `.ai-chat-file-chip-icon` / `.ai-chat-file-chip-name` / `.ai-chat-file-chip-size` 样式正确
- [x] 截断标记复用 `.ai-chat-ref-chip-trunc` 样式
- [x] 删除按钮 `.ai-chat-file-chip-remove` 与 `.ai-chat-ref-chip-remove` 一致
- [x] 批量清除按钮复用 `.ai-chat-ref-chip-remove-all` 样式

## 前端 JS
- [x] DOM 元素引用正确获取（`#aiChatFileBtn`、`#aiChatFileBar`、`#aiChatFileChips`）
- [x] 定义了 `uploadedFiles = []` 数组存储文件数据
- [x] 上传按钮 click 事件中：禁用按钮 → 调用 `SelectAIChatFiles()` → 恢复按钮
- [x] 失败文件弹出 `showNotification(error, 'error')`
- [x] 成功文件追加到 `uploadedFiles` 并调用 `renderFileChips()`
- [x] `renderFileChips()` 空列表时隐藏 `#aiChatFileBar`
- [x] 文件 chip 渲染：文件图标 + 文件名 + 大小 + 截断标记（如有）+ × 按钮
- [x] 文件数量 ≥ 3 时追加"清除所有上传文件"按钮（红色虚线边框）
- [x] 单个删除（按 index 移除）
- [x] 批量清除（清空数组 + 隐藏 bar）
- [x] 支持多次上传追加（不覆盖已有）
- [x] `onSend()` 中文件内容拼入 `systemContext`（笔记引用 → 追问引用 → 文件内容）
- [x] 拼接格式正确：`"用户上传了以下文件内容...\n\n--- 文件: name (size) ---\n{content}\n---"`
- [x] truncated 文件在内容后追加截断标记
- [x] 发送后清空 `uploadedFiles` 并隐藏 chips 栏
