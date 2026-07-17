# Checklist

## 后端
- [ ] SettingsConfig 结构体新增 `ScreenLockEnabled bool` + `ScreenLockPassword string` 字段
- [ ] `GetAllSettings()` 正确读取两个新字段
- [ ] `SaveAllSettings()` 正确存储：密码为空时保留旧值，非空时 SHA-256 哈希
- [ ] `InitDefaultSettings()` 包含默认值 `screen_lock_enabled=false` + `screen_lock_password=""`
- [ ] `VerifyScreenLockPassword()` 绑定方法正确实现哈希比对
- [ ] 无密码时（空字符串），`VerifyScreenLockPassword()` 返回 `true`（不锁）

## 前端 HTML
- [ ] 锁屏遮罩层 `#lockScreen` 包含：应用图标/名称、密码输入框、解锁按钮
- [ ] 锁屏遮罩层位于 body 最末尾，`z-index: 9999`
- [ ] 锁屏遮罩层包含密码显示/隐藏切换按钮
- [ ] 设置页中"锁屏密码"卡片位于"回收站清理"卡片之前
- [ ] 设置卡片包含 toggle 开关 + 可展开的密码输入区域

## 前端 CSS
- [ ] 锁屏 `backdrop-filter: blur(8px)` 毛玻璃效果
- [ ] 解锁退出动画：blur 淡出 + opacity + translateY（`.lock-screen-exit`）
- [ ] 输错密码 shake 抖动动画
- [ ] 输入框 focus 发光效果
- [ ] 密码输入框展开/收起动画（max-height 过渡）
- [ ] 锁屏适配当前主题（跟随 data-theme）

## 前端 JS
- [ ] `loadSettings()` 读取 `screen_lock_enabled`
- [ ] `saveSettings()` 收集锁屏设置并保存
- [ ] `init()` 中 `loadSettings()` 后检查并显示锁屏
- [ ] 锁屏输入框自动聚焦
- [ ] 回车触发解锁，正确调用 `VerifyScreenLockPassword()`
- [ ] 解锁成功：动画过渡后移除锁屏
- [ ] 解锁失败：shake 动画 + 清空输入框 + 重新聚焦

## 交互验证
- [ ] 默认启动不显示锁屏（功能默认关闭）
- [ ] 设置页开启开关后，重启应用显示锁屏
- [ ] 输入正确密码后进入应用
- [ ] 输入错误密码看到抖动动画
- [ ] 设置页可修改密码
- [ ] 设置页关闭功能后，重启不再显示锁屏
