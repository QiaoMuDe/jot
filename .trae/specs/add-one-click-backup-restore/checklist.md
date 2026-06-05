# Checklist

## 后端
- [x] `BackupDir() / EnsureBackupDir()` 工具函数实现
- [x] `BackupToDir()` 绑定方法 — VACUUM INTO 导出到 `~/.jot/backup/` 固定目录
- [x] `RestoreFromDir()` 绑定方法 — 从 `~/.jot/backup/` 最新备份还原
- [x] `GetBackupInfo()` 绑定方法 — 返回最新备份文件名/时间/大小
- [x] 无备份时 `RestoreFromDir` 返回 "暂无可用备份"
- [x] 还原失败时从 `.bak` 自动回滚
- [x] `go build ./...` 通过 ✅

## 前端
- [x] 数据管理页面新增「备份还原」分区 UI（一键备份 + 一键还原按钮）
- [x] 备份信息标签显示最新备份信息
- [x] 无备份时显示"暂无备份"
- [x] 点击一键备份 → Toast 结果 + 标签刷新
- [x] 点击一键还原 → Toast 结果 + 刷新统计数据与笔记列表
- [x] 备份还原按钮样式与现有按钮风格统一（复用 `.data-action-btn`）
- [x] `npx vite build` 通过 ✅
