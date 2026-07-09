# 孤儿图片清理功能 Spec

## Why

用户粘贴图片到笔记后，图片文件保存在 `~/.jot/images/` 目录。但笔记删除或移除图片引用后，文件仍残留在目录中，长期积累会占用磁盘空间。需要一个清理功能，自动识别并删除未被任何笔记引用的孤儿图片。

## What Changes

- **Go 后端**：新增 `CleanupOrphanImages()` 方法，遍历 `~/.jot/images/` 目录，与所有笔记（含回收站）的 content 字段做引用对比，删除未引用的图片文件，返回删除数量
- **前端数据管理页**：在"数据清理"分组中（清空AI会话、清空已完成待办之后）新增"清理未引用图片"按钮
- **存储优化联动**：点击"存储优化"按钮时额外附带执行孤儿图片清理逻辑

## Impact

- Affected specs: add-data-management, add-markdown-image-support
- Affected code:
  - `app.go` — 新增 `CleanupOrphanImages()` 绑定方法
  - `frontend/index.html` — 新增"清理未引用图片"按钮 DOM
  - `frontend/src/main.js` — 绑定按钮事件 + 导出函数
  - `frontend/src/js/data-management.js` — 新增 `cleanupOrphanImages()` 函数

## ADDED Requirements

### Requirement: Go 后端孤儿图片清理 API

系统 SHALL 通过 Wails Bind 暴露 `CleanupOrphanImages() int` 方法。

#### Scenario: 清理孤儿图片成功
- **GIVEN** `~/.jot/images/` 目录中有 3 张图片文件，其中 1 张被笔记引用，2 张未被引用
- **WHEN** 调用 `CleanupOrphanImages()`
- **THEN** 扫描所有笔记（含回收站）的 content 字段，匹配 `![](/images/` 引用
- **THEN** 删除未被任何笔记引用的 2 张图片
- **THEN** 返回删除数量 2

#### Scenario: 无孤儿图片
- **GIVEN** `~/.jot/images/` 目录中所有图片均被笔记引用
- **WHEN** 调用 `CleanupOrphanImages()`
- **THEN** 不删除任何文件
- **THEN** 返回删除数量 0

#### Scenario: 目录不存在
- **GIVEN** `~/.jot/images/` 目录不存在
- **WHEN** 调用 `CleanupOrphanImages()`
- **THEN** 返回删除数量 0，不报错

#### Scenario: 目录为空
- **GIVEN** `~/.jot/images/` 目录存在但为空
- **WHEN** 调用 `CleanupOrphanImages()`
- **THEN** 返回删除数量 0

### Requirement: 引用匹配逻辑

系统 SHALL 通过扫描所有笔记（含软删除/回收站笔记）的 content 字段来匹配图片引用。

- 匹配方式：对于每个图片文件名 `uuid_name.ext`，检查 `content` 是否包含字符串 `/images/uuid_name.ext`
- 必须包含回收站笔记的引用（防止用户恢复笔记后图片丢失）
- 只匹配 `~/.jot/images/` 目录下的文件名，不匹配外部 URL

### Requirement: 前端"清理未引用图片"按钮

系统 SHALL 在数据管理页的"数据清理"分组中增加一个"清理未引用图片"按钮。

#### Scenario: 点击清理按钮
- **GIVEN** 数据管理页面的"数据清理"分组
- **WHEN** 用户点击"清理未引用图片"按钮
- **THEN** 弹出确认对话框"确定要清理未引用的图片吗？这将删除笔记中不再使用的图片文件。"
- **THEN** 用户确认后调用 `CleanupOrphanImages()`
- **THEN** 显示清理结果 Toast（如"已清理 2 张未引用图片"）

### Requirement: 存储优化联动

系统 SHALL 在用户点击"存储优化"按钮时，在完成 VACUUM 后自动附带执行孤儿图片清理。

#### Scenario: 存储优化时自动清理
- **GIVEN** 数据管理页面的"存储优化"按钮
- **WHEN** 用户点击"存储优化"
- **THEN** 先执行数据库 VACUUM
- **THEN** 自动执行孤儿图片清理
- **THEN** 在成功提示中包含清理结果（如"数据库已瘦身，已清理 2 张未引用图片"）

## MODIFIED Requirements

### Requirement: 存储优化按钮行为（修改）

原"存储优化"按钮仅执行数据库 VACUUM。修改后，执行 VACUUM + 孤儿图片清理，并合并显示结果。

## REMOVED Requirements

（无删除项）
