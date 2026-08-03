# 启动时未完成待办弹窗提示

## Summary

应用每次启动时，若存在未完成的待办事项，弹出确认框提示数量；点击"去查看"跳转到待办清单视图，点击"取消"关闭。若启用了锁屏，则等解锁成功后延迟一段时间再弹出。每次启动仅提示一次。

## Current State Analysis

- 启动流程：前端 [init()](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L7879-L7940) 末尾调用 `checkScreenLock()`（**非阻塞**，仅 setTimeout 100ms 后显示锁屏遮罩）。
- 锁屏解锁：[unlockApp()](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L7976) 成功路径在 500ms 动画后 `lockScreen.style.display = 'none'`（[main.js:8012-8017](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L8012-L8017)），**解锁完成无任何对外事件**。
- 待办统计：`TodoService` 已有 `Count()` / `CountCompleted()`（[todo_service.go:72-88](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/todo_service.go#L72-L88)），但**前端无可直接调用的绑定方法**——现有 todo 绑定仅 CreateTodo / ListTodos / ToggleTodo / DeleteTodo / UpdateTodo / ClearCompletedTodos（[app.go:3390-3443](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L3390-L3443)）。
- 弹窗机制：`showConfirmDialog(msg)` 返回 Promise<boolean>（[main.js:1103-1124](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1103-L1124)），confirm 弹窗按钮文本固定为"取消"/"确定"（[index.html:1844-1846](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1844-L1846)），需支持自定义"去查看"文本。

## Proposed Changes

### 1. `internal/services/todo_service.go` — 新增未完成计数方法

在 `CountCompleted()` 后新增：

```go
// CountUnfinished 统计未完成待办数量
func (s *TodoService) CountUnfinished() (int64, error) {
	var count int64
	if err := s.db.Model(&models.Todo{}).Where("done = ?", false).Count(&count).Error; err != nil {
		s.logger.Errorw("TodoService.CountUnfinished 失败", fastlog.Error(err))
		return 0, err
	}
	return count, nil
}
```

模式与现有 `CountCompleted()` 完全一致。

### 2. `app.go` — 新增前端绑定方法

在 `ClearCompletedTodos()` 后新增（沿用现有 todo 绑定块的注释与日志风格）：

```go
// CountUnfinishedTodos 返回未完成待办数量（启动弹窗提示用）
func (a *App) CountUnfinishedTodos() (int64, error) {
	a.LogSvc.Logger.Debugw("CountUnfinishedTodos")
	return a.todoService.CountUnfinished()
}
```

### 3. `frontend/src/main.js` — 三处改动

#### 3a. 扩展 `showConfirmDialog` 支持自定义按钮文本（向后兼容）

```js
function showConfirmDialog(msg, okText = '确定', cancelText = '取消') {
    return new Promise((resolve) => {
        if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
        els.confirmDialogMsg.textContent = msg;
        // 自定义按钮文本
        if (els.confirmOkBtn) els.confirmOkBtn.textContent = okText;
        if (els.confirmCancelBtn) els.confirmCancelBtn.textContent = cancelText;
        els.confirmDialog.classList.add('visible');

        const cleanup = (result) => {
            els.confirmDialog.classList.remove('visible');
            if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
            // 恢复默认按钮文本，防止状态泄漏（符合项目工程约定）
            if (els.confirmOkBtn) els.confirmOkBtn.textContent = '确定';
            if (els.confirmCancelBtn) els.confirmCancelBtn.textContent = '取消';
            resolve(result);
        };

        els.confirmOkBtn.onclick = () => cleanup(true);
        els.confirmCancelBtn.onclick = () => cleanup(false);
        els.confirmDialog.onclick = (e) => {
            if (e.target === els.confirmDialog) cleanup(false);
        };
    });
}
```

现有调用（不带参数）行为不变。

#### 3b. 解锁成功处派发事件

[unlockApp()](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L8012-L8017) 解锁成功 `setTimeout` 内 `lockScreen.style.display = 'none'` 处追加一行：

```js
document.dispatchEvent(new CustomEvent('app-unlocked'));
```

#### 3c. init() 末尾挂载 + 新增检测函数

在 `await checkScreenLock();` 之后调用：

```js
// --- 未完成待办启动提示 ---
checkUnfinishedTodosReminder();
```

新增函数（放在 `checkScreenLock` 附近）：

```js
/**
 * 启动时提示未完成待办：锁屏启用时等解锁后延迟弹出，否则直接延迟弹出；每次启动仅一次
 */
async function checkUnfinishedTodosReminder() {
    try {
        // 读锁屏配置，判断是否启用
        const cfg = await window.go.main.App.GetAllSettings();
        const lockEnabled = cfg.screen_lock_enabled === true || cfg.screen_lock_enabled === 'true';

        const show = async () => {
            // 解锁后/无锁屏时延迟，等界面渲染稳定
            setTimeout(async () => {
                try {
                    // 查询未完成待办数量
                    const count = await window.go.main.App.CountUnfinishedTodos();
                    if (!count || count <= 0) return;
                    // 弹窗：去查看 → 跳转待办视图
                    const go = await showConfirmDialog(`你有 ${count} 个未完成的待办事项，是否现在去查看？`, '去查看', '取消');
                    if (go) switchView('todo');
                } catch (e) {
                    console.warn('未完成待办提示失败:', e);
                }
            }, lockEnabled ? 1000 : 600);
        };

        if (lockEnabled) {
            // 等解锁成功后延迟弹出；避免监听器重复，用一次性监听
            const onUnlocked = () => {
                document.removeEventListener('app-unlocked', onUnlocked);
                show();
            };
            document.addEventListener('app-unlocked', onUnlocked);
        } else {
            show();
        }
    } catch (e) {
        console.warn('未完成待办提示检查失败:', e);
    }
}
```

说明：
- 锁屏配置判断与 `checkScreenLock()` 完全一致（`screen_lock_enabled`）。
- 锁屏启用时用一次性事件监听，避免解锁事件后重复注册；延迟 1000ms（解锁动画 500ms 结束后再等 500ms）。
- 无锁屏时延迟 600ms，等主界面渲染稳定再弹，避免与入场动画叠映。
- 未完成数为 0 时静默返回，不弹窗。

## Assumptions & Decisions

- **每次启动一次**：只在 `init()` 中调用一次，无需 localStorage 标记。
- **弹窗形式**：复用现有 confirm 弹窗，不新建 modal。
- **数量获取走后端**：新增 `CountUnfinishedTodos()` 绑定，与项目"后端统计"惯例一致（对比 ListTodos 前端过滤方案：避免拉全量、语义清晰）。
- **锁屏判定**：以 `GetAllSettings()` 的 `screen_lock_enabled` 为准（与 checkScreenLock 同源），不依赖 `lockScreen.style.display`（避免与 checkScreenLock 的 100ms setTimeout 竞态）。
- 不修改 index.html —— 按钮文本通过 JS 动态设置并在 cleanup 时恢复。

## Verification

1. `go build ./...` 通过（新增绑定方法无编译错误）。
2. 前端 dev 模式 / 构建后启动应用：
   - 有未完成待办且未开锁屏 → 启动后约 0.6s 弹窗，显示正确数量；点"去查看"跳转待办视图；点"取消"关闭后停留当前页。
   - 无未完成待办 → 不弹窗。
   - 启用锁屏 → 解锁成功后约 1s 弹窗；未解锁前不弹。
   - 弹窗按钮文本为"去查看/取消"，关闭后下次弹出仍是默认"确定/取消"文本（验证 3a 状态恢复）。
