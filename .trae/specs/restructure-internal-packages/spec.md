# 整合 Go 子包到 internal 目录 Spec

## Why

当前 Go 子包（`database/`、`fontutil/`、`models/`、`services/`）散落在项目根目录，与 `main.go` / `app.go` 混在一起。按照 Go 惯例将内部包统一放入 `internal/` 目录，明确访问边界，提升项目结构清晰度。

## What Changes

- 将 `database/` 目录整体移至 `internal/database/`
- 将 `fontutil/` 目录整体移至 `internal/fontutil/`
- 将 `models/` 目录整体移至 `internal/models/`
- 将 `services/` 目录整体移至 `internal/services/`
- 更新所有 import 路径，从 `jot/<pkg>` 改为 `jot/internal/<pkg>`
- 删除旧的根目录子包文件夹

**不涉及**：
- 包名保持不变（`package database` / `package fontutil` / `package models` / `package services`）
- 代码逻辑、类型定义、函数签名均不变
- `app.go` 和 `main.go` 保持在根目录不动

## Impact

- Affected specs: 无直接影响
- Affected code:
  - `app.go` — 4 处 import 路径变更
  - `services/note_service.go` — import 路径变更
  - `services/tag_service.go` — import 路径变更
  - `services/setting_service.go` — import 路径变更
  - `database/db.go` — import 路径变更
- 后端行为无变化，纯目录结构调整

## ADDED Requirements

### Requirement: 子包移入 internal 目录

系统 SHALL 将所有内部 Go 子包置于 `internal/` 目录下。

#### Scenario: 目录移动
- **WHEN** 项目结构重组后
- **THEN** `database/`、`fontutil/`、`models/`、`services/` 应位于 `internal/` 之下

#### Scenario: 编译通过
- **WHEN** 执行 `go build`
- **THEN** 编译成功无报错

## MODIFIED Requirements

无
