# Checklist

- [x] Go 后端：`app.startup()` 自动创建 `~/.jot/images/` 目录实现
- [x] Go 后端：`SaveImage` API 实现（UUID 防冲突、正确写入、返回 `/images/` 路径）
- [x] Go 后端：`main.go` AssetServer Handler 拦截 `/images/*` 请求提供文件服务
- [x] 前端：CodeMirror 6 粘贴事件监听（仅 `.md` 编辑模式）
- [x] 前端：图片文件上传 → 插入 `![](/images/uuid_name.ext)` 流程
- [x] 前端：非图片/非 `.md` 粘贴不受影响
- [ ] 验证：查看模式下 `.md` 笔记图片正常显示
- [ ] 验证：编辑模式预览切换图片正常显示
- [ ] 验证：新建 `.md` 笔记粘贴→保存→重开→预览完整链路
