# AI 引用笔记总长度改为 10MB

## 概述

将 `BuildNoteRefContext` 中硬编码的总上下文长度上限从 8000 字节改为 10MB。

## 改动

仅修改一个文件：

**`internal/services/note_service.go:141`**

```
- const maxTotalChars = 8000
+ const maxTotalChars = 10 * 1024 * 1024
```

## 验证

启动应用，引用多篇长笔记发送 AI 对话，确认不再受旧的 8000 字节限制，能正常包含更多内容。