# Tasks

- [x] Task 1: 后端 — 添加数据统计查询方法
  - [x] 在 `services/note_service.go` 中添加 `GetStats()` 方法，返回笔记总数、回收站数、置顶数
  - [x] 在 `services/tag_service.go` 中添加 `Count()` 方法，返回标签总数
  - [x] 在 `app.go` 中添加 `GetDataStats()` 绑定方法，聚合统计数据返回给前端

- [x] Task 2: 后端 — 添加数据导出功能
  - [x] 定义导出 JSON 结构（含笔记所有字段和标签）
  - [x] 在 `services/note_service.go` 中添加 `ExportAll()` 方法，查询所有未删除笔记并组装导出结构
  - [x] 在 `app.go` 中添加 `ExportData()` 绑定方法，返回可序列化的导出数据

- [x] Task 3: 后端 — 添加数据导入功能
  - [x] 在 `services/note_service.go` 中添加 `ImportFromJSON(data []byte)` 方法，解析 JSON 并逐条创建笔记和标签关联
  - [x] 在 `app.go` 中添加 `ImportData(jsonData string)` 绑定方法，返回导入统计（成功/失败数）

- [x] Task 4: 前端 — 数据管理页面 HTML 和 CSS
  - [x] 在 `index.html` 的更多菜单中添加「数据管理」选项
  - [x] 在 `index.html` 中新增数据管理视图（统计卡片 + 导入/导出按钮）
  - [x] 在 `style.css` 中添加数据管理页面相关样式

- [x] Task 5: 前端 — 数据管理交互逻辑
  - [x] 在 `main.js` 中添加视图切换、渲染统计数据
  - [x] 添加导出按钮事件（调用后端 API，将返回的 JSON 写入 Blob 下载）
  - [x] 添加导入按钮事件（用 `<input type="file">` 选择文件，读取后调用后端 API）

# Task Dependencies
- [Task 4] 可独立于 [Task 1/2/3] 并行开发
- [Task 5] 依赖 [Task 1/2/3] 的后端接口和 [Task 4] 的 DOM
