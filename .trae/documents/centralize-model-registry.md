# 集中注册数据模型（方案 1）实施计划

## Summary

将散落在 InitDB（`internal/database/db.go`）与 ResetDatabase（`app.go`）中的两套模型列表收敛为一个全局注册表 `database.AllModels`，消除"新增模型需改 3 处、漏改导致重置遗漏"的维护问题。

## 现状分析

* **db.go L60**：`InitDB` 的 AutoMigrate 列表（11 个模型，含 NoteVector）

* **app.go L3494-L3506**：`ResetDatabase` 的 DropTable 列表（11 个模型）

* **app.go L3519-L3523**：`ResetDatabase` 的 AutoMigrate 列表（11 个模型）

* **app.go L3514**：`note_tags` 多对多表无对应 model struct，需显式 `DROP TABLE IF EXISTS`

* 外键情况：全库仅 `models.Note.Notebook`（note.go L21）一处显式 foreignKey；其余（含 NoteVector）无外键约束 → DropTable 顺序无强约束，但保持"子表在前"惯例

* `vec-poc/`（实验目录）的 AutoMigrate 与本项目无关，不动

## 变更内容

### 1. 新增 `internal/database/models.go`

```go
package database

import "jot/internal/models"

// AllModels 全部数据模型，按"子表在前"顺序排列
// 供 AutoMigrate（InitDB / ResetDatabase）与 DropTable（ResetDatabase）共用，
// 新增模型只需在此注册一处，两端自动同步
var AllModels = []interface{}{
	&models.AIMessage{},       // 子表：SessionID → AISession
	&models.AISessionConfig{}, // 子表：SessionID → AISession
	&models.AISession{},
	&models.AIPrompt{},
	&models.APIProfile{},
	&models.Todo{},
	&models.Setting{},
	&models.NoteVector{},      // 子表：NoteID → Note
	&models.Note{},            // 子表：NotebookID → Notebook
	&models.Tag{},
	&models.Notebook{},
}
```

### 2. 修改 `internal/database/db.go` L60

`db.AutoMigrate(&models.Note{}, ...)` → `db.AutoMigrate(AllModels...)`

（db.go 与 AllModels 同属 `database` 包，直接引用；可删掉不再使用的 `models` import？——不可，db.go 其他地方仍用 `models.AIPrompt`/`models.Setting` 等，import 保留）

### 3. 修改 `app.go` ResetDatabase

* **DropTable 循环**（L3494-L3510）：`tables := []interface{}{...}` 列表 + for 循环 → 改为遍历 `database.AllModels`：

```go
// 1. 删除所有表（自动处理外键依赖顺序，顺序由 AllModels 子表在前保证）
for _, table := range database.AllModels {
	if err := a.db.Migrator().DropTable(table); err != nil {
		return err
	}
}
```

* **AutoMigrate**（L3518-L3523）：显式列表 → `a.db.AutoMigrate(database.AllModels...)`

* **note\_tags 显式 DROP**（L3514）保留不变（无 model struct，无法进入 AllModels）

## 假设与决策

* AllModels 顺序 = DropTable 安全顺序（子表在前），AutoMigrate 对顺序无要求，同一列表两用

* 不引入文件删除（方案 2）与 Delete 清空（方案 3），保留"删表重建"的出厂语义

* 新文件命名 `models.go`（database 包内职责清晰），不塞进已有 574 行的 db.go

## 验证步骤

1. `go build ./...` 通过
2. `golangci-lint run ./...` 0 issues
3. 手动核对：重构前后 DropTable/AutoMigrate 的模型集合完全一致（11 模型 + note\_tags），无增删
4. （可选）`wails dev` 执行一次重置出厂设置，确认正常重建全部表

