# 联网搜索来源结果数设置 Spec

## Why

AI 助手联网搜索的来源结果数目前硬编码为 5，用户无法按需调整。将此值迁移到设置页，存入数据库，使前端可在设置页中修改，后端在搜索时动态读取。

## What Changes

- **修改**：`SearchWeb` 函数新增 `maxResults` 参数，替代硬编码的 `5`
- **新增**：后端绑定 `GetAISearchResultLimit` / `SetAISearchResultLimit`，读写 `ai_search_result_limit` 设置项
- **新增**：设置页搜索区域增加"搜索结果数"输入项，默认 5，可自定义
- **新增**：前端 `loadAISettings` 中加载该配置并填入输入框，用户修改后自动保存
- **影响**：设置页默认值 5 写入数据库后，所有联网搜索均按新值返回相应条数的结果

## Impact

- Affected specs: `add-ai-assistant`, `add-web-search-tavily`
- Affected code: `app.go`, `internal/services/search_service.go`, `frontend/index.html`, `frontend/src/main.js`

---

## ADDED Requirements

### Requirement: 后端可配置搜索结果数

The system SHALL replace the hardcoded `MaxResults: 5` in `SearchWeb` with a configurable value read from `ai_search_result_limit` setting.

#### Scenario: 读取搜索结果数配置
- **WHEN** `CallAIStream` runs with search enabled
- **THEN** the system reads setting key `ai_search_result_limit` from the database
- **AND** if the value is empty or not a valid positive integer, uses 5 as default
- **AND** passes this value to `SearchWeb` as the `maxResults` parameter
- **AND** `SearchWeb` sets `MaxResults` on the Tavily search query accordingly

#### Scenario: 设置搜索结果数配置
- **WHEN** frontend calls `SetAISearchResultLimit(limit)`
- **THEN** the system stores the value as setting key `ai_search_result_limit`
- **AND** if limit is < 1, returns an error "搜索结果数必须大于 0"
- **AND** if limit is > 20, returns an error "搜索结果数不能超过 20"

### Requirement: 设置页 UI

The system SHALL display a "搜索结果数" input field in the 联网搜索 section of the AI settings page.

#### Scenario: 加载设置
- **WHEN** `loadAISettings()` runs
- **THEN** call `GetAISearchResultLimit()` and display the value in the input field

#### Scenario: 修改设置
- **WHEN** user changes the input value
- **THEN** call `SetAISearchResultLimit(newValue)` to persist
- **AND** show a success/error toast notification

---

## MODIFIED Requirements

### Requirement: `SearchWeb`（已存在）

Add a `maxResults int` parameter to the function signature, replacing the hardcoded `5`.

### Requirement: `app.go` 绑定（已存在）

Add `GetAISearchResultLimit()` (returns int) and `SetAISearchResultLimit(limit int)` bindings, following the same pattern as `GetAIRefMaxChars` / `SetAIRefMaxChars`.

---

## REMOVED Requirements

None.
