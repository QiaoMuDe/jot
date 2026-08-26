# Tasks

- [ ] Task 1: 创建密码记录数据模型
  - [ ] SubTask 1.1: 创建 `internal/models/password_record.go` 文件
  - [ ] SubTask 1.2: 在 `internal/database/models.go` 注册新模型到 AllModels

- [ ] Task 2: 创建密码记录服务层
  - [ ] SubTask 2.1: 创建 `internal/services/password_service.go` 文件（含构造函数 `NewPasswordService(db *gorm.DB, logger ...)`）
  - [ ] SubTask 2.2: 实现 Create 方法（创建密码记录，password 字段调用 EncodeB64 编码）
  - [ ] SubTask 2.3: 实现 List 方法（查询密码记录列表，仅返回名称、用户名、URL，不解码密码）
  - [ ] SubTask 2.4: 实现 Search 方法（在名称、用户名、URL、备注中模糊搜索，不解码密码）
  - [ ] SubTask 2.5: 实现 GetPasswordRecord 方法（根据 ID 查询单条记录，password 字段调用 DecodeB64 解码）
  - [ ] SubTask 2.6: 实现 Update 方法（更新密码记录，password 字段调用 EncodeB64 编码）
  - [ ] SubTask 2.7: 实现 Delete 方法（软删除密码记录）
  - [ ] SubTask 2.8: 实现 BatchDelete 方法（根据 ID 列表批量软删除密码记录）

- [ ] Task 3: 创建 Wails 绑定方法
  - [ ] SubTask 3.1: 在 app.go 中添加 PasswordService 字段
  - [ ] SubTask 3.2: 在 main.go 中调用 `NewPasswordService(db, logger)` 创建实例并赋值给 App
  - [ ] SubTask 3.3: 实现 CreatePasswordRecord 方法
  - [ ] SubTask 3.4: 实现 GetPasswordRecord 方法（根据 ID 查询单条，含密码解码）
  - [ ] SubTask 3.5: 实现 ListPasswordRecords 方法
  - [ ] SubTask 3.6: 实现 SearchPasswordRecords 方法
  - [ ] SubTask 3.7: 实现 UpdatePasswordRecord 方法
  - [ ] SubTask 3.8: 实现 DeletePasswordRecord 方法
  - [ ] SubTask 3.9: 实现 BatchDeletePasswordRecords 方法（接收 ID 列表，调用 BatchDelete）

- [ ] Task 4: 创建前端密码管理视图
  - [ ] SubTask 4.1: 在 index.html 中添加密码管理视图容器 `#viewPasswordManager`
  - [ ] SubTask 4.2: 创建 `frontend/src/css/components/password-manager.css` 样式文件
  - [ ] SubTask 4.3: 在 index.html 的 CSS 入口引入新样式文件
  - [ ] SubTask 4.4: 创建 `frontend/src/js/password-manager.js` 独立模块（参考 calendar.js 模式）
  - [ ] SubTask 4.5: 在 main.js 中 import 并初始化 password-manager 模块
  - [ ] SubTask 4.6: 在 main.js switchView 中注册 password-manager 路由
  - [ ] SubTask 4.7: 实现列表视图渲染（三栏等宽布局：左侧名称、中间用户名、右侧 URL；操作栏搜索框在左、按钮在右）+ URL 超长 tooltip
  - [ ] SubTask 4.8: 实现列表空状态（无数据时显示提示文字 + 引导）
  - [ ] SubTask 4.9: 实现搜索框（支持在名称、用户名、URL、备注中模糊搜索）
  - [ ] SubTask 4.10: 实现批量操作按钮（搜索框旁，点击进入/退出批量模式）
  - [ ] SubTask 4.11: 实现批量模式下列表项复选框（勾选/取消 + 行高亮）
  - [ ] SubTask 4.12: 实现底部批量操作栏（全选复选框 + 已选计数 + 删除按钮）
  - [ ] SubTask 4.13: 实现全选逻辑（选中当前搜索结果所有条目，再点取消全选）
  - [ ] SubTask 4.14: 实现批量删除（确认对话框 + 调用 BatchDeletePasswordRecords + 刷新列表）
  - [ ] SubTask 4.15: 实现退出批量模式（清空选择、隐藏复选框和操作栏、恢复正常视图）
  - [ ] SubTask 4.16: 在 password-manager.css 中添加右键菜单样式（含分隔线）+ 批量操作栏样式 + 复选框样式 + 对话框样式
  - [ ] SubTask 4.17: 实现右键菜单（菜单项：复制用户名、复制密码、复制链接、查看详情、编辑、删除；分隔线分组；点击外部/Esc 关闭）
  - [ ] SubTask 4.18: 实现右键菜单"复制用户名"功能（复制到剪贴板 + 通知提示）
  - [ ] SubTask 4.19: 实现右键菜单"复制密码"功能（复制到剪贴板 + 通知提示）
  - [ ] SubTask 4.20: 实现右键菜单"复制链接"功能（URL 非空时显示，复制到剪贴板 + 通知提示）
  - [ ] SubTask 4.21: 实现查看详情对话框（标题栏 + 信息区：名称/用户名+复制/密码+掩码切换+复制/URL+复制+打开/备注/时间 + 底部编辑/删除按钮）
  - [ ] SubTask 4.22: 实现添加/编辑对话框（标题栏 + 表单区：名称/用户名/密码+显示隐藏/URL/备注 + 底部取消/保存；含表单校验：名称、用户名、密码必填）
  - [ ] SubTask 4.23: 定义各字段最大长度常量（name: 200, username: 200, password: 500, url: 500）
  - [ ] SubTask 4.24: 实现输入长度实时校验（`input` 事件监听，超长时截断值 + 抖动动画 + 通知提示）
  - [ ] SubTask 4.25: 在 password-manager.css 中添加输入框抖动动画样式
  - [ ] SubTask 4.26: 实现右键菜单"删除"功能（确认对话框 + 软删除）

- [ ] Task 5: 添加菜单入口
  - [ ] SubTask 5.1: 在 index.html 更多菜单中添加密码管理入口（ai-chat 之后）
  - [ ] SubTask 5.2: 在 launcher.js 中添加密码管理启动器入口

- [ ] Task 6: 集成测试
  - [ ] SubTask 6.1: 验证数据库表自动创建（password_records 表结构正确）
  - [ ] SubTask 6.2: 验证 CRUD 功能正常（创建、查询、更新、删除单条）
  - [ ] SubTask 6.3: 验证加解密存储（写入后数据库为编码值，读取后解码为明文）
  - [ ] SubTask 6.4: 验证批量删除功能正常
  - [ ] SubTask 6.5: 验证列表视图展示（三栏布局、空状态、搜索过滤）
  - [ ] SubTask 6.6: 验证右键菜单功能（复制用户名/密码/链接、查看详情、编辑、删除）
  - [ ] SubTask 6.7: 验证对话框交互（添加/编辑/查看详情对话框的打开、关闭、表单校验、实时长度校验）
  - [ ] SubTask 6.8: 验证批量操作流程（进入批量模式、全选、勾选、批量删除、退出批量模式）

# Task Dependencies
- Task 2 depends on Task 1
- Task 3 depends on Task 2
- Task 4 depends on Task 3
- Task 5 depends on Task 4
- Task 6 depends on Task 5
