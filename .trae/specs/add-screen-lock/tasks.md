# Tasks

- [ ] Task 1: Go 后端 — SettingsConfig 新增字段 + 数据库默认值
  - 在 `types.go` 的 `SettingsConfig` 结构体中添加 `ScreenLockEnabled bool` 和 `ScreenLockPassword string`
  - 在 `GetAllSettings()` 中读取这两个字段
  - 在 `SaveAllSettings()` 中处理：密码为空时保留旧值，非空时 SHA-256 哈希后存储
  - 在 `db.go` 的 `InitDefaultSettings()` 中添加 `screen_lock_enabled`（false）和 `screen_lock_password`（""）默认值
  - 在 `app.go` 中新增 `VerifyScreenLockPassword(password string) bool` 绑定方法（读取 DB 哈希值比对）

- [ ] Task 2: 前端 HTML — 锁屏遮罩层 + 设置卡片
  - 在 `index.html` 中新增锁屏遮罩层 `#lockScreen`（位于 body 最末尾，作为所有视图之上的覆盖层）
    - 包含：应用图标/名称、密码输入框、解锁按钮、显示/隐藏密码按钮
    - `z-index: 9999`，全屏固定定位
  - 在设置页面（`#viewSettings` 内）新增"锁屏密码"设置卡片
    - 位置：在"回收站清理"卡片之前
    - 结构：`ai-group-header`（锁图标）+ toggle 开关 + 可展开的密码输入区域

- [ ] Task 3: 前端 CSS — 锁屏"雾散之门"动画 + 设置卡片样式
  - 锁屏遮罩层样式：
    - `backdrop-filter: blur(8px)` 毛玻璃效果
    - 背景使用当前主题色（半透明）或深色半透明遮罩
    - 居中布局，输入框 focus 发光效果
    - **关键动画**: `.lock-screen-exit` 类触发 blur 8px→0px + opacity 1→0 + translateY(0→-20px)，持续 400ms ease-out
    - 输入框 shake 动画（错密码时触发）
  - 设置卡片：
    - 密码输入框展开/收起动画（max-height 变换，200ms ease-out）
    - 与现有设置卡片视觉风格一致

- [ ] Task 4: 前端 JS — 锁屏逻辑 + 设置联动
  - `loadSettings()` 中：读取 `screen_lock_enabled` 和 `screen_lock_password`（仅用于判断是否有密码，不显示明文）
  - `saveSettings()` 中：收集锁屏设置的值
  - `init()` 中：`loadSettings()` 完成后检查 `screen_lock_enabled`，若开启则显示锁屏
  - 锁屏交互：
    - 自动聚焦到密码输入框
    - 回车键触发 `VerifyScreenLockPassword()`
    - 解锁成功：给锁屏添加 `.lock-screen-exit` 类，动画结束后移除 DOM
    - 解锁失败：触发 shake 动画，清空输入框并重新聚焦
  - 设置卡片交互：
    - toggle 开关切换时展开/收起密码输入框
    - 密码输入框支持回车保存

# Task Dependencies
- [Task 1] 依赖无（后端独立）
- [Task 2] 依赖无（HTML 独立）
- [Task 3] 依赖 [Task 2]（需要对应的 HTML 结构）
- [Task 4] 依赖 [Task 1][Task 2]（需要后端 API 和前端 DOM）
