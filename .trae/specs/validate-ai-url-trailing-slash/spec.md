# AI Base URL 尾斜杠校验 Spec

## Why

后端在 `internal/aicli/client.go` 中已通过 `strings.TrimRight(cfg.BaseURL, "/")` 自动处理尾斜杠，但用户输入时缺少前端反馈——用户不知道尾斜杠会被自动清除，且某些场景下可能导致 URL 拼接异常。需要在前端三个入口添加实时校验，以红色抖动提示用户不能以斜杠结尾。

## What Changes

1. **CSS 动画**：在 `animations.css` 中新增 `shake` 抖动动画 + 输入框红色边框样式
2. **设置页 Base URL 输入框**：`change` 事件中校验尾斜杠，非法时阻止保存并抖动提示
3. **预设新增/编辑弹窗**：`savePresetModal()` 中校验 `#presetModalURL` 尾斜杠，非法时阻止保存并抖动提示

## Impact

* Affected specs: AI 设置、预设配置管理

* Affected code: `frontend/src/css/animations.css`, `frontend/src/main.js`

## ADDED Requirements

### Requirement: Base URL 尾斜杠校验

系统 SHALL 在三个入口校验 AI Base URL 输入是否以斜杠结尾，非法时以红色抖动提示用户且阻止保存。

#### Scenario: 设置页直接输入 URL

* **GIVEN** 用户在设置页的 Base URL 输入框中输入以 `/` 结尾的 URL（如 `https://api.openai.com/v1/`）

* **WHEN** 输入框触发 `change` 事件

* **THEN** 输入框添加红色边框 + 抖动动画

* **AND** 不执行保存操作

* **AND** Toast 提示「API 地址不能以斜杠结尾」

* **AND** 用户修正后恢复正常样式

#### Scenario: 新增预设配置

* **GIVEN** 用户在预设新增弹窗中输入以 `/` 结尾的 URL

* **WHEN** 点击「保存」按钮

* **THEN** URL 输入框添加红色边框 + 抖动动画

* **AND** 不执行保存操作

* **AND** Toast 提示「API 地址不能以斜杠结尾」

#### Scenario: 编辑预设配置

* **GIVEN** 用户在预设编辑弹窗中修改 URL 为以 `/` 结尾

* **WHEN** 点击「保存」按钮

* **THEN** URL 输入框添加红色边框 + 抖动动画

* **AND** 不执行保存操作

* **AND** Toast 提示「API 地址不能以斜杠结尾」

#### Scenario: 用户修正后恢复正常

* **GIVEN** 输入框处于红色抖动错误状态

* **WHEN** 用户修改输入内容，不再以 `/` 结尾

* **THEN** 红色边框和抖动样式自动移除

* **AND** 后续保存操作正常执行

