# Tasks

- [x] Task 1: 移动子包目录并更新 import 路径
  - [x] 将 `database/` → `internal/database/`，更新 `db.go` 中的 import
  - [x] 将 `fontutil/` → `internal/fontutil/`（无内部 import 需改）
  - [x] 将 `models/` → `internal/models/`（无内部 import 需改）
  - [x] 将 `services/` → `internal/services/`，更新各文件的 import
  - [x] 更新 `app.go` 中的 import 路径
  - [x] 删除旧的根目录子包文件夹

- [x] Task 2: 验证编译通过
  - [x] 执行 `go build ./...` 确认无误
  - [x] 执行 `go vet ./...` 确认无误
