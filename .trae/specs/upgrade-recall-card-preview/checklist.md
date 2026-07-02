# Checklist

- [x] openEditor 函数签名新增 hideEditBtn 参数（默认 false），hideEditBtn=true 时隐藏编辑按钮
- [x] 召回卡片点击回调改用 window.openEditor(card.id, true, false, true)
- [x] openCardPreview 函数已删除
- [x] _cardPreviewWorker 变量已删除
- [x] aiCardPreviewModal HTML 结构已删除
- [x] CSS 中 .ai-card-preview-* 样式已删除
- [x] JS 中预览浮层事件绑定已删除
