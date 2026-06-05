# Tasks

## Task 1: 后端备份目录路径工具

- [x] 新增 `BackupDir()` 返回 `~/.jot/backup/` 路径（基于 `DefaultDBPath()` 的上层目录 + `backup` 子目录）
- [x] 新增 `EnsureBackupDir()` 确保目录存在（`os.MkdirAll`）

## Task 2: 后端备份方法 — BackupToDir

- [x] 使用 `s.db.Exec("VACUUM INTO ?", fullPath)` 创建压缩副本
- [x] 生成文件名 `jot-backup-YYYY-MM-DD_HHmmss.db`
- [x] 在 `app.go` 新增绑定方法 `BackupToDir() (string, error)`
- [x] 调用 `database.EnsureBackupDir()` + `noteService.ExportBackup(filePath)`
- [x] 成功返回 `"备份成功：文件名"`
- [x] 失败返回错误

## Task 3: 后端还原方法 — RestoreFromDir

- [x] 获取备份目录路径，列出所有 `*.db` 文件，按修改时间排序取最新
- [x] 无备份文件时返回 `ImportResult{Message: "暂无可用备份", SuccessCount: 0}`
- [x] 复用导入逻辑（备份 → 关连接 → 覆盖 → 重开 → 重建）
- [x] 成功返回 `ImportResult{Message: "已从备份文件恢复：文件名", SuccessCount: 1}`
- [x] 出错回滚（`.bak` 恢复）

## Task 4: 后端备份信息查询 — GetBackupInfo

- [x] 读取 `~/.jot/backup/` 目录下所有 `*.db` 文件，取最新者
- [x] 返回 `{"file_name": "...", "file_time": "YYYY-MM-DD HH:mm", "file_size": "X.XX MB"}`
- [x] 无备份时返回空值

## Task 5: 前端 UI — 备份还原分区

- [x] 在 `index.html` 数据管理页面的「数据操作」区新增「备份还原」分区
- [x] 两个按钮横向排列：「一键备份」「一键还原」
- [x] 备份信息标签显示最新备份信息

## Task 6: 前端交互 — 按钮事件绑定 + 信息刷新

- [x] 新增 DOM 引用：`backupBtn`、`restoreBtn`、`backupInfo`
- [x] 新增 `loadBackupInfo()` 函数
- [x] `backupBtn` 点击 → 调用 `App.BackupToDir()` → Toast 结果 → `loadBackupInfo()`
- [x] `restoreBtn` 点击 → 调用 `App.RestoreFromDir()` → Toast 结果 → `loadDataStats()` + `loadNotes()`
- [x] 在 `loadDataStats()` 末尾调用 `loadBackupInfo()`

## Task 7: 前端样式 — 备份还原区域样式

- [x] `.data-actions-row` 按钮横向 flex 布局，间距 12px
- [x] `.backup-info` 小字灰色标签，`font-size: 0.75rem`，`margin-top: 12px`

# Task Dependencies

- Task 1 是 Task 2/3/4 的前置
- Task 2、3、4 可并行
- Task 5 依赖 Task 2/3/4（需确定后端 API 签名）
- Task 6 依赖 Task 5（需前端 DOM 就位）
- Task 7 可单独并行
