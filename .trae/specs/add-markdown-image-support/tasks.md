# Tasks

- [x] Task 1: Go 后端 — 图片存储目录初始化 + 图片上传 API + AssetServer Handler
  - 修改 `app.startup()` 在 `~/.jot/images/` 目录不存在时自动创建
  - 在 `app.go` 新增 `SaveImage(name string, data []byte) (string, error)` 方法（Wails Bind 自动暴露）：
    - 生成 `uuid_原始文件名` 避免命名冲突
    - 写入 `~/.jot/images/` 目录
    - 返回 `/images/uuid_文件名.ext` 访问路径
  - 修改 `main.go` 的 `AssetServer` 配置：
    - 增加 `Handler` 字段，注册 `http.Handler`
    - 拦截路径以 `/images/` 开头的请求，`http.StripPrefix` + `http.FileServer` 从 `~/.jot/images/` 提供文件服务
    - 非 `/images/` 路径直接 `http.NotFound`（由 Wails 回退到 embed FS）

- [x] Task 2: 前端 CodeMirror 6 — 粘贴图片上传与插入
  - 在 `main.js` 中找到 CodeMirror 编辑器初始化后的粘贴事件入口
  - 检测条件：笔记后缀为 `.md` + 用户为编辑模式（非只读）
  - 读取 `e.clipboardData.files`，过滤 `type.startsWith('image/')` 的图片文件
  - 对每个图片文件：
    - 调用 `window.go.main.App.SaveImage(file.name, arrayBuffer)` 上传
    - 获取返回的 URL 路径（含 `/images/` 前缀）
    - 通过 CodeMirror 6 的 `view.dispatch` 在光标位置插入 `![](/images/uuid_name.ext)`
  - 非图片粘贴（纯文本等）不影响原有粘贴行为
  - 非 `.md` 笔记粘贴图片不影响（保留默认行为）

- [ ] Task 3: 验证图片在所有三种模式中正常显示（需手动运行应用验证）
  - 验证查看模式（只读 `.md` 笔记）：`![图](/images/xxx.jpg)` 在预览区正常显示
  - 验证编辑模式预览切换：编辑模式下切"预览"图片正常显示
  - 验证新建 `.md` 笔记后粘贴图片→保存→重新打开→预览显示完整链路
  - 验证 `.txt` 笔记粘贴图片不影响（不触发上传）

# Task Dependencies

- Task 1 是 Task 2 和 Task 3 的前置依赖（后端须先就绪，前端粘贴才能上传成功）
- Task 2 完成后 Task 3 方可完整验证
