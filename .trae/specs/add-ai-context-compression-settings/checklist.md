# Checklist

- [x] 后端 `SettingsConfig` 新增 `ai_context_token_budget` 与 `ai_context_summary_trigger_ratio` 两字段，并在 `GetAllSettings`/`SaveAllSettings` 中正确读取、钳制与落库
- [x] 后端 `ai_context.go` 范围常量已校准为预算 `[32768, 524288]`、触发比例 `[0.1, 0.9]`，默认 128K / 0.8 不变
- [x] 设置页「对话与搜索」面板新增「摘要压缩预算」（32~512K，默认 128）与「压缩触发比例」（0.1~0.9，默认 0.8）两项，样式与交互同现有设置项
- [x] 前端 `loadSettings` 用数据库中最新值渲染两项；`saveSettings` 将两项（预算 ×1024）写入统一设置保存流程
- [x] 前端两项 `change` 有越界钳制/重置与保存成功通知
- [x] 运行时压缩判断（`tail >= budget * ratio`）与前端压缩进度圆环每次从数据库读取最新值，修改保存后立即生效
- [x] 变更未生成任何图片/视频资源，仅涉及代码与 HTML