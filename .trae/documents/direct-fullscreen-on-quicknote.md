# 快速笔记直接打开全屏

## 现状问题

当启用快速笔记时，`loadQuickNoteSetting()` 依次调用：
1. `openEditor(null)` → 弹出悬浮状态卡片（overlay + `modalEnter`）
2. `toggleEditorFullscreen()` → 关闭→切换→重开全屏动画

结果：用户看到**先闪现悬浮卡片，再切换成全屏**，视觉上有一次明显的"闪烁"。

## 当前代码分析

- `openEditor(noteId, readOnly)` 在 `main.js:2472-2620`，动画阶段在 L2605-2620
- `loadQuickNoteSetting()` 在 `main.js:5059-5089`，第 5079-5080 行先后调用 `openEditor` 和 `toggleEditorFullscreen`
- `toggleEditorFullscreen()` 在 `main.js:2840-2897`，执行关闭→切换→重开动画
- 所有其他调用 `openEditor` 的地方（新建笔记、查看笔记、编辑笔记等）都是常规模式

## 修改方案

给 `openEditor()` 新增第三个参数 `startFullscreen`（默认为 `false`），当 `true` 时跳过悬浮卡片状态，直接在全屏模式下打开。

### 修改 1：`openEditor()` 签名和调用逻辑

**文件**: `frontend/src/main.js`

- 签名改为 `openEditor(noteId, readOnly = false, startFullscreen = false)`
- 在动画阶段（L2605-2620），当 `startFullscreen = true` 时：
  - overlay 正常执行 `overlayFadeIn` 动画 + 额外添加 `fullscreening` 类
  - panel 正常执行 `modalEnter` 动画 + 额外添加 `fullscreen` 类（尺寸即为全屏尺寸）
  - 设置 `state._isFullscreen = true`
  - 更新全屏按钮图标为退出全屏状态
  - 收起侧栏
  - 不再需要后续的 `toggleEditorFullscreen()` 调用

**为什么这样不会闪烁**：
- Panel 的 entry 动画 `modalEnter`（scale 0.85→1, opacity 0→1）仍然存在
- 但 panel 从一开始就是 `100vw × calc(100vh-56px)` 的尺寸
- 所以用户看到的是"面板从 85% 放大到 100% 进入全屏"，而非"小卡片出现→消失→再变成全屏"

### 修改 2：`loadQuickNoteSetting()`

**文件**: `frontend/src/main.js`

- 将第 5079-5080 行的：
  ```javascript
  openEditor(null);
  toggleEditorFullscreen();
  ```
  改为：
  ```javascript
  openEditor(null, false, true);
  ```

### 修改 3：所有其他 `openEditor()` 调用者

**文件**: `frontend/src/main.js`

所有其他调用点不需要改动，因为第三个参数默认为 `false`，不影响现有行为。

需要确认的调用点（已有 readOnly 参数的保持不动，第三个参数不传）：
- `openEditor(null)` → 正常新建
- `openEditor(id, false)` → 查看笔记编辑模式
- `openEditor(id, true)` → 查看笔记只读模式
- `openEditor(noteId, false)` → 编辑按钮
- `openEditor(noteId, true)` → 返回查看按钮
- `openEditor(lastNoteId, true)` → 恢复上次笔记

### 不修改的文件

- 所有 CSS 文件：全屏样式（`.editor-panel.fullscreen`）已经存在，不需要改动
- 后端 Go 代码：不需要改动
- HTML 文件：不需要改动

## 验证步骤

1. 启用快速笔记 → 重启应用 → 编辑器以全屏尺寸直接打开，不闪现悬浮卡片
2. 禁用快速笔记 → 重启应用 → 正常主界面，不打开编辑器
3. 正常点击新建笔记 → 仍然以悬浮卡片打开
4. 全屏切换按钮 → 正常工作
5. 关闭编辑器（Ctrl+W 或关闭按钮）→ 正常关闭
6. 从全屏状态关闭后，再新建笔记 → 以悬浮卡片打开
