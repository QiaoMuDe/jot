# Checklist

- [x] index.html 中已新增 `#aiMsgContextMenu` 元素
- [x] 右键菜单使用现有 `.context-menu` / `.context-menu-item` / `.context-menu-divider` 样式
- [x] `showAiMsgContextMenu()` 函数实现：用户消息显示「复制」，AI 回复显示「复制」「保存为笔记」「重新生成」「追问此条回复」
- [x] `hideAiMsgContextMenu()` 函数实现：点击菜单外部 / Escape 键 / 点击菜单项关闭
- [x] 菜单位置跟随鼠标坐标，不溢出视口
- [x] 用户消息气泡绑定 `contextmenu` 事件
- [x] AI 回复消息气泡绑定 `contextmenu` 事件
- [x] e.preventDefault() 阻止浏览器默认菜单
- [x] 右键菜单中的「复制」功能与悬浮按钮一致
- [x] 右键菜单中的「保存为笔记」功能与悬浮按钮一致
- [x] 右键菜单中的「重新生成」功能与悬浮按钮一致
- [x] 右键菜单中的「追问此条回复」功能与悬浮按钮一致
- [x] Wails dev 构建无错误
