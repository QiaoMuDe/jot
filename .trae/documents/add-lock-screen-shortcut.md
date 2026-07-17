# 锁屏全局快捷键功能计划

## 一、概述

为 Jot 增加一个全局快捷键 `Ctrl+0`，用于快速锁屏。启用锁屏时：按快捷键 → 切换到笔记首页 → 显示锁屏遮罩。未启用锁屏时：按快捷键 → 弹出通知提示用户先启用。

## 二、当前状态分析

### 现有基础设施（全部就绪）

| 能力                                  | 状态                                                 | 关键代码位置                                                                                            |
| ----------------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 锁屏遮罩层 (`#lockScreen`)               | 已实现，含入场动画                                          | [index.html#L1932-L1968](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1932-L1968)              |
| 锁屏显示函数                              | `checkScreenLock()` 中已有展示逻辑                        | [main.js#L7236-L7247](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L7236-L7247)                |
| 锁屏设置 (toggle + 密码)                  | 设置面板已完成                                            | [main.js#L8074-L8088](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L8074-L8088)                |
| 后端 API (`GetAllSettings`)           | 可返回 `screen_lock_enabled` + `screen_lock_password` | [types.go#L85-L86](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go#L85-L86)                 |
| 后端 API (`VerifyScreenLockPassword`) | 已实现 SHA-256 验证                                     | [app.go#L1183-L1193](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1183-L1193)                               |
| 全局快捷键系统                             | `handleKeyboardNavigation()` 处理 Ctrl+1\~8          | [main.js#L5399-L5684](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5399-L5684)                |
| 视图切换                                | `switchView('grid')` + `loadNotes()`               | [main.js#L552-L659](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L552-L659)                    |
| 通知系统                                | `window.showNotification(msg, type)`               | [notification.js#L154-L158](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/notification.js#L154-L158) |
| 锁屏屏蔽快捷键                             | 已在锁屏打开时屏蔽 Ctrl+数字                                  | [main.js#L5593-L5597](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5593-L5597)                |

## 三、变更方案

### 修改文件：`frontend/src/main.js`

#### 3.1 在 `handleKeyboardNavigation` 的 Ctrl+数字块中新增 `case '0'`

**位置**：`main.js` 第 5641 行（`case '8'` 之后，`switch` 闭合之前）

**逻辑**：

```
case '0':
    e.preventDefault();

    // 1. 获取设置
    const lockCfg = await window.go.main.App.GetAllSettings();
    const lockEnabled = lockCfg.screen_lock_enabled === true || lockCfg.screen_lock_enabled === 'true';
    const hasPassword = lockCfg.screen_lock_password && lockCfg.screen_lock_password !== '';

    // 2. 未启用锁屏 → 提示
    if (!lockEnabled) {
        nm.show('请先在「设置 → 锁屏密码」中启用锁屏功能', 'warning');
        return;
    }

    // 3. 已启用但无密码 → 提示
    if (!hasPassword) {
        nm.show('请先设置锁屏密码后再使用锁屏功能', 'warning');
        return;
    }

    // 4. 已启用且有密码 → 切首页 → 锁屏
    switchView('grid');
    await loadNotes();

    const lockScreen = document.getElementById('lockScreen');
    if (!lockScreen) return;
    lockScreen.style.display = 'flex';
    requestAnimationFrame(() => lockScreen.classList.add('entering'));
    setTimeout(() => lockScreen.classList.remove('entering'), 700);
    setTimeout(() => document.getElementById('lockPasswordInput')?.focus(), 100);
    return;
```

#### 3.2 锁屏状态下快捷键已被屏蔽（无需额外修改）

现有代码第 5593-5597 行在锁屏遮罩显示时直接 `return`，Ctrl+0 会自动被屏蔽。

```js
const lockScreen = document.getElementById('lockScreen');
if (lockScreen && lockScreen.style.display !== 'none') {
    return;
}
```

#### 3.3 关于 `nm` 变量的可用性

`nm`（`NotificationManager` 实例）在第 272 行定义，作用域为 `main.js` 模块级，`handleKeyboardNavigation` 在同一文件中，可直接使用 `nm.show()`。

## 四、不涉及的变更

| 项        | 说明                             |
| -------- | ------------------------------ |
| 后端代码     | 无需修改；`GetAllSettings` 已可返回所需字段 |
| HTML/CSS | 无需修改；锁屏 UI 元素已存在               |
| 其他快捷键    | 不影响现有 Ctrl+1\~8 等快捷键逻辑         |
| 快捷键帮助页   | Ctrl+0 暂时不加入快捷键列表（如需可后续补充）     |

## 五、假设与决策记录

| 问题        | 用户决策                                                |
| --------- | --------------------------------------------------- |
| 快捷键键位     | `Ctrl+0`                                            |
| 锁屏前行为     | 先切到笔记首页（`switchView('grid')` + `loadNotes()`），再显示锁屏 |
| 锁屏状态下的快捷键 | 被现有屏蔽逻辑自动拦截，不生效                                     |
| 启用但无密码    | 不显示锁屏，弹出 warning 通知提示先设置密码                          |

## 六、验证步骤

1. **未启用锁屏时按 Ctrl+0**：应弹出黄色 warning 通知 "请先在「设置 → 锁屏密码」中启用锁屏功能"
2. **启用锁屏但未设密码时按 Ctrl+0**：应弹出黄色 warning 通知 "请先设置锁屏密码后再使用锁屏功能"
3. **启用锁屏且已设密码时按 Ctrl+0**：应切换到笔记首页（网格视图），接着显示锁屏遮罩层，密码输入框自动聚焦
4. **锁屏状态下按 Ctrl+0**：无任何反应（被屏蔽）
5. **Ctrl+0 不影响 Ctrl+1\~8 等其他快捷键**：需逐一验证

