# 锁屏密码功能 Spec

## Why
保护用户隐私，防止他人直接打开应用后看到所有笔记内容。作为一款本地笔记应用，用户可能在公共场合或共享电脑上使用，需要一道简单的安全屏障。

## What Changes
- **后端新增**: `SettingsConfig` 增加 `screen_lock_enabled`（bool）和 `screen_lock_password`（string，存 SHA256 哈希）两个字段；新增 `VerifyScreenLockPassword(password string) bool` 绑定方法
- **前端新增**: 锁屏遮罩层（`#lockScreen`），全屏覆盖，精美动画过渡
- **设置页新增**: "锁屏密码"设置卡片，包含开关 toggle 和密码输入框
- **默认关闭**: 功能默认不开启，`screen_lock_enabled` 默认值为 `false`

## 设计概念: "雾散之门" (The Mist Gate)

### 美学方向
- **基调**: 静谧、精致、安全 — 锁屏不是一堵墙，而是一层"薄雾"
- **核心视觉**: 毛玻璃效果（backdrop-filter: blur），主界面内容隐约可见，营造"就在后面"的安全感
- **关键动画**: 输对密码时，模糊如晨雾散去（blur 8px → 0px + opacity 渐变），界面优雅地"浮现"
- **差异化记忆点**: 输错密码时输入框轻微抖动（shakeshake 动画），不是生硬的错误提示

### 交互细节
- 输入框自动获得焦点
- 回车键触发解锁
- 密码显示/隐藏切换按钮
- 锁屏适配当前主题（跟随 `data-theme`）

## Impact
- Affected specs: 设置页面、应用启动流程、视图切换
- Affected code:
  - `internal/services/types.go` — SettingsConfig 新增字段 + GetAllSettings/SaveAllSettings 处理
  - `internal/database/db.go` — InitDefaultSettings 新增默认值
  - `app.go` — 新增 VerifyScreenLockPassword 绑定方法
  - `frontend/index.html` — 新增锁屏遮罩层 HTML + 设置页卡片
  - `frontend/src/main.js` — init/loadSettings/saveSettings 修改 + 锁屏逻辑
  - `frontend/src/css/components/settings-panel.css` — 设置卡片样式
  - `frontend/src/css/components/modals.css` — 锁屏遮罩样式（或新增 lock-screen.css）

## ADDED Requirements

### Requirement: 后端密码验证
The system SHALL provide a method to verify the lock screen password.

#### Scenario: 密码验证成功
- **WHEN** 前端调用 `VerifyScreenLockPassword("正确密码")`
- **THEN** 返回 `true`

#### Scenario: 密码验证失败
- **WHEN** 前端调用 `VerifyScreenLockPassword("错误密码")`
- **THEN** 返回 `false`

#### Scenario: 无密码时的验证
- **WHEN** 锁屏密码设为空（功能关闭状态）
- **THEN** `VerifyScreenLockPassword("任意值")` 始终返回 `true`

### Requirement: 锁屏密码存储
The system SHALL store the lock screen password as a SHA-256 hash (hex-encoded), never in plaintext.

#### Scenario: 密码保存
- **WHEN** 用户设置/修改锁屏密码
- **THEN** 密码经过 SHA-256 哈希后存入 `screen_lock_password` 设置项

#### Scenario: 密码为空时不更新
- **WHEN** 保存设置时 `screen_lock_password` 字段为空字符串
- **THEN** 保持数据库中原来的密码值不变（不覆盖为空）

### Requirement: 锁屏 UI
The system SHALL display a full-screen lock overlay on startup when the feature is enabled.

#### Scenario: 启动时显示锁屏
- **WHEN** 应用启动，`loadSettings` 完成后检测到 `screen_lock_enabled = true`
- **THEN** 显示锁屏遮罩层，覆盖所有界面，且界面对用户不可交互
- **AND** 密码输入框自动获得焦点

#### Scenario: 解锁成功
- **WHEN** 用户输入正确密码并按回车或点击解锁按钮
- **THEN** 锁屏遮罩以"雾散"动画优雅消失（blur 淡出 + 向上位移 + opacity 过渡，约 400ms）
- **AND** 恢复正常应用交互

#### Scenario: 解锁失败
- **WHEN** 用户输入错误密码
- **THEN** 输入框执行 shake 抖动动画（约 400ms）
- **AND** 输入框内容被清空，重新聚焦

### Requirement: 设置页锁屏卡片
The system SHALL provide a settings card for lock screen configuration in the settings page.

#### Scenario: 锁屏设置卡片
- **WHEN** 用户进入设置页面
- **THEN** 在"回收站清理"卡片上方显示"锁屏密码"设置卡片
- **AND** 卡片包含：一个 toggle 开关（"启用锁屏密码"）+ 当开关开启时展开的密码输入框

#### Scenario: 启用开关
- **WHEN** 用户点击"启用锁屏密码"开关
- **THEN** 密码输入框以展开动画显示（高度从 0 到 auto，约 200ms ease-out）
- **AND** 调用 `saveSettings()` 保存设置

#### Scenario: 设置密码
- **WHEN** 用户在密码输入框中输入并回车、或点击保存按钮
- **THEN** 调用 `saveSettings()` 保存密码
- **AND** 底部弹出通知提示"锁屏密码已保存"

#### Scenario: 关闭功能
- **WHEN** 用户关闭 toggle 开关
- **THEN** 密码输入框收起隐藏
- **AND** 调用 `saveSettings()`，`screen_lock_enabled` 设为 `false`

## MODIFIED Requirements

### Requirement: SettingsConfig 结构体
`SettingsConfig` SHALL include two new fields:
```go
ScreenLockEnabled  bool   `json:"screen_lock_enabled"`
ScreenLockPassword string `json:"screen_lock_password"`
```

### Requirement: GetAllSettings
`GetAllSettings()` SHALL read `screen_lock_enabled` and `screen_lock_password` from the database.

### Requirement: SaveAllSettings
`SaveAllSettings()` SHALL:
1. 如果 `ScreenLockPassword` 为空字符串，保留数据库中原来的密码值
2. 如果 `ScreenLockPassword` 非空，SHA-256 哈希后存储
3. 写入 `screen_lock_enabled` 和 `screen_lock_password` 到数据库

### Requirement: init() startup flow
在 `init()` 的 `loadSettings()` 完成后：
- 如果 `screen_lock_enabled === true`，显示锁屏遮罩
- 否则正常进入主界面

### Requirement: Default settings
默认设置中新增两个条目：
- `screen_lock_enabled` → `"false"`
- `screen_lock_password` → `""`
