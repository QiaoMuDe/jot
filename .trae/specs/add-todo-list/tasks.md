# Tasks

- [x] Task 1: 创建后端 Todo 数据模型和数据库迁移
  - 在 `internal/models/todo.go` 中定义 Todo 结构体（ID, Text, Done, CreatedAt, UpdatedAt）
  - 在 `internal/database/db.go` 的 AutoMigrate 中添加 `&models.Todo{}`
  - 参考 Note 模型的 GORM 标签模式

- [x] Task 2: 创建 TodoService 后端服务层
  - 在 `internal/services/todo_service.go` 中实现 TodoService
  - 方法：Create(text), List(), Toggle(id), Delete(id), Update(id, text)
  - 参考 NoteService 的服务模式

- [x] Task 3: 在 app.go 中添加 Todo CRUD 绑定方法
  - 在 App struct 中添加 `todoService *services.TodoService`
  - 在 NewApp() 中初始化 `services.NewTodoService(db)`
  - 在 rebuildServices() 中重建 todoService
  - 暴露绑定方法：CreateTodo, ListTodos, ToggleTodo, DeleteTodo, UpdateTodo

- [x] Task 4: 在 index.html 中添加待办清单视图和菜单入口
  - 在 `#moreMenu` 中的"AI 助手"下方添加 `data-action="todo"` 菜单项
  - 创建 `#viewTodo` 视图结构（view-header + 返回按钮 + 输入区 + 列表区 + 筛选栏 + 统计栏）
  - 在 index.css 中引入 todo.css

- [x] Task 5: 创建 todo.css 待办清单样式文件
  - 设计温暖的纸质笔记本风格样式
  - 自定义圆形复选框（使用 --accent CSS 变量）
  - 输入框、列表项、筛选按钮、统计栏样式
  - 动画关键帧（todoEnter, todoExit）
  - 空状态和编辑模式样式

- [x] Task 6: 在 main.js 中添加待办清单功能逻辑
  - 在 `switchView()` 中添加 `todo` 视图映射和加载逻辑
  - 在更多菜单点击事件中添加 `data-action="todo"` 处理
  - 实现 TodoManager 模块，通过 Wails 调用后端：
    - loadTodos() → ListTodos
    - addTodo(text) → CreateTodo
    - toggleTodo(id) → ToggleTodo
    - deleteTodo(id) → DeleteTodo
    - editTodo(id, newText) → UpdateTodo
    - renderTodos(filter) — 渲染列表
    - updateStats() — 更新统计

# Task Dependencies
- [Task 1] (model) → [Task 2] (service) → [Task 3] (bindings) — 后端链
- [Task 4] (HTML) → [Task 5] (CSS) — 前端结构链
- [Task 6] (JS logic) depends on [Task 3] + [Task 5] — 前后端对接