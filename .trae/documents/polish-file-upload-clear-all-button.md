# 优化上传文件"批量清除"按钮文案

## Summary

上传文件 chips 栏中，3 个文件以上出现的批量清除按钮文案"清除所有上传文件"过于生硬，需要改得更自然，与引用笔记的"移除全部 N 篇"风格对齐。

## Current State

- 引用笔记的批量移除按钮文案：`移除全部 ${referencedNotes.length} 篇`（如"移除全部 3 篇"），title 为"一键移除全部引用"
- 上传文件的批量清除按钮文案：`清除所有上传文件`（固定文字，无动态计数），title 为"清除所有上传文件"

两者使用完全相同的 CSS 类（`.ai-chat-ref-chip-remove-all`）和 SVG 图标，仅文案不同。

## Proposed Changes

### 文件：`frontend/src/js/ai-chat.js`

修改 `renderFileChips()` 函数中批量清除按钮的文案（仅 2 行变化）：

| 当前 | 改为 |
|------|------|
| `title="清除所有上传文件"` | `title="一键移除全部文件"` |
| `<span>清除所有上传文件</span>` | `<span>移除全部 ${uploadedFiles.length} 个</span>` |

与引用笔记的 `移除全部 ${referencedNotes.length} 篇` 保持一致模式：
- 引用笔记用"篇"（笔记单位）
- 上传文件用"个"（文件单位）
- 都有动态计数，让人一眼知道要移除多少个

### 不涉及修改的文件

- `app.go` — 后端逻辑无变化
- `index.html` — HTML 结构无变化
- `ai-chat.css` — 样式已正确复用，无变化

## Verification

1. 读取 `ai-chat.js` 中 `renderFileChips()` 函数，确认文案已更新
2. 确保 `go build ./...` 编译通过（仅改前端 JS，后端不受影响）
