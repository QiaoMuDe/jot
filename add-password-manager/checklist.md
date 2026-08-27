# Checklist

## 数据层
- [ ] PasswordRecord 模型定义正确，包含所有必要字段和索引
- [ ] AllModels 中已注册 PasswordRecord 模型
- [ ] PasswordService 实现了完整的 CRUD 操作（含 GetPasswordRecord 单条查询）

## 加密存储
- [ ] 创建/更新密码记录时，password 字段调用 `EncodeB64()` 编码
- [ ] 查询密码记录时，password 字段调用 `DecodeB64()` 解码
- [ ] 兼容存量无 `(zk)` 前缀的明文密码（DecodeB64 原样返回）

## Wails 绑定
- [ ] app.go 中添加了 PasswordService 字段和所有 Wails 绑定方法（含 GetPasswordRecord）
- [ ] app.go 的 `NewApp()` 中调用 `NewPasswordService(db, logSvc.Logger)` 创建实例并赋值给 App
- [ ] app.go 的 `rebuildServices()` 中同步重建了 PasswordService（防止数据库重置后持有旧连接）

## 前端模块
- [ ] 创建独立 `password-manager.js` 模块文件
- [ ] index.html 中添加密码管理视图容器
- [ ] main.js 中 import 并初始化 password-manager 模块
- [ ] main.js switchView 中注册 password-manager 路由

## 菜单入口
- [ ] 更多菜单中添加了密码管理入口
- [ ] 启动器中添加了密码管理入口

## 视图交互
- [ ] 操作栏：搜索框在左侧（占剩余空间），添加按钮和批量操作按钮在右侧
- [ ] 列表项三栏等宽布局：左侧名称(主文字)、中间用户名(次要)、右侧 URL(次要)
- [ ] URL 超长时 `text-overflow: ellipsis` 截断，hover tooltip 显示完整内容
- [ ] 密码和备注不出现在主列表中
- [ ] 列表为空时显示空状态提示
- [ ] 搜索框支持在名称、用户名、URL、备注中模糊搜索
- [ ] 右键菜单包含：复制用户名、复制密码、复制链接（URL 非空时）、查看详情、编辑、删除
- [ ] 右键菜单有分隔线分组，点击外部或按 Escape 可关闭
- [ ] 复制操作完成后有通知提示
- [ ] 查看详情对话框：标题栏(记录名称) + 信息区(名称/用户名+复制/密码+掩码切换+复制/URL+复制+打开/备注/时间) + 底部(编辑/删除)
- [ ] 添加/编辑对话框：标题栏 + 表单区(名称/用户名/密码+显示隐藏/URL/备注) + 底部(取消/保存)
- [ ] 密码输入框有显示/隐藏切换按钮
- [ ] 表单校验：名称、用户名和密码必填，为空时提示
- [ ] 实时长度校验：名称(200)、用户名(200)、密码(500)、URL(500) 超长时截断 + 抖动 + 通知提示
- [ ] 删除操作通过右键菜单触发，有确认提示

## 批量操作
- [ ] 批量操作按钮在操作栏右侧按钮组内，点击进入/退出批量模式
- [ ] 批量模式下列表项左侧显示复选框，可勾选/取消，选中行高亮
- [ ] 底部批量操作栏包含：全选复选框 + 已选计数 + 删除按钮
- [ ] 全选操作选中当前搜索结果所有条目，再点取消全选
- [ ] 批量删除有确认对话框（显示删除数量），确认后调用 BatchDeletePasswordRecords
- [ ] 退出批量模式时清空选择、隐藏复选框和操作栏
