# 数据库路径变更：`./data/jot.db` → `~/.jot/data/jot.db`

## 摘要

将数据库路径从程序运行目录下的 `data/jot.db` 改为用户家目录下的 `.jot/data/jot.db`。不进行数据迁移。

## 当前状态

| 文件                   | 行号  | 当前代码                             |
| -------------------- | --- | -------------------------------- |
| `app.go`             | L27 | `database.InitDB("data/jot.db")` |
| `tools/seed/main.go` | L17 | `dbPath := "data/jot.db"`        |

## 改动

### 1. 新增 `DefaultDBPath()` 函数

**文件：** `internal/database/db.go`

```go
// DefaultDBPath 返回默认数据库路径：~/.jot/data/jot.db
func DefaultDBPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("cannot get user home directory: %w", err)
    }
    return filepath.Join(home, ".jot", "data", "jot.db"), nil
}
```

### 2. 更新 `app.go`

```go
dbPath, err := database.DefaultDBPath()
if err != nil {
    panic(err)
}
db, err := database.InitDB(dbPath)
```

### 3. 更新 `tools/seed/main.go`

默认路径改为调用 `DefaultDBPath()`，命令行参数仍可覆盖。

### 4. 更新 seed 测试笔记内容

文件中第 126 行的测试笔记内容里的 `data/jot.db` 改为 `~/.jot/data/jot.db`。

## 影响范围

| 文件                        | 改动                       |
| ------------------------- | ------------------------ |
| `internal/database/db.go` | 新增 `DefaultDBPath()`     |
| `app.go`                  | 调用 `DefaultDBPath()`     |
| `tools/seed/main.go`      | 默认路径改用 `DefaultDBPath()` |

## 验证

1. `go build` 编译通过
2. `go run tools/seed/main.go` 在 `~/.jot/data/jot.db` 创建数据库

