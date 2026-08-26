# Tasks

- [ ] Task 1: 创建密码记录数据模型
  - [ ] SubTask 1.1: 创建 `internal/models/password_record.go` 文件
  - [ ] SubTask 1.2: 在 `internal/database/models.go` 注册新模型到 AllModels

- [ ] Task 2: 创建密码记录服务层
  - [ ] SubTask 2.1: 创建 `internal/services/password_service.go` 文件
  - [ ] SubTask 2.2: 实现 Create 方法（创建密码记录）
  - [ ] SubTask 2.3: 实现 List 方法（查询密码记录列表）
  - [ ] SubTask 2.4: 实现 Search 方法（搜索密码记录）
  - [ ] SubTask 2.5: 实现 Update 方法（更新密码记录）
  - [ ] SubTask 2.6: 实现 Delete 方法（删除密码记录）

- [ ] Task 3: 创建 Wails 绑定方法
  - [ ] SubTask 3.1: 在 app.go 中添加 PasswordService 字段
  - [ ] SubTask 3.2: 实现 CreatePasswordRecord 方法
  - [ ] SubTask 3.3: 实现 ListPasswordRecords 方法
  - [ ] SubTask 3.4: 实现 SearchPasswordRecords 方法
  - [ ] SubTask 3.5: 实现 UpdatePasswordRecord 方法
  - [ ] SubTask 3.6: 实现 DeletePasswordRecord 方法

- [ ] Task 4: 创建前端密码管理视图
  - [ ] SubTask 4.1: 在 index.html 中添加密码管理视图容器 `#viewPasswordManager`
  - [ ] SubTask 4.2: 创建 `frontend/src/css/components/password-manager.css` 样式文件
  - [ ] SubTask 4.3: 在 index.html 的 CSS 入口引入新样式文件
  - [ ] SubTask 4.4: 在 main.js 中添加 switchView 路由
  - [ ] SubTask 4.5: 实现密码管理视图渲染逻辑（表格）
  - [ ] SubTask 4.6: 实现添加/编辑对话框
  - [ ] SubTask 4.7: 实现删除确认对话框
  - [ ] SubTask 4.8: 实现密码掩码切换功能
  - [ ] SubTask 4.9: 实现复制到剪贴板功能

- [ ] Task 5: 添加菜单入口
  - [ ] SubTask 5.1: 在 index.html 更多菜单中添加密码管理入口（ai-chat 之后）
  - [ ] SubTask 5.2: 在 launcher.js 中添加密码管理启动器入口

- [ ] Task 6: 集成测试
  - [ ] SubTask 6.1: 验证数据库表自动创建
  - [ ] SubTask 6.2: 验证 CRUD 功能正常
  - [ ] SubTask 6.3: 验证视图展示和交互正常

# Task Dependencies
- Task 2 depends on Task 1
- Task 3 depends on Task 2
- Task 4 depends on Task 3
- Task 5 depends on Task 4
- Task 6 depends on Task 5
