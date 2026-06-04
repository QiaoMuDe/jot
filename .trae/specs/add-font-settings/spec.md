# 字体设置功能 Spec

## Why

用户希望在设置页面中可以自定义字体族和字体大小，提升阅读体验和个性化程度。同时需要持久化存储这些设置。

## What Changes

### 后端

1. **新增 Setting 数据模型**：在 `models/` 下新增 `setting.go`，定义 KV 结构的配置表
2. **新增 SettingService**：在 `services/` 下新增 `setting_service.go`，封装配置的读写操作
3. **App 层绑定**：在 `app.go` 中新增 `GetSetting`/`SetSetting`/`GetSystemFonts` 三个绑定方法
4. **数据库迁移**：在 `database/db.go` 的 `AutoMigrate` 中追加 `&models.Setting{}`
5. **注册 SettingService**：在 `NewApp()` 中初始化 SettingService

### 前端

6. **设置页面新增字体设置区**：在标签管理上方新增"字体设置"分区
7. **字体族选择器**：获取系统字体列表，支持下键盘上下键和回车选择的自定义下拉菜单
8. **字体大小设置**：预设按钮组 + 自定义输入框
9. **字体设置即时生效**：修改后通过 CSS 变量立即应用到页面所有组件
10. **设置持久化**：字体偏好保存到后端数据库中，重新打开应用后自动恢复

## Impact

- Affected specs: 设置页面功能
- Affected code:
  - 新增: `models/setting.go`, `services/setting_service.go`
  - 修改: `database/db.go`, `app.go`, `frontend/index.html`, `frontend/src/style.css`, `frontend/src/main.js`, `frontend/src/app.css`, `frontend/wailsjs/go/main/App.js`
  - 自动生成: `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/models.ts`

## ADDED Requirements

### Requirement: 设置数据持久化

系统 SHALL 提供 KV 结构的配置存储表。

#### Scenario: 设置保存
- **WHEN** 用户修改字体设置并确认
- **THEN** 设置值通过 API 保存到数据库
- **THEN** 前端立即应用新设置

#### Scenario: 设置恢复
- **WHEN** 应用启动
- **THEN** 从数据库读取已保存的字体设置并应用

### Requirement: 字体族选择

系统 SHALL 提供系统字体列表供用户选择。

#### Scenario: 获取系统字体
- **WHEN** 用户打开字体族下拉菜单
- **THEN** 后端返回系统可用字体列表
- **THEN** 下拉菜单高亮当前选中的字体

#### Scenario: 键盘导航
- **WHEN** 下拉菜单打开时用户按上下键
- **THEN** 高亮选项跟随移动
- **WHEN** 用户按回车
- **THEN** 选中当前高亮的字体并关闭菜单
- **WHEN** 用户按 Escape 或点击菜单外区域
- **THEN** 关闭菜单但不改变选择

### Requirement: 字体大小设置

系统 SHALL 提供基础字体大小设置，影响所有组件。

#### Scenario: 预设选择
- **WHEN** 用户点击预设大小按钮
- **THEN** 字体大小立即应用并保存

#### Scenario: 自定义输入
- **WHEN** 用户在自定义输入框中输入数值
- **THEN** 字体大小立即应用并保存

## MODIFIED Requirements

无

## REMOVED Requirements

无
