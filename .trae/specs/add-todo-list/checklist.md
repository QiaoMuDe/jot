# Checklist

## 后端模型与服务
- [x] `internal/models/todo.go` 存在，定义 Todo 结构体
- [x] `internal/services/todo_service.go` 存在，实现 CRUD 方法
- [x] `internal/database/db.go` 的 AutoMigrate 包含 `&models.Todo{}`
- [x] `app.go` 中 `todoService` 初始化并暴露 5 个绑定方法

## HTML 结构
- [x] `#viewTodo` 视图存在于 index.html 中
- [x] "待办清单"菜单项存在于 `#moreMenu` 中 AI 助手下方，`data-action="todo"`
- [x] todo.css 被引入 index.css

## 视觉样式
- [x] todo.css 文件存在且包含所有样式定义
- [x] 自定义圆形复选框使用 `--accent` 颜色
- [x] 已完成项有删除线和降低透明度
- [x] 添加动画（todoEnter, todoExit）已定义
- [x] 空状态有友好提示
- [x] 所有主题下可读性良好（使用 CSS 变量）

## 功能逻辑
- [x] 从"更多"菜单点击可切换到待办清单视图
- [x] 可按 Enter 添加待办项（后端创建）
- [x] 可点击复选框完成/取消完成（后端切换）
- [x] 可删除待办项（后端删除）
- [x] 可双击编辑待办项文本（后端更新）
- [x] 筛选按钮（全部/待办/已完成）正常工作
- [x] 统计栏显示正确数据
- [x] 数据通过后端 SQLite 持久化

## 用户体验
- [x] 切换视图时有正确的进入动画
- [x] 返回按钮可回到笔记首页
- [x] 添加/完成/删除操作有平滑动画反馈
- [x] 待办项排序正确（未完成在前，按创建时间倒序）