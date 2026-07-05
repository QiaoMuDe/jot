# 默认值统一入库 Spec

## Why

当前应用的默认值分散在前端 JS 和后端 Go 代码的多个位置（硬编码常量、Getter 函数的 fallback 逻辑），每次修改默认值需要同时改动前端和后端，双向维护极易出错。

## What Changes

- 在 `database.InitDB()` 中新增 `initDefaultSettings(db)` 函数，一次性地将所有默认值插入 `settings` 表
- 后端所有 Getter 方法去掉硬编码的 fallback 逻辑，依赖 DB 中已有值
- 前端所有 load 函数去掉硬编码的 fallback 默认值，统一从后端 `GetSetting` 获取
- `ResetDatabase()` 也调用 `initDefaultSettings(db)` 恢复出厂默认值

## Impact

- Affected specs: 无（纯架构重构，不新增功能）
- Affected code:
  - `internal/database/db.go` — 新增 `initDefaultSettings()`
  - `app.go` — 移除各 Getter 中的硬编码 fallback
  - `internal/services/ai_service.go` — 移除硬编码 fallback
  - `internal/services/setting_service.go` — 可能新增 `EnsureSetting()` 等辅助方法
  - `frontend/src/main.js` — 简化各 load* 函数，移除硬编码默认值

## Requirements

### Requirement: 在 DB 初始化时写入默认值

The system SHALL provide all default settings as rows in the `settings` table during `InitDB()`.

#### Scenario: 初始化时写入
- **WHEN** `InitDB()` 执行 AutoMigrate 之后
- **THEN** 遍历所有默认值，对数据库中不存在的 key 执行 INSERT

#### Scenario: 重置数据库
- **WHEN** `ResetDatabase()` 执行
- **THEN** 同样调用 `initDefaultSettings()` 恢复默认值

### Requirement: 后端 Getter 简化

The system SHALL remove hardcoded default fallbacks from all backend Getter methods.

#### Scenario: 读取已存在默认值
- **WHEN** 后端 `GetSetting(key)` 读取任何已有默认值的 key
- **THEN** 返回该默认值（不再返回空字符串）

#### Scenario: 读取无默认值的 key
- **WHEN** `GetSetting(key)` 读取无默认值的 key
- **THEN** 仍返回空字符串（保持向后兼容）

### Requirement: 前端简化

The system SHALL remove hardcoded default fallbacks in frontend load functions.

#### Scenario: 前端加载设置
- **WHEN** 前端加载各设置
- **THEN** 直接使用 `GetSetting(key)` 的返回值，不再判断空值后使用硬编码默认值

## 默认值清单

| Key | 默认值 | 说明 |
|-----|--------|------|
| theme | "default" | 主题 |
| font_family | "" | 字体族（空=系统默认） |
| font_size | "16" | 字号 |
| code_highlight_theme | "monokai-dimmed" | 代码高亮主题 |
| note_open_fullscreen | "false" | 笔记全屏打开 |
| sort_order | "updated_at" | 排序方式 |
| page_size | "20" | 分页大小 |
| quick_note_enabled | "false" | 快速笔记 |
| cm_syntax_highlight | "true" | 语法高亮 |
| ai_provider | "openai" | AI 服务商 |
| ai_base_url | "" | AI API 地址 |
| ai_api_key | "" | API 密钥 |
| ai_model | "" | AI 模型 |
| ai_thinking_enabled | "false" | 深度思考 |
| tavily_api_key | "" | Tavily 密钥 |
| ai_web_search_enabled | "false" | 联网搜索 |
| ai_card_recall_enabled | "false" | 卡片召回 |
| ai_card_recall_limit | "5" | 卡片召回条数 |
| ai_ref_max_chars | "5000" | 引用截断字数 |
| ai_search_result_limit | "5" | 搜索结果数 |

## 依赖关系

| 步骤 | 依赖 | 说明 |
|------|------|------|
| 1. DB 初始化逻辑 | 无 | 先写 `initDefaultSettings` |
| 2. 后端 Getter 简化 | 步骤 1 | 依赖 DB 中有值 |
| 3. 前端简化 | 步骤 1 | 依赖后端 GetSetting 稳定返回值 |
