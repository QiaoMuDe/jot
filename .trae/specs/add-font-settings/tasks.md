# Tasks

## Task 1: 新增 Setting 数据模型

- [x] 创建 `models/setting.go`，定义 Setting 结构体（ID, Key, Value, CreatedAt, UpdatedAt）
- [x] Key 字段设唯一索引

## Task 2: 新增 SettingService

- [x] 创建 `services/setting_service.go`，实现 Get(key)/Set(key, value)/GetAll() 方法
- [x] 使用 GORM 操作 settings 表，Get 查不到返回空字符串不报错

## Task 3: App 层绑定 SettingService

- [x] 在 `app.go` 的 App 结构体中添加 `settingService *services.SettingService`
- [x] 在 `NewApp()` 中初始化 SettingService
- [x] 新增 `GetSetting(key string) (string, error)` 绑定方法
- [x] 新增 `SetSetting(key, value string) error` 绑定方法
- [x] 新增 `GetSystemFonts() ([]string, error)` 绑定方法（返回常用系统字体列表 + Win/Mac 系统字体检测）

## Task 4: 数据库迁移

- [x] 在 `database/db.go` 的 `AutoMigrate` 中追加 `&models.Setting{}`

## Task 5: 前端 App.js 绑定

- [x] 在 `frontend/wailsjs/go/main/App.js` 中添加 `GetSetting`/`SetSetting`/`GetSystemFonts` 导出函数

## Task 6: 设置页面 HTML 结构

- [x] 在 `frontend/index.html` 的设置页面中添加字体设置区域 HTML：
  - 字体族选择器（自定义下拉菜单，显示当前选中字体预览）
  - 字体大小设置（预设按钮组 xs/sm/base/lg/xl + 自定义输入框）

## Task 7: 前端 JS 逻辑 — 字体族

- [x] 实现 `loadFontSettings()`：从后端读取已保存的字体设置并应用
- [x] 实现字体族下拉菜单：打开时获取系统字体列表，渲染选项列表，支持键盘上下键/回车/Escape 导航，高亮当前选中项，点击外部关闭
- [x] 选中字体后调用 `SetSetting('font_family', fontName)` 保存并应用

## Task 8: 前端 JS 逻辑 — 字体大小

- [x] 实现预设大小按钮：点击后设置对应基础字号，调用 `SetSetting('font_size', value)` 保存
- [x] 实现自定义输入框：输入完按回车或失焦时应用并保存
- [x] 大小设置通过设置 `html` 元素的 `font-size` 生效

## Task 9: CSS 适配 — px 转 rem

- [x] 将 `app.css` 中的 `html` font-size 设置为动态值（由 JS 控制）
- [x] 将 `style.css` 中所有硬编码的 `font-size: Xpx` 转换为 `font-size: Xrem`（除以 16），使所有字体随基础字号缩放

## Task 10: 启动加载字体设置

- [x] 在应用初始化（`init` 或第一次切换到设置页面）时调用 `GetSetting` 加载已保存的字体设置
- [x] 应用后刷新设置页面的 UI 状态（下拉菜单显示当前字体、大小按钮高亮）

# Task Dependencies

- [Task 1] → [Task 2] → [Task 3] → [Task 4] → [Task 5]
- [Task 5] 和 [Task 6] 可并行
- [Task 7] 和 [Task 8] 依赖 [Task 5] 和 [Task 6]
- [Task 9] 可独立进行
- [Task 10] 依赖 [Task 7] 和 [Task 8]
