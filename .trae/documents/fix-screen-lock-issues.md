# 锁屏密码问题修复计划

## 问题列表

| # | 问题 | 类型 | 严重度 |
|---|------|------|--------|
| 1 | 设置页输入密码后无保存提示，且库中密码字段为空 | Bug | 高 |
| 2 | 锁屏界面密码输入框显示浏览器原生密码眼睛图标，与自定义按钮重叠 | Bug | 中 |
| 3 | 关闭锁屏功能时未清空密码字段 | 需求 | 低 |

---

## 问题分析

### 问题 1: 保存后无提示且密码为空

**根因**: `saveSettings()` 函数收集 `screen_lock_password` 时，使用的是 `#screenLockPasswordInput?.value || ''`。当 toggle 开关被点击时也会调用 `saveSettings()`，而此时密码输入框可能还未输入任何内容（值为空字符串 `""`），后端收到空字符串后执行"保留旧值"逻辑（旧值也是空）。而用户随后按下回车保存密码时：

1. 调用 `saveSettings()` → 收集密码值 → 后端哈希存储 → 正确
2. 调用 `nm.show('锁屏密码已保存', 'success')` → 显示通知

但实际上问题是：**当用户在密码输入框中按下回车时，除了触发 keydown handler，也可能触发表单的默认提交行为导致页面刷新或其他副作用**。

此外，`saveSettings()` 的 try-catch 吞没了所有错误，如果 Wails 绑定不可用则会静默失败。

**修复方案**:
1. 在 `keydown` 事件处理中添加 `e.preventDefault()` 防止默认行为
2. 确保 `nm.show` 在 `saveSettings()` 完成后正确执行

### 问题 2: 浏览器原生密码眼睛图标

**根因**: Edge/WebView2 为 `input[type="password"]` 自动添加了原生的密码显示/隐藏按钮（`::-ms-reveal`）。该原生按钮与自定义的 `#lockPasswordToggle` 按钮叠放在一起，导致视觉冲突。

**修复方案**: 在 CSS 中隐藏原生密码切换按钮：
```css
.lock-screen-input::-ms-reveal,
.lock-screen-input::-ms-clear {
    display: none;
}
```

### 问题 3: 关闭锁屏时未清空密码

**根因**: 当用户关闭 toggle 时，`saveSettings()` 被调用，收集 `screen_lock_password` 值为密码输入框当前内容。后端由于该值非空，会再次哈希存储，导致密码保留了下来。用户希望关闭时密码也被清空。

**修复方案**: 
- 在 toggle 关闭时，清空密码输入框的值，然后调用 `saveSettings()`
- 在后端 `SaveAllSettings()` 中，当 `ScreenLockEnabled` 为 `false` 时也将 `ScreenLockPassword` 置空

---

## 修改计划

### 修改 1: `frontend/src/main.js` — 输入框保存事件修复

**文件**: `frontend/src/main.js`，约 2300 行

```javascript
// 修改前
screenLockPwdInput.addEventListener('keydown', async (e) => {
    if (e.key === 'Enter') {
        await saveSettings();
        nm.show('锁屏密码已保存', 'success');
    }
});

// 修改后
screenLockPwdInput.addEventListener('keydown', async (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        await saveSettings();
        nm.show('锁屏密码已保存', 'success');
    }
});
```

### 修改 2: `frontend/src/main.js` — 关闭 toggle 时清空密码

**文件**: `frontend/src/main.js`，约 2280 行

```javascript
// 修改前
const isActive = toggleSwitch.classList.toggle('active');
const pwdRow = document.getElementById('screenLockPasswordRow');
if (pwdRow) {
    pwdRow.style.display = isActive ? '' : 'none';
}
await saveSettings();

// 修改后
const isActive = toggleSwitch.classList.toggle('active');
const pwdRow = document.getElementById('screenLockPasswordRow');
if (pwdRow) {
    pwdRow.style.display = isActive ? '' : 'none';
}
// 关闭锁屏时清空密码输入框，使后端清除密码
if (!isActive) {
    const pwdInput = document.getElementById('screenLockPasswordInput');
    if (pwdInput) pwdInput.value = '';
}
await saveSettings();
```

### 修改 3: `internal/services/types.go` — 关闭锁屏时清空密码

**文件**: `internal/services/types.go`，`SaveAllSettings()` 方法

```go
// 修改密码处理逻辑：在现有的 oldPassword 逻辑之前添加
// 如果锁屏被关闭，清空密码
if !cfg.ScreenLockEnabled {
    cfg.ScreenLockPassword = ""
}
```

This means when `ScreenLockEnabled` is `false`, we always clear the password.

### 修改 4: `frontend/src/css/components/modals.css` — 隐藏浏览器原生密码切换按钮

**文件**: `frontend/src/css/components/modals.css`，在 Section L 中添加

```css
/* 隐藏浏览器原生密码显示按钮 */
.lock-screen-input::-ms-reveal,
.lock-screen-input::-ms-clear {
    display: none;
}
```

---

## 验证步骤

1. **问题 1 验证**:
   - 打开设置页 → 开启锁屏密码 → 输入密码后按回车 → 应看到"锁屏密码已保存"通知
   - 重启应用 → 应显示锁屏 → 输入密码后应能进入

2. **问题 2 验证**:
   - 启动后显示锁屏界面 → 密码输入框右侧不应有重叠的眼睛图标

3. **问题 3 验证**:
   - 设置页关闭锁屏密码开关 → 重启应用 → 不应显示锁屏
   - 重新开启开关 → 需要重新设置密码（旧密码已清空）
