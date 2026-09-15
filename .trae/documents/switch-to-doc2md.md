# 切换 md 转换库为独立库 doc2md

## Summary

项目当前通过 `replace` 指令把搬入项目内的 markitdown 副本（`internal/markitdown/`，模块名 `github.com/conductor-oss/markitdown`）作为转换库使用。该库现已独立发布为 `gitee.com/MM-Q/doc2md v1.0.0`（包名仍为 `markitdown`，API 与现有用法完全兼容）。本次切换：替换 import、移除 replace、删除内嵌副本、验证构建。

## Current State Analysis

| 项目    | 现状                                                                                                                                                                                                                                                                            |
| ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 内嵌副本  | `internal/markitdown/`（自带 go.mod、cmd、testdata、golden、LICENSE，共 50+ 文件），通过主 `go.mod` L138 的 `replace github.com/conductor-oss/markitdown v0.0.1 => ./internal/markitdown` 生效                                                                                                   |
| 唯一使用方 | [converter.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/converter/converter.go)：import `markitdownlib "github.com/conductor-oss/markitdown"`，使用 `New()` / `StreamInfo{Extension, Filename, LocalPath}` / `ConvertReader(f, info)` / `IsUnsupportedFormat(err)` / `r.Markdown` |
| 上层调用  | [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go) 仅调用 `converter.IsOfficeFile` / `converter.ConvertToMarkdown`（L1968-1977 聊天上传、L4089/4232-4243 笔记导入），不直接接触转换库                                                                                                                  |
| 目标库   | `gitee.com/MM-Q/doc2md v1.0.0`，已确认可从 gitee 拉取；要求 Go ≥ 1.24（项目为 1.26.0，满足）                                                                                                                                                                                                     |

**API 兼容性**：doc2md 导出 API 与现用 API 一一对应（`New`、`ConvertReader`、`StreamInfo` 同名字段、`IsUnsupportedFormat`、`Result.Markdown`），`internal/converter/converter.go` 的逻辑无需改动，仅需替换 import 路径。

## Proposed Changes

### 1. 修改 `internal/converter/converter.go`

* import 从 `markitdownlib "github.com/conductor-oss/markitdown"` 改为 `markitdownlib "gitee.com/MM-Q/doc2md"`（包名同为 `markitdown`，别名保留，其余代码零改动）。

### 2. 修改主 `go.mod`

* 删除 L138 的 `replace github.com/conductor-oss/markitdown v0.0.1 => ./internal/markitdown`。

* 执行 `go get gitee.com/MM-Q/doc2md@v1.0.0` 引入依赖，随后 `go mod tidy` 自动移除 `github.com/conductor-oss/markitdown` 依赖项。

### 3. 删除内嵌副本 `internal/markitdown/` 整个目录

* 含其独立 go.mod/go.sum、cmd、testdata（含 golden 与二进制样例）、LICENSE、README 等。独立成库后此目录完全冗余。

### 4. 顺带更新 `app.go` 中 4 处提及 "markitdown" 的注释（L1895、L1969、L4059、L4233）

* 改为提及 `doc2md`，保持注释与实际依赖一致（仅注释，不动逻辑）。

## Assumptions & Decisions

* **版本**：锁定 `v1.0.0`（当前唯一已发布版本）。

* **行为不变**：`officeExtensions` 支持范围（.docx/.xlsx/.xls/.pptx/.pdf/.epub）不变；60 秒超时、panic 拦截、错误翻译逻辑全部保留。doc2md 虽支持 ZIP/HTML 等更多格式，但项目约定 ZIP 已弃用、纯文本走二进制检测兜底，故不扩大支持范围。

* **不做**：不引入 `WithKeepDataURIs`/`WithStyleMap` 等新选项（默认行为与现有一致）；不改前端；不跑 `wails build`（纯 Go 依赖切换，`go build ./...` 足以验证，前端资源不受影响）。

## Verification Steps

1. `go build ./...` — 编译通过，确认无 `github.com/conductor-oss/markitdown` 残留引用。
2. `go vet ./...` — 无告警。
3. `go test ./internal/converter/... ./...` 中相关包 — 通过（若存在测试）。
4. `go mod tidy && git diff go.mod go.sum` — 确认 replace 已删、新依赖 `gitee.com/MM-Q/doc2md v1.0.0` 已加入、旧依赖已移除。
5. 确认 `internal/markitdown/` 目录已删除。

