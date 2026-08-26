# 密码管理功能 Spec

## Why
用户需要一个本地密码管理器来存储和管理各类账号密码信息，方便快速查看、复制和管理。

## What Changes
- 新增 `PasswordRecord` 数据模型
- 新增 `PasswordService` 服务层（CRUD 操作 + 密码加解密，构造函数签名 `NewPasswordService(db *gorm.DB, logger ...)` 与现有服务一致）
- 复用 `crypto.go` 中的 `EncodeB64`/`DecodeB64` 进行密码字段的编解码存储
- 新增 Wails 绑定方法
- 新增密码管理视图（列表展示 + 右键菜单 + 查看详情对话框 + 添加/编辑对话框 + 批量操作）
- 新增 `password-manager.js` 前端独立模块
- 在更多菜单和启动器中新增入口

## Impact
- Affected specs: 无（新功能）
- Affected code: 
  - `internal/models/` - 新增模型文件
  - `internal/database/models.go` - 注册新模型
  - `internal/services/` - 新增服务文件
  - `internal/services/crypto.go` - 复用 EncodeB64/DecodeB64（无需修改）
  - `main.go` - 创建 PasswordService 实例并赋值给 App
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
- **NOTE** 列表接口仅返回名称、用户名、URL，不解码密码字段

#### Scenario: 查询单条密码记录
- **WHEN** 用户右键点击列表项并选择"查看详情"
- **THEN** 系统根据 ID 返回该条记录的全部字段（含密码明文，已解码）

#### Scenario: 搜索密码记录
- **WHEN** 用户输入搜索关键词
- **THEN** 系统在名称、用户名、URL、备注字段中模糊搜索
- **NOTE** 搜索接口不解码密码字段

#### Scenario: 更新密码记录
- **WHEN** 用户修改记录信息并点击保存
- **THEN** 系统更新对应记录

#### Scenario: 删除密码记录
- **WHEN** 用户点击删除按钮并确认
- **THEN** 系统软删除该记录

#### Scenario: 批量删除密码记录
- **WHEN** 系统接收到批量删除请求（传入 ID 列表）
- **THEN** 系统对所有传入的 ID 执行软删除

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
系统 SHALL 提供密码管理视图用于展示和管理密码记录。前端逻辑封装在独立模块 `password-manager.js` 中。密码不在主列表中显示，所有操作通过右键菜单触发。

#### Scenario: 视图展示
- **WHEN** 用户进入密码管理页面
- **THEN** 页面布局从上到下依次为：操作栏 → 列表区域
- **AND** 操作栏布局：左侧为搜索框（占剩余空间），右侧为"添加"按钮和"批量操作"按钮
- **AND** 列表每个条目为三栏布局，等宽分配：
  - 左侧：名称（主文字，粗体 14px）
  - 中间：用户名（次要文字，灰色 12px）
  - 右侧：URL（次要文字，灰色 12px，超长以 `text-overflow: ellipsis` 截断，hover tooltip 显示完整内容）
- **AND** 密码和备注不出现在主列表中，备注仅在查看详情对话框中展示
- **AND** 列表为空时显示空状态提示："还没有密码记录，点击'添加'按钮创建第一条"

#### Scenario: 添加/编辑对话框
- **WHEN** 用户点击操作栏"添加"按钮，或右键点击列表项选择"编辑"
- **THEN** 弹出居中对话框，半透明遮罩覆盖页面，对话框布局如下：
  - 标题栏：添加时显示"添加密码记录"，编辑时显示"编辑密码记录"，右侧有关闭按钮
  - 表单区域（垂直排列）：
    - 名称：文本输入框（必填，placeholder: "如 GitHub、淘宝"）
    - 用户名：文本输入框（必填，placeholder: "如 zhangsan@gmail.com"）
    - 密码：密码输入框（必填，左侧显示，右侧有"显示/隐藏"切换按钮）
    - URL：文本输入框（可选，placeholder: "如 https://github.com"）
    - 备注：多行文本框（可选，placeholder: "补充说明"，3 行高度）
  - 操作栏：左侧"取消"按钮，右侧"保存"按钮

