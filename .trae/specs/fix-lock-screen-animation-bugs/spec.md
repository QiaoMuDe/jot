# Fix Lock Screen & Settings Animation Bugs Spec

## Why
锁屏页退出动画背景瞬间消失（opacity 无 transition）、入场动画尾部元素闪烁（entering 类过早移除）、以及设置页锁屏密码行的显示/隐藏过渡完全无效（CSS transition 被 display:none 阻断），影响用户体验。

## What Changes

### 1. 修复退出动画背景 opacity 过渡
- `modals.css` L848：`.lock-screen-backdrop` 的 `transition` 添加 `opacity 0.4s ease-out`

### 2. 修复入场动画尾部闪烁
- `main.js` L7129：entering 清除超时从 500ms 延长至 700ms，覆盖最长 stagger 动画(0.25s 延迟 + 0.4s 动画)

### 3. 修复 screenLockPasswordRow 过渡无效
- 重构设置页锁屏密码行的显示/隐藏方案：不再使用 `display: none`，改用 `visibility` + `max-height` + `opacity` + `margin-top` + `padding` 过渡
- `settings-panel.css` L1024-L1038：重写 CSS 规则，使用 `[style*="visibility: hidden"]` 属性选择器或 class 切换
- `main.js` L2261-L2264：不再使用 `style.display`，改用 class 切换来控制隐藏状态
- `main.js` L7968-L7970：设置加载时同样使用 class 切换

### 4. 关闭锁屏密码时添加确认弹窗
- `main.js` L2265-L2268：调用 confirm 弹窗确认后再清空密码

## Impact
- Affected specs: 锁屏页面动画、设置页面交互
- Affected code:
  - `frontend/src/css/components/modals.css`：退出动画 transition
  - `frontend/src/css/components/settings-panel.css`：密码行显示/隐藏 CSS
  - `frontend/src/main.js`：entering 超时、toggle 交互、设置加载

## ADDED Requirements

### Requirement: 锁屏退出动画平滑
The system SHALL smoothly fade the lock screen backdrop when unlocking.

#### Scenario: 解锁成功
- **WHEN** user enters correct password and clicks unlock
- **THEN** backdrop opacity fades from 1 to 0 with 0.4s ease-out transition (in sync with backdrop-filter)
- **AND** content fades/moves up with 0.45s transition (unchanged)

### Requirement: 锁屏入场动画完整播放
The system SHALL allow all stagger entrance animations to complete before removing the entering state.

#### Scenario: 锁屏显示
- **WHEN** lock screen appears
- **THEN** all 5 stagger elements (icon, title, input-wrap, btn, exit-btn) complete their entrance animation
- **AND** no element snaps to final position prematurely

### Requirement: 设置页密码行显示/隐藏动画
The system SHALL animate the screen lock password row when toggling lock on/off.

#### Scenario: 启用锁屏密码
- **WHEN** user toggles screen lock ON
- **THEN** password row smoothly expands (max-height/opacity transition) over 0.2s

#### Scenario: 禁用锁屏密码
- **WHEN** user toggles screen lock OFF
- **THEN** password row smoothly collapses (max-height/opacity transition) over 0.2s
- **AND** after the collapse animation completes, the element is hidden

### Requirement: 禁用锁屏确认
The system SHALL confirm before disabling screen lock and clearing the password.

#### Scenario: 点击 toggle 关闭
- **WHEN** user clicks toggle to disable screen lock
- **THEN** a confirmation dialog appears asking "确定关闭锁屏密码？关闭后已设置的密码将被清除。"
- **AND** the action proceeds only if user confirms
