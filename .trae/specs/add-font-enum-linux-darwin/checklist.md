# Checklist

- [x] `internal/fontutil/fonts_fc.go` 存在，含 `//go:build linux || darwin` 构建标签，提供 fc-list 解析 / 目录扫描 / 去重排序三个辅助函数
- [x] `internal/fontutil/fonts_linux.go` 存在，实现 `GetFonts() []string`（fc-list 优先，目录扫描兜底）
- [x] `internal/fontutil/fonts_darwin.go` 存在，实现 `GetFonts() []string`（fc-list → system_profiler → 目录扫描三级回退）
- [x] 三平台 `GetFonts()` 契约一致：返回去重、排序的 `[]string`，失败返回空列表而非 panic
- [x] `fonts_windows.go` 未改动，Windows 行为无回归
- [x] 本机 `go build ./...`（Windows）编译通过
- [x] `GOOS=linux go build ./...` 交叉编译通过，无 undefined symbol
- [x] `GOOS=darwin go build ./...` 交叉编译通过，无 undefined symbol
- [x] `golangci-lint run ./internal/fontutil/...` 无告警
