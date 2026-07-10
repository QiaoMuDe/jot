# Checklist

- [x] `internal/database/db.go` 默认值改为 `"10000"`
- [x] `internal/services/types.go` `GetAllSettings()` fallback 改为 `10000`
- [x] `internal/services/types.go` `SaveAllSettings()` `< 1` 重置值改为 `10000`，`> 100000` 限制为 `100000`
- [x] `app.go` `GetAIRefMaxChars()` fallback 改为 `10000`，上限改为 `100000`
- [x] `app.go` `SetAIRefMaxChars()` 上限改为 `100000`
- [x] `frontend/index.html` input value 改为 `"10000"`，max 改为 `"100000"`
- [x] `frontend/src/main.js` change 事件中重置值改为 `10000`
- [x] 编译通过
