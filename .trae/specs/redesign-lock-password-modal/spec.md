# Redesign Lock Screen Password Modal Spec

## Why
当前锁屏密码设置使用 inline 输入框 + 自动保存（失焦触发），存在两个问题：
1. **密码已设置后输入框始终为空**，显示/隐藏密码按钮点击后看到的是空白输入框，功能鸡肋
2. **显示按钮 click 事件触发 input blur → change → 意外保存**，用户没来得及查看密码就被保存清空

## What Changes
- **后端**: 新增 `SetScreenLockPassword(oldPwd, newPwd string) error` Wails 绑定方法
- **前端 HTML**: 将密码输入行改为状态显示行 +「修改密码」按钮；新增独立密码修改模态框
- **前端 CSS**: 新增密码修改模态框样式（复用模态框设计系统）
- **前端 JS**: 移除 `change` 自动保存逻辑；新增模态框打开/关闭/保存逻辑

## Impact
- Affected specs: 锁屏密码设置流程
- Affected code:
  - `app.go` — 新增 `SetScreenLockPassword` 方法
  - `frontend/index.html` — 替换密码输入行 + 新增模态框 HTML
  - `frontend/src/css/components/settings-panel.css` — 新增模态框样式（或在 modals.css 中追加）
  - `frontend/src/main.js` — 移除自动保存事件，新增模态框交互逻辑
  - `frontend/src/css/components/modals.css` — 可选：新增模态框样式

## REMOVED Requirements
### Requirement: Inline Password Auto-Save
**Reason**: 方案F 用独立模态框替代 inline 输入 + 失焦保存
**Migration**: 
- 移除 `screenLockPwdInput` 的 `change` 事件监听器
- 移除 `screenLockPwdToggleBtn` 的 `click` 事件监听器（显示/隐藏按钮不再需要）
- 保留后端 `VerifyScreenLockPassword` 方法不变

## ADDED Requirements
### Requirement: Password Status Display
The system SHALL display lock screen password status in settings:
- **WHEN** no password has been set and toggle is enabled
- **THEN** show "未设置" status label + "设置密码" button

- **WHEN** password has been set and toggle is enabled
- **THEN** show "已启用 ✓" status label + "修改密码" button

### Requirement: Password Change Modal
The system SHALL provide a modal overlay for password modification:
- **WHEN** user clicks "设置密码" or "修改密码" button
- **THEN** open a modal overlay with:
  - Old password field (required only if a password is already set)
  - New password field (required, auto-focus)
  - Confirm password field (required, must match new password)
  - Show/hide eye buttons for each field (mousedown/mouseup hold-to-show)
  - Save button (disabled until all required fields valid)
  - Cancel button (closes modal)

#### Scenario: First-time password setup
- **GIVEN** no password has been set
- **WHEN** user opens modal, enters new password + confirm, clicks Save
- **THEN** call `SetScreenLockPassword("", newPwd)` → backend hashes and saves → modal closes → status shows "已启用 ✓"
- **AND** if new password and confirm do not match, show inline error, do not close modal

#### Scenario: Change existing password
- **GIVEN** a password is already set
- **WHEN** user enters old + new + confirm, clicks Save
- **THEN** call `SetScreenLockPassword(oldPwd, newPwd)` → backend verifies old, hashes new, saves → modal closes → toast "密码已修改"
- **AND** if old password is incorrect, show inline error "旧密码不正确"
- **AND** if new password and confirm do not match, show inline error "两次密码输入不一致"

#### Scenario: Cancel
- **WHEN** user clicks Cancel or clicks overlay background
- **THEN** close modal, clear all input fields

### Requirement: Backend SetScreenLockPassword
The system SHALL provide `SetScreenLockPassword(oldPwd, newPwd string) error`:
- **WHEN** stored password is not empty and `oldPwd` hash does not match
- **THEN** return error "旧密码不正确"
- **WHEN** `newPwd` is empty
- **THEN** return error "新密码不能为空"
- **WHEN** verification succeeds
- **THEN** hash `newPwd` with salt, save to DB via `settingService.Set("screen_lock_password", hash)`, return nil
