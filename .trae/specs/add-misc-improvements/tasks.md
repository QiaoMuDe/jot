# Tasks

- [x] Task 1: 美化字体下拉菜单滚动条
  - [x] 在 style.css 中添加 `.font-family-options` 的自定义滚动条样式（::-webkit-scrollbar）
  - [x] 样式与全局滚动条统一（圆角细条、--divider 颜色、hover 变 --accent）
  - [x] 验证：下拉菜单中滚动条美观和谐

- [x] Task 2: 新增默认标签种子数据
  - [x] 在 `services/tag_service.go` 中新增 `InitDefaultTags(db)` 方法
  - [x] 方法逻辑：检测标签表是否为空，为空则插入默认标签（待办、工作、生活、个人、学习、重要）
  - [x] 在 `database/db.go` 中 `InitDB()` 返回前调用 `InitDefaultTags()`
  - [x] 验证：清理数据库后重启，标签列表出现默认 6 个标签

- [x] Task 3: 新增快捷键说明页面
  - [x] 在 `index.html` 中新增 `shortcuts-view` 视图，包含快捷键表格和关闭按钮
  - [x] 在 `main.js` 中添加 `renderShortcutsPage()` 渲染快捷键表格
  - [x] 在 `main.js` 中 `switchView()` 添加 `shortcuts` 分支
  - [x] 在关于页面中添加 `?` 按钮，点击切换到快捷键视图
  - [x] 在 `style.css` 中添加快捷键页样式（卡片风格，快捷键表格布局）
  - [x] 添加键盘快捷键：Escape 从快捷键页返回
  - [x] 验证：进入关于页，点击 `?` 显示快捷键列表，Escape 关闭返回

# Task Dependencies
- 无依赖，三个任务可并行执行