#### Scenario: 查看详情对话框
- **WHEN** 用户右键点击列表项并选择"查看详情"
- **THEN** 弹出居中只读对话框，布局如下：
  - 标题栏：显示记录名称，右侧有关闭按钮
  - 信息区域（垂直排列，每行标签 + 值）：
    - 名称：显示值
    - 用户名：显示值 + 右侧"复制"按钮
    - 密码：默认显示掩码 `••••••`，右侧有"显示/隐藏"切换按钮 + "复制"按钮
    - URL：显示值 + 右侧"复制"按钮和"打开"按钮（URL 非空时，点击调用 `runtime.BrowserOpenURL` 在默认浏览器中打开）
    - 备注：显示值（为空时显示"-"）
    - 创建时间：格式化显示
    - 更新时间：格式化显示
  - 底部操作栏："编辑"按钮和"删除"按钮

#### Scenario: 表单校验
- **WHEN** 用户提交添加/编辑表单
- **THEN** 系统校验名称、用户名和密码为必填项，为空时提示错误

#### Scenario: 实时输入长度校验与截断
- **WHEN** 用户在添加/编辑对话框中输入内容
- **THEN** 系统对以下字段实时校验输入长度（`input` 事件触发），超出最大长度时自动截断至最大长度，并对输入框执行抖动动画 + 调用通知方法提示用户：
  - 名称（`name`）：最大 200 字符
  - 用户名（`username`）：最大 200 字符
  - 密码（`password`）：最大 500 字符
  - 网址（`url`）：最大 500 字符
- **NOTE** 备注（`note`）为 text 类型，不设长度限制

#### Scenario: 右键菜单
- **WHEN** 用户右键点击列表中的某一项
- **THEN** 显示上下文菜单，包含以下菜单项：
  - 复制用户名
  - 复制密码
  - 复制链接（仅当 URL 字段非空时显示）
  - ─── 分割线 ───
  - 查看详情
  - 编辑
  - ─── 分割线 ───
  - 删除
- **AND** 点击菜单外部或按 Escape 键关闭菜单
- **AND** 复制操作完成后调用通知方法提示"已复制"

#### Scenario: 进入批量模式
- **WHEN** 用户点击操作栏右侧的"批量操作"按钮
- **THEN** 列表进入批量模式：每个列表项左侧显示复选框，搜索框旁按钮变为"退出批量"，底部出现批量操作栏

#### Scenario: 选择条目
- **WHEN** 用户在批量模式下点击列表项的复选框
- **THEN** 该条目被选中（复选框勾选 + 行高亮）
- **AND** 底部操作栏实时显示"已选 N 项"

#### Scenario: 全选
- **WHEN** 用户在批量模式下点击底部操作栏的"全选"复选框
- **THEN** 选中当前搜索结果中的所有条目
- **AND** 再次点击取消全选

#### Scenario: 批量删除
- **WHEN** 用户在批量模式下选中条目后点击底部操作栏的"删除"按钮
- **THEN** 弹出确认对话框（显示"确定删除 N 条记录？"），确认后批量软删除选中记录

#### Scenario: 退出批量模式
- **WHEN** 用户点击"退出批量"按钮
- **THEN** 退出批量模式，清空所有选择，隐藏复选框和底部操作栏，恢复正常列表视图

### Requirement: 菜单入口
系统 SHALL 在更多菜单和启动器中提供密码管理入口。

#### Scenario: 更多菜单入口
- **WHEN** 用户打开更多菜单
- **THEN** 在"笔记日历"、"待办清单"、"AI 助手"同组显示"密码管理"入口

#### Scenario: 启动器入口
- **WHEN** 用户打开启动器（Ctrl+P）
- **THEN** 显示"密码管理"入口卡片
