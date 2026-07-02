# 移除空回复标记字段（is_empty_response）

## Why

之前为了将 AI 空回复作为占位消息保存到数据库，在消息结构中新增了 `is_empty_response` 字段。但实际正确的做法是：AI 空回复不应入库，仅通过右上角通知告知用户即可。该字段已无存在必要，需要完整移除。

## What Changes

- **移除** `internal/models/ai_message.go` 中 `AIMessage` 结构体的 `IsEmptyResponse` 字段
- **移除** `internal/services/ai_service.go` 中 `Message` DTO 的 `IsEmptyResponse` 字段及相关引用
- **移除** `frontend/wailsjs/go/models.ts` 中 TypeScript `Message` 类的 `is_empty_response` 属性
- **修改** `frontend/src/js/ai-chat.js` 中空回复的处理逻辑：不再保存到库，改为调用 `showNotification` 通知
- **移除** `frontend/src/js/ai-chat.js` 中 `addMessage` 函数的 `isEmptyResponse` 参数及相关占位渲染逻辑
- **移除** `frontend/src/js/ai-chat.js` 中 `saveSessionMessages` 调用时传递的 `is_empty_response` 字段
- **移除** `frontend/src/css/components/ai-chat.css` 中 `.ai-msg-empty` 相关样式

## Impact

- Affected code:
  - `internal/models/ai_message.go` — ORM 模型
  - `internal/services/ai_service.go` — 服务层 DTO 和 DB 操作
  - `frontend/wailsjs/go/models.ts` — TypeScript 声明（自动生成，但手动修改过）
  - `frontend/src/js/ai-chat.js` — 前端消息渲染和保存逻辑
  - `frontend/src/css/components/ai-chat.css` — 空回复占位样式
  - SQLite 数据库 — 需要迁移删除 `is_empty_response` 列

## Requirements

### Requirement: 移除后端字段

The system SHALL remove the `IsEmptyResponse` field from the `AIMessage` ORM model and `Message` DTO.

#### Scenario: 编译验证
- **WHEN** 后端代码编译
- **THEN** 不应出现对 `IsEmptyResponse` 的引用错误

### Requirement: 修改前端空回复处理

The system SHALL handle AI empty responses by showing a notification instead of saving to the database.

#### Scenario: AI 返回空内容
- **WHEN** 流式输出的 `finalContent` 为空或仅空白字符
- **THEN** 系统不再将空消息保存到数据库，改为调用 `showNotification('AI 未返回内容，请尝试重新生成', 'warning')`
- **AND** 该轮用户消息照常入库（现有逻辑不变）

#### Scenario: 加载历史消息
- **WHEN** 切换会话加载历史消息
- **THEN** 不应再出现对 `msg.is_empty_response` 的引用
- **AND** 数据库中已存在的 `is_empty_response` 列数据将被忽略（不做读取）

### Requirement: 数据库迁移

The system SHALL migrate the SQLite database to remove the `is_empty_response` column.

#### Scenario: 启动时自动迁移
- **WHEN** 应用启动执行 `AutoMigrate`
- **THEN** `is_empty_response` 列不再被 GORM 管理
- **AND** 需要手动或通过 GORM 迁移移除该列（通过设置 `-is_empty_response` 注释或手动 SQL）

Note: GORM `AutoMigrate` 默认不会删除已存在的列，需要额外处理。有三种方案：
1. 手动执行 SQL `ALTER TABLE ai_messages DROP COLUMN is_empty_response`
2. 使用 GORM 的 `Migrator().DropColumn()` 方法
3. 保留该列但不读取（不影响功能，首次启动时若列不存在则 GORM 自动创建，若已存在则忽略）

推荐方案：使用 `Migrator().DropColumn()` 在 `AutoMigrate` 后执行，确保干净迁移。
