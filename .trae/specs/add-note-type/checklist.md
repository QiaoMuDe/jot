# Checklist

- [x] Note 模型 `NoteType` 字段已添加
- [x] AutoMigrate 自动处理新增列，无需手动 SQL
- [x] 新建笔记默认选中「纯文本」（text）
- [x] 新建笔记时底部「编辑/预览」切换按钮隐藏
- [x] 编辑器中切换「Markdown」→ 底部显示「编辑/预览」切换按钮
- [x] 编辑器中切换「纯文本」→ 底部隐藏「编辑/预览」切换按钮
- [x] 编辑已有笔记：`note_type=""` 时按纯文本处理，隐藏切换按钮
- [x] 编辑已有笔记：`note_type="text"` 时隐藏切换按钮
- [x] 编辑已有笔记：`note_type="markdown"` 时显示切换按钮
- [x] 查看纯文本笔记：内容以 textarea readonly 原始显示，跳过 Markdown 渲染
- [x] 查看 Markdown 笔记（`note_type="markdown"`）：走现有渲染流程
- [x] 查看 `note_type=""` 的旧笔记：默认按 text 处理，原始文本显示
- [x] 创建笔记时 `note_type` 字段正确保存
- [x] 更新笔记时 `note_type` 字段正确保存
- [x] 类型切换即时生效，无额外确认弹窗
