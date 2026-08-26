# Checklist

## 数据层
- [ ] PasswordRecord 模型定义正确，包含所有必要字段和索引
- [ ] AllModels 中已注册 PasswordRecord 模型
- [ ] PasswordService 实现了完整的 CRUD 操作

## 加密存储
- [ ] 创建/更新密码记录时，password 字段调用 `EncodeB64()` 编码
- [ ] 查询密码记录时，password 字段调用 `DecodeB64()` 解码
- [ ] 兼容存量无 `(zk)` 前缀的明文密码（DecodeB64 原样返回）

## Wails 绑定
- [ ] app.go 中添加了 PasswordService 字段和所有 Wails 绑定方法

## 前端模块
- [ ] 创建独立 `password-manager.js` 模块文件
- [ ] index.html 中添加密码管理视图容器
- [ ] main.js 中 import 并初始化 password-manager 模块
- [ ] main.js switchView 中注册 password-manager 路由

## 菜单入口
- [ ] 更多菜单中添加了密码管理入口
- [ ] 启动器中添加了密码管理入口

## 视图交互
- [ ] 表格正确展示所有字段（名称、用户名、密码、URL、备注）
- [ ] 密码列默认掩码显示，点击可切换
- [ ] 添加对话框包含所有输入字段
- [ ] 编辑对话框可修改已有记录
- [ ] 表单校验：名称和用户名必填，为空时提示
- [ ] 删除操作有确认提示
- [ ] 复制密码功能正常
- [ ] URL 可点击在浏览器中打开
- [ ] 搜索功能支持模糊匹配
