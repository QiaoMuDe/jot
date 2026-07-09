# ZIP 格式备份/还原（含图片）Spec

## Why

笔记支持粘贴图片后，`~/.jot/images/` 目录中的图片成为笔记数据的重要组成部分。当前所有导出/备份操作只处理 `.db` 文件，导致还原后图片全部丢失为坏链。需将导出/备份改为 ZIP 格式（数据库 + 图片目录一并打包），同时重构导入/还原侧高度重复的核心逻辑。

## What Changes

- **核心抽象**：抽取 3 个内部函数 `exportSnapshot()` / `importFromArchive()` / `replaceDatabase()`，消除 `ImportDatabaseWithDialog` 和 `RestoreFromDir` 之间 ~40 行重复代码
- **导出/备份**：从纯 `.db` 输出改为 ZIP 打包（`数据库.db` + `images/` 目录）
- **导入/还原**：从 `.db` 直接替换改为 ZIP 解压 → 替换数据库 + 替换图片目录 → 重连
- **ResetDatabase**：末尾追加清空 `~/.jot/images/` 目录
- **不兼容旧 .db 备份**：导入/还原只接收 `.zip` 文件，旧 `.db` 不再支持
- **前端**：导出/导入对话框筛选器改为 `*.zip`，描述文案更新

## Impact

- Affected code: [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)（核心修改）、[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html)（按钮描述文案）、[data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js)（toast 文案适配）
- **BREAKING**：旧的 `.db` 备份文件不再兼容，需在更新说明中告知用户

## ADDED Requirements

### Requirement: 核心抽象函数

系统 SHALL 提供以下内部辅助函数：

#### Scenario: exportSnapshot — 统一导出
- **WHEN** 调用 `exportSnapshot(destZipPath string) error`
- **THEN** 执行 VACUUM INTO 到临时 `.db` → 将临时 `.db` 和 `images/` 目录打包到 `destZipPath`（ZIP 内结构：`jot-backup.db` + `images/uuid_name.ext`）→ 清理临时文件
- **THEN** `destZipPath` 后缀为 `.zip`

#### Scenario: importFromArchive — 统一解压
- **WHEN** 调用 `importFromArchive(srcZipPath string) error`
- **THEN** 解压 `.zip` 到临时目录 → 提取 `jot-backup.db` 和 `images/` → 调用 `replaceDatabase()` → 清理临时目录

#### Scenario: replaceDatabase — 统一替换
- **WHEN** 调用 `replaceDatabase(srcDBPath, srcImagesDir string) error`
- **THEN** 备份当前 `dbPath` → 关闭旧连接 → `srcDBPath` 复制到 `dbPath` → `srcImagesDir` 内容替换 `~/.jot/images/` → 重新初始化数据库 → 重建服务 → 清理备份
- **THEN** 任何步骤失败时自动回滚备份

### Requirement: ResetDatabase 清空图片目录

- **WHEN** 用户执行恢复出厂设置
- **THEN** 删除 `~/.jot/images/` 目录下所有文件

## MODIFIED Requirements

### Requirement: ExportDataWithDialog

**修改为：**
- **WHEN** 用户点击「导出数据」
- **THEN** 弹出保存对话框，筛选器为 `ZIP 文件 (*.zip)`，默认文件名 `jot-backup-YYYY-MM-DD.zip`
- **THEN** 调用 `exportSnapshot()` 打包到用户选择的路径
- **THEN** 返回「导出成功：xxx.zip」

### Requirement: ImportDatabaseWithDialog

**修改为：**
- **WHEN** 用户点击「导入数据」
- **THEN** 弹出打开对话框，筛选器为 `ZIP 文件 (*.zip)`
- **THEN** 调用 `importFromArchive()` 完成恢复
- **THEN** 返回恢复结果

### Requirement: BackupToDir

**修改为：**
- **WHEN** 用户点击「一键备份」
- **THEN** 调用 `exportSnapshot(backup/jot-backup.zip)` 输出到备份目录
- **THEN** 返回「备份成功：jot-backup.zip」

### Requirement: RestoreFromDir

**修改为：**
- **WHEN** 用户点击「一键还原」
- **THEN** 检查 `backup/jot-backup.zip` 是否存在
- **THEN** 调用 `importFromArchive(backup/jot-backup.zip)` 完成恢复
- **THEN** 返回恢复结果

### Requirement: GetBackupInfo

**修改为：**
- **WHEN** 读取备份信息
- **THEN** 检查 `backup/jot-backup.zip`（而非 `.db`）的文件信息
- **THEN** 显示文件名/修改时间/文件大小

## REMOVED Requirements

### Requirement: 旧 .db 格式兼容

**Reason**: 保持代码简洁，避免 `if .zip / else if .db` 分支。用户后续可通过「导出数据」重新生成 ZIP 备份。

**Migration**: 更新日志中说明旧 `.db` 备份不再支持，请用户提前通过旧版本导出为 `.db` 再导入新版本的方式无法被支持，需在新版本中重新备份。
