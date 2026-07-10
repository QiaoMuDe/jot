# 调整引用截断默认值和上限 Spec

## Why

当前 `ai_ref_max_chars` 默认值为 5000，上限为 50000。5000 字符对于单条笔记来说偏小（约 2-3 页 A4），配合新的 1MB 文件上传限制显得更不合理。需要将默认值提升到 10000，上限提升到 100000。

## What Changes

- **默认值** `ai_ref_max_chars`：5000 → 10000
- **上限** `ai_ref_max_chars`：50000 → 100000
- 更新前后端所有相关的默认值和校验边界

## Impact

- Affected code: `internal/database/db.go`、`internal/services/types.go`、`app.go`、`frontend/index.html`、`frontend/src/main.js`

## MODIFIED Requirements

### Requirement: 引用截断默认值

系统 SHALL 将 `ai_ref_max_chars` 的默认值从 5000 改为 10000。

#### 修改点
- `internal/database/db.go` 默认值 `"5000"` → `"10000"`
- `internal/services/types.go` `GetAllSettings()` fallback `5000` → `10000`
- `internal/services/types.go` `SaveAllSettings()` `< 1` 时重置值 `5000` → `10000`
- `app.go` `GetAIRefMaxChars()` fallback `5000` → `10000`
- `frontend/index.html` input `value="5000"` → `value="10000"`
- `frontend/src/main.js` change 事件中重置值 `5000` → `10000`

### Requirement: 引用截断上限

系统 SHALL 将 `ai_ref_max_chars` 的上限从 50000 改为 100000。

#### 修改点
- `internal/services/types.go` `SaveAllSettings()` `> 50000` → `> 100000`
- `app.go` `GetAIRefMaxChars()` `> 50000` → `> 100000`
- `app.go` `SetAIRefMaxChars()` `> 50000` → `> 100000`
- `frontend/index.html` input `max="50000"` → `max="100000"`
- `frontend/src/main.js` change 事件中 `> 50000` → `> 100000`
