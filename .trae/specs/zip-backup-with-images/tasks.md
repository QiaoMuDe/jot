# Tasks

- [x] Task 1: 重构核心抽象函数 — 抽取 `exportSnapshot()`、`importFromArchive()`、`replaceDatabase()` 三个内部函数，消除导入/还原侧重复代码
  - [x] SubTask 1.1: 实现 `exportSnapshot(destZipPath)`：VACUUM INTO 临时 .db → `archive/zip` 打包 {`jot-backup.db`, `images/`} → 清理临时文件
  - [x] SubTask 1.2: 实现 `replaceDatabase(srcDBPath, srcImagesDir)`：备份当前 db → 关闭旧连接 → 复制 db + 替换 images/ → 重新初始化 → 重建服务 → 清理备份（含回滚机制）
  - [x] SubTask 1.3: 实现 `importFromArchive(srcZipPath)`：解压 ZIP 到临时目录 → 调用 `replaceDatabase()` → 清理临时目录
  - [x] SubTask 1.4: 验证 `go build ./...` 编译通过

- [x] Task 2: 重构 ExportDataWithDialog — 改为调用 `exportSnapshot()` 输出 `.zip`
  - [x] SubTask 2.1: 前端筛选器改为 `*.zip`，默认文件名改为 `jot-backup-YYYY-MM-DD.zip`
  - [x] SubTask 2.2: 调用 `exportSnapshot()` 代替当前 VACUUM INTO + CopyEx
  - [x] SubTask 2.3: 验证 `npx vite build` 编译通过

- [x] Task 3: 重构 ImportDatabaseWithDialog — 改为调用 `importFromArchive()` 从 `.zip` 恢复
  - [x] SubTask 3.1: 前端筛选器改为 `*.zip`
  - [x] SubTask 3.2: 调用 `importFromArchive()` 代替当前复制 .db + 重连逻辑

- [x] Task 4: 重构 BackupToDir — 改为调用 `exportSnapshot(backup/jot-backup.zip)`
  - [x] SubTask 4.1: 目标文件从 `jot-backup.db` 改为 `jot-backup.zip`
  - [x] SubTask 4.2: 调用 `exportSnapshot()` 代替当前 VACUUM INTO

- [x] Task 5: 重构 RestoreFromDir — 改为调用 `importFromArchive(backup/jot-backup.zip)`
  - [x] SubTask 5.1: 检查 `jot-backup.zip` 存在性代替 `.db`
  - [x] SubTask 5.2: 调用 `importFromArchive()` 代替当前复制 .db + 重连逻辑

- [x] Task 6: 重构 GetBackupInfo — 改为读取 `jot-backup.zip` 的文件信息
  - [x] SubTask 6.1: 文件名返回 `jot-backup.zip`，Stat 路径改为 `.zip`

- [x] Task 7: ResetDatabase 清空图片目录
  - [x] SubTask 7.1: `ResetDatabase()` 末尾调用 `os.RemoveAll(imageDir)` 后重新 `os.MkdirAll(imageDir)`

- [x] Task 8: 前端文案适配
  - [x] SubTask 8.1: [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html) 导出描述从「备份为 .db 文件」改为「备份为 .zip 文件」，导入描述从「从备份文件还原」改为「从备份文件还原（.zip）」
  - [x] SubTask 8.2: [data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js) 中任何硬编码的 `.db` toast 文案改为 `.zip`

- [x] Task 9: 编译与运行验证
  - [x] SubTask 9.1: `go build ./...` 编译通过
  - [x] SubTask 9.2: `npx vite build` 前端构建通过

# Task Dependencies

- Task 1 是所有其他 Task 的前置依赖（抽象函数先完成）
- Task 2~6 可并行（各自重构一个业务方法，互不依赖）
- Task 7 可并行（独立功能）
- Task 8 可并行（纯前端文案）
- Task 9 是最终验证
