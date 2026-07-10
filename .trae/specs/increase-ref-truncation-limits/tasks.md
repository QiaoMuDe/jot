# Tasks

- [x] Task 1: 更新后端 `ai_ref_max_chars` 的默认值和校验边界
  - `internal/database/db.go`：`"5000"` → `"10000"`
  - `internal/services/types.go`：`GetAllSettings()` fallback `5000` → `10000`
  - `internal/services/types.go`：`SaveAllSettings()` 中 `< 1` 重置值 `5000` → `10000`，`> 50000` 限制 → `> 100000`
  - `app.go`：`GetAIRefMaxChars()` fallback `5000` → `10000`，`> 50000` → `> 100000`
  - `app.go`：`SetAIRefMaxChars()` `> 50000` → `> 100000`

- [x] Task 2: 更新前端 `aiRefMaxChars` 的默认值和 max 属性
  - `frontend/index.html`：input `value="5000"` → `value="10000"`, `max="50000"` → `max="100000"`
  - `frontend/src/main.js`：change 事件中重置值 `5000` → `10000`

## Task Dependencies

- Task 1 和 Task 2 无依赖关系，可并行执行
