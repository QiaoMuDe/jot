# AI 历史会话摘要压缩设置项 Spec

## Why
AI 助手模块的历史会话摘要压缩功能的两个关键参数——上下文 token 预算（`ai_context_token_budget`）与压缩触发比例（`ai_context_summary_trigger_ratio`）——目前只在数据库初始化时写入默认值，前端设置页没有可视化修改入口，用户想调整必须改库，不方便。

## What Changes
- 在设置页「对话与搜索」面板中新增两个设置项：**摘要压缩预算**（K 单位，范围 32~512，默认 128）与**压缩触发比例**（范围 0.1~0.9，默认 0.8）。
- 后端 `SettingsConfig` 增加两个字段，纳入统一的 `GetAllSettings`/`SaveAllSettings` 流程，修改保存即写入数据库。
- 校准后端 `ai_context.go` 中的取值上下限常量，使运行时钳制范围与设置项一致（token 预算 `[32768, 524288]`，触发比例 `[0.1, 0.9]`）。
- 运行时压缩逻辑与前端进度圆环均**每次从数据库读取最新值**（现有 `GetContextTokenBudget`/`GetContextSummaryTriggerRatio` 已按 `SettingService.Get` 实时读取，无需改动读取方式，仅保证设置持久化即生效）。
- 无数据库表结构变更：两个 key 已在 `InitDefaultSettings` 中初始化。

## Impact
- Affected specs: 对话设置（dialog-search 面板）、AI 摘要压缩、设置统一保存/加载
- Affected code:
  - `internal/services/types.go`（`SettingsConfig`、`GetAllSettings`、`SaveAllSettings`）
  - `internal/services/ai_context.go`（范围常量）
  - `frontend/index.html`（对话设置面板新增两项）
  - `frontend/src/main.js`（`loadSettings`/`saveSettings`/`initAISettings`）

## ADDED Requirements
### Requirement: 摘录压缩预算与触发比例设置项
系统 SHALL 在设置页「对话与搜索」面板中提供可编辑的「摘要压缩预算」与「压缩触发比例」两项数值设置。

#### 结果：展示与保存
- **WHEN** 用户打开设置页「对话与搜索」面板
- **THEN** 显示两个输入项：摘要压缩预算（K tokens，min 32 / max 512，默认 128）与压缩触发比例（min 0.1 / max 0.9，步进 0.05，默认 0.8），并展示当前数据库中的最新值
- **AND** 两项的交互、样式、hover/动效与同面板现有设置项（如卡片召回数）保持一致，且不生成任何图片/视频资源

#### 结果：合法性钳制与保存
- **WHEN** 用户修改任一输入值并触发 `change`
- **THEN** 越界值被钳制到合法范围（预算 <32 → 重置 128；预算 >512 → 512；比例 <0.1 → 0.8；比例 >0.9 → 0.9），随后调用统一保存流程写入数据库 `ai_context_token_budget`（预算 K 值 × 1024 存为实际 token 数）与 `ai_context_summary_trigger_ratio`
- **AND** 显示保存成功通知

#### 结果：运行与展示实时生效
- **WHEN** 保存成功且后续发起 AI 对话或刷新设置页
- **THEN** 摘要压缩触发判断（`tail >= budget * ratio`）以前端进度指示器展示均使用数据库中的最新值（运行时与展示均每次实时读取）

## MODIFIED Requirements
### Requirement: AI 上下文 token 预算与触发比例取值范围
后端 `ai_context.go` 的取值钳制常量 SHALL 与设置项范围一致（原 `<4096,1M>` 调整为 `<32768, 524288>`；原 `<0.05,1.0>` 调整为 `<0.1, 0.9>`），默认值保持 128K / 0.8 不变，运行时读取逻辑（实时读库）不变。

### Requirement: 设置统一保存/加载
`SettingsConfig` SHALL 新增 `ai_context_token_budget`（int，存实际 token 数）与 `ai_context_summary_trigger_ratio`（float64）两个字段，纳入 `GetAllSettings` 返回与 `SaveAllSettings` 落库（预算值在进出前端时按 K=value、库=value×1024 换算）。

## REMOVED Requirements
无。