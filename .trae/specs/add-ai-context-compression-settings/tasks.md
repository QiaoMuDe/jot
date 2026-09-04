# Tasks

- [x] Task 1: 后端 `SettingsConfig` 增加两个字段并纳入统一读/存流程
  - [x] 1.1 在 `internal/services/types.go` 的 `SettingsConfig` 增加 `AIContextTokenBudget int \`json:"ai_context_token_budget"\`` 与 `AIContextSummaryTriggerRatio float64 \`json:"ai_context_summary_trigger_ratio"\``
  - [x] 1.2 `GetAllSettings` 中从 setting 表读取两键并填充（预算读原始 token 数，触发比例读原始值）
  - [x] 1.3 `SaveAllSettings` 中钳制预算到 `[32768, 524288]`（非法重置为 131072）、触发比例到 `[0.1, 0.9]`（非法重置为 0.8），并写入 `ai_context_token_budget` / `ai_context_summary_trigger_ratio` 两键
- [x] Task 2: 校准后端 `ai_context.go` 范围常量
  - [x] 2.1 `MinContextTokenBudget` 4096 → 32768，`MaxContextTokenBudget` 1048576 → 524288
  - [x] 2.2 `MinSummaryTriggerRatio` 0.05 → 0.1，`MaxSummaryTriggerRatio` 1.0 → 0.9（默认值 131072 / 0.8 保持不变）
- [x] Task 3: 前端设置页「对话与搜索」面板新增两个设置项（`index.html`）
  - [x] 3.1 在 `data-panel="dialog-search"` 面板末尾新增「摘要压缩预算」`<input type="number" id="aiSummaryTokenBudget" min="32" max="512" value="128">`（K 单位）与说明文字
  - [x] 3.2 新增「压缩触发比例」`<input type="number" id="aiSummaryTriggerRatio" min="0.1" max="0.9" step="0.05" value="0.8">` 与说明文字
- [x] Task 4: 前端 `main.js` 接入加载、保存与交互
  - [x] 4.1 `loadSettings` 中填充两项（预算显示 `cfg.ai_context_token_budget / 1024`，比例显示 `cfg.ai_context_summary_trigger_ratio`）
  - [x] 4.2 `saveSettings` 的 cfg 中加入 `ai_context_token_budget: parseInt(预算输入)*1024` 与 `ai_context_summary_trigger_ratio: parseFloat(比例输入)`，并以空值兜底默认
  - [x] 4.3 `initAISettings` 中为两项注册 `change` 处理器：越界钳制/重置 + `saveSettings()` + 保存成功通知（复用现有数值项交互范式）

# Task Dependencies
- [Task 1] 无
- [Task 2] 无（可与 Task 1 并行）
- [Task 3] 无（可与 Task 1、Task 2 并行）
- [Task 4] 依赖 [Task 1]、[Task 3]（字段与 DOM id 确定后接入）