# 密码管理功能 Spec

## Why
用户需要一个本地密码管理器来存储和管理各类账号密码信息，方便快速查看、复制和管理。

## What Changes
- 新增 `PasswordRecord` 数据模型
- 新增 `PasswordService` 服务层（CRUD 操作 + 密码加解密）
- 复用 `crypto.go` 中的 `EncodeB64`/`DecodeB64` 进行密码字段的编解码存储
- 新增 Wails 绑定方法
- 新增密码管理视图（表格展示 + 添加/编辑对话框）
- 新增 `password-manager.js` 前端独立模块
- 在更多菜单和启动器中新增入口

## Impact
- Affected specs: 无（新功能）
- Affected code: 
  - `internal/models/` - 新增模型文件
  - `internal/database/models.go` - 注册新模型
  - `internal/services/` - 新增服务文件
  - `internal/services/crypto.go` - 复用 EncodeB64/DecodeB64（无需修改）
  - `app.go` - 新增 Wails 绑定方法
  - `frontend/index.html` - 新增视图容器 + 菜单项
  - `frontend/src/js/password-manager.js` - 新增独立模块
  - `frontend/src/js/launcher.js` - 新增启动器入口
  - `frontend/src/main.js` - import 模块 + switchView 路由
  - `frontend/src/css/components/` - 新增样式文件

## ADDED Requirements

### Requirement: Password Record 数据模型
系统 SHALL 提供密码记录数据模型用于存储密码信息。

#### Scenario: 数据模型结构
- **WHEN** 系统初始化
- **THEN** 创建 `password_records` 表，包含以下字段：
  - `id` (uint, 主键)
  - `name` (string, 长度200, 必填) - 名称（如"GitHub"、"淘宝"）
  - `username` (string, 长度200, 必填) - 用户名
  - `password` (string, 长度500, 必填) - 密码
  - `url` (string, 长度500) - 网址（可选）
  - `note` (text) - 备注（可选）
  - `created_at` (time) - 创建时间
  - `updated_at` (time) - 更新时间
  - `deleted_at` (time, 索引) - 软删除时间

### Requirement: 密码记录 CRUD 操作
系统 SHALL 提供密码记录的增删改查操作。

#### Scenario: 创建密码记录
- **WHEN** 用户填写名称、用户名、密码、URL、备注并点击保存
- **THEN** 系统创建新记录并返回

#### Scenario: 查询密码记录列表
- **WHEN** 用户进入密码管理页面
- **THEN** 系统返回所有密码记录（按创建时间倒序）

#### Scenario: 搜索密码记录
- **WHEN** 用户输入搜索关键词
- **THEN** 系统在名称、用户名、URL、备注字段中模糊搜索

#### Scenario: 更新密码记录
- **WHEN** 用户修改记录信息并点击保存
- **THEN** 系统更新对应记录

#### Scenario: 删除密码记录
- **WHEN** 用户点击删除按钮并确认
- **THEN** 系统软删除该记录

### Requirement: 密码安全存储
系统 SHALL 使用 `(zk)` 前缀 + Base64 编码方式存储密码字段，复用现有 `crypto.go` 中的 `EncodeB64`/`DecodeB64` 函数。

#### Scenario: 写入时编码
- **WHEN** 系统创建或更新密码记录
- **THEN** 对 `password` 字段调用 `services.EncodeB64()` 编码后存入数据库

#### Scenario: 读取时解码
- **WHEN** 系统查询密码记录（单条或列表）
- **THEN** 对 `password` 字段调用 `services.DecodeB64()` 解码后返回前端

#### Scenario: 兼容存量明文
- **WHEN** 数据库中存在旧版无 `(zk)` 前缀的明文密码
- **THEN** `DecodeB64()` 应原样返回明文，不报错（与现有 API Key 迁移行为一致）

### Requirement: 密码管理视图
系统 SHALL 提供密码管理视图用于展示和管理密码记录。前端逻辑封装在独立模块 `password-manager.js` 中。

#### Scenario: 视图展示
- **WHEN** 用户进入密码管理页面
- **THEN** 显示表格，列包含：名称、用户名、密码（掩码）、URL、备注、操作

#### Scenario: 密码掩码显示
- **WHEN** 表格展示密码列
- **THEN** 默认显示 `••••••`，点击可切换显示/隐藏

#### Scenario: 添加/编辑对话框
- **WHEN** 用户点击"添加"或编辑按钮
- **THEN** 弹出对话框，包含名称、用户名、密码、URL、备注输入框

#### Scenario: 表单校验
- **WHEN** 用户提交添加/编辑表单
- **THEN** 系统校验名称和用户名为必填项，为空时提示错误

#### Scenario: 复制密码
- **WHEN** 用户点击复制按钮
- **THEN** 密码复制到剪贴板

#### Scenario: 打开链接
- **WHEN** 用户点击 URL 列的链接
- **THEN** 系统调用 `runtime.BrowserOpenURL` 在默认浏览器中打开该链接

### Requirement: 菜单入口
系统 SHALL 在更多菜单和启动器中提供密码管理入口。

#### Scenario: 更多菜单入口
- **WHEN** 用户打开更多菜单
- **THEN** 在"笔记日历"、"待办清单"、"AI 助手"同组显示"密码管理"入口

#### Scenario: 启动器入口
- **WHEN** 用户打开启动器（Ctrl+P）
- **THEN** 显示"密码管理"入口卡片
