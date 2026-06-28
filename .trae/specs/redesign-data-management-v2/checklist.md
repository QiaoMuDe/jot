# Checklist

- [x] 统计卡片 5 列网格保留，padding 缩小，hover 效果简化
- [x] 操作按钮从全宽卡片改为设置列表行样式（icon + label/desc + chevron）
- [x] 操作列表分为三个功能组（数据迁移/数据库维护/快速备份+数据目录）
- [x] 危险操作（恢复出厂设置）独立为底部单独区域
- [x] 所有按钮 id 保持不变（exportDataBtn, importDataBtn, vacuumDbBtn, resetAllBtn, openDataDirBtn, backupBtn, restoreBtn, backupInfo）
- [x] 旧 CSS 类 `.data-action-btn`、`.dab-icon`、`.dab-text`、`.dab-desc` 已清理
- [x] 分隔线样式正确（左侧缩进 48px）
- [x] 窄屏响应式适配正常
- [x] 所有按钮点击功能正常
- [x] 构建通过无错误