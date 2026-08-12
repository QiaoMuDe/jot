# Tasks

- [x] Task 1: 新增共享辅助文件 `internal/fontutil/fonts_fc.go`（`//go:build linux || darwin`）
  - [x] SubTask 1.1: 实现 `fontFamiliesFromFCList() []string`：执行 `fc-list --format="%{family}\n"`（带 5s 超时），按行解析逗号分隔 family，trim 空格、去重
  - [x] SubTask 1.2: 实现 `scanFontDirs(dirs ...string) []string`：递归遍历目录收集 `.ttf/.otf/.ttc` 文件，以文件基名去扩展名作为 family 名，去重
  - [x] SubTask 1.3: 实现 `uniqueSorted(names []string) []string`：去重 + 字典序排序（供三个平台统一使用）

- [x] Task 2: 新增 `internal/fontutil/fonts_linux.go`
  - [x] SubTask 2.1: 实现 `GetFonts() []string`：优先 `fontFamiliesFromFCList()`，非空直接返回
  - [x] SubTask 2.2: fc-list 失败时回退 `scanFontDirs("/usr/share/fonts", "/usr/local/share/fonts", "~/.fonts")`，最终返回 `uniqueSorted` 结果（空则返回空列表）

- [x] Task 3: 新增 `internal/fontutil/fonts_darwin.go`
  - [x] SubTask 3.1: 实现 `GetFonts() []string`：优先 `fontFamiliesFromFCList()`，非空直接返回
  - [x] SubTask 3.2: fc-list 失败时执行 `system_profiler SPFontsDataType -json`（带超时），解析 JSON 数组的 `family` 字段（解析失败静默忽略）
  - [x] SubTask 3.3: system_profiler 失败时回退 `scanFontDirs("/System/Library/Fonts", "/Library/Fonts", "~/.fonts")`，最终返回 `uniqueSorted` 结果（空则返回空列表）

- [x] Task 4: 编译与静态检查验证
  - [x] SubTask 4.1: 本机（Windows）`go build ./...` 通过（回归验证 Windows 不受影响）
  - [x] SubTask 4.2: `GOOS=linux go build ./...` 交叉编译通过
  - [x] SubTask 4.3: `GOOS=darwin go build ./...` 交叉编译通过
  - [x] SubTask 4.4: `golangci-lint run ./internal/fontutil/...` 无告警（含注释错误处理）

# Task Dependencies

- [Task 2] depends on [Task 1]（复用 fc-list 解析与目录扫描辅助）
- [Task 3] depends on [Task 1]（同上）
- [Task 4] depends on [Task 2]、[Task 3]
