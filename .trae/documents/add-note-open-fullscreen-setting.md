# 添加"笔记默认全屏打开"设置

## 概述

在设置页增加一个开关，控制笔记在"查看"、"编辑"、"新建"时是否默认以全屏模式打开。默认为普通模式（关闭）。

## 当前状态分析

### 笔记打开入口

所有笔记打开最终都通过 `openEditor(noteId, readOnly, startFullscreen)` 函数（`main.js` 第 2188 行）：

| 操作 | 入口函数 | 调用方式 | 行号 |
|------|----------|----------|------|
| 卡片点击查看 | `window.viewNote(id)` | `openEditor(id, true)` | 2886 |
| 右键→查看 | `window.viewNote(id)` | 同上 | 2927 |
| 右键→编辑 | `window.openNote(id)` | `openEditor(id, false)` | 2930 |
| FAB 新建笔记 | 直接调用 | `openEditor(null)` | 3482 |
| Ctrl+N 新建 | 直接调用 | `openEditor(null)` | 3868 |
| 搜索弹窗打开 | `_openNoteFromSearch` | `openEditor(noteId, true)` | 5206 |
| AI 保存通知 | 直接调用 | `openEditor(note.id, true)` | — |

### 全屏机制

`openEditor` 的第三个参数 `startFullscreen` 控制是否立即进入全屏。全屏模式通过给 `.editor-panel` 添加 `.fullscreen` CSS 类实现（CSS 第 66-77 行）。

### 设置系统

已有的设置系统使用 `SetSetting(key, value)` / `GetSetting(key)` 后端绑定，配合 `localStorage` 降级。布尔值存储为字符串 `'true'` / `'false'`。设置页的开关使用 `toggle-switch` 或 `.ai-chat-toggle-switch` 组件。

## 变更方案

### 涉及文件

| 文件 | 变更类型 |
|------|----------|
| `frontend/index.html` | 新增 toggle 开关 UI |
| `frontend/src/main.js` | 新增加载/保存逻辑 + 修改打开入口 |

**无需后端变更**：复用现有的 `SetSetting`/`GetSetting` 通用绑定，设置 key 为 `note_open_fullscreen`。

### 具体改动

#### 1. `frontend/index.html` — 设置页新增开关

在"编辑器选项"设置块（`.settings-section`，约第 336 行）中的"启用 CM6 语法高亮"和"代码高亮主题"之间，新增一行：

```html
<!-- 全屏打开笔记 -->
<div class="font-setting-row">
    <label class="font-setting-label" style="min-width:72px;">全屏打开</label>
    <div class="toggle-switch">
        <input type="checkbox" id="noteOpenFullscreenToggle" />
        <label class="toggle-track" for="noteOpenFullscreenToggle">
            <span class="toggle-thumb"></span>
        </label>
    </div>
</div>
```

使用已有的 `.toggle-switch` 样式，与快速笔记开关、语法高亮开关风格一致。

#### 2. `frontend/src/main.js` — 加载/保存逻辑

**a. `els` 对象新增 DOM 引用**（约第 230 行）：

```javascript
noteOpenFullscreenToggle: $('noteOpenFullscreenToggle'),
```

**b. 新增 `loadNoteOpenFullscreenSetting()` 函数**（约在 `loadSyntaxHighlightSetting` 附近）：

```javascript
// 加载笔记全屏打开设置
window.loadNoteOpenFullscreenSetting = async function() {
    try {
        let enabled = false;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('note_open_fullscreen');
            enabled = val === 'true';
        } else {
            enabled = localStorage.getItem('note_open_fullscreen') === 'true';
        }
        els.noteOpenFullscreenToggle.checked = enabled;
    } catch (_) {}
};
```

**c. `init()` 中调用加载函数**（约第 5520 行 `loadQuickNoteSetting()` 附近）：

```javascript
await loadNoteOpenFullscreenSetting();
```

**d. 绑定 change 事件**（在 `reloadSettings` 后的初始化区域，约第 3664 行 `quickNoteToggle` 附近）：

```javascript
// 全屏打开笔记开关
els.noteOpenFullscreenToggle.addEventListener('change', async (e) => {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
            await window.go.main.App.SetSetting('note_open_fullscreen', String(e.target.checked));
        } else {
            localStorage.setItem('note_open_fullscreen', String(e.target.checked));
        }
        nm.show('设置已保存', 'success');
    } catch (err) {
        console.error('保存全屏打开设置失败:', err);
    }
});
```

#### 3. `frontend/src/main.js` — 修改笔记打开入口

**a. 新增辅助函数**（在 `loadNoteOpenFullscreenSetting` 附近）：

```javascript
// 检查是否应以全屏模式打开笔记
function getNoteOpenFullscreen() {
    return els.noteOpenFullscreenToggle?.checked || false;
}
```

**b. 修改 `window.viewNote`**（第 2886 行）：

```javascript
// 原代码
window.viewNote = function(id) {
    openEditor(id, true);
};

// 改为
window.viewNote = function(id) {
    openEditor(id, true, getNoteOpenFullscreen());
};
```

**c. 修改 `window.openNote`**（第 2879 行）：

```javascript
// 原代码
window.openNote = function(id) {
    openEditor(id, false);
};

// 改为
window.openNote = function(id) {
    openEditor(id, false, getNoteOpenFullscreen());
};
```

**d. 修改 FAB 新建笔记**（第 3481-3483 行）：

```javascript
// 原代码
els.fabNewNote.addEventListener('click', () => {
    openEditor(null);
});

// 改为
els.fabNewNote.addEventListener('click', () => {
    openEditor(null, false, getNoteOpenFullscreen());
});
```

**e. 修改 Ctrl+N 快捷键**（第 3865-3868 行）：

```javascript
// 原代码
if (e.ctrlKey && e.key === 'n') {
    e.preventDefault();
    if (!els.viewEditor.classList.contains('active')) {
        openEditor(null);
    }
}

// 改为
if (e.ctrlKey && e.key === 'n') {
    e.preventDefault();
    if (!els.viewEditor.classList.contains('active')) {
        openEditor(null, false, getNoteOpenFullscreen());
    }
}
```

**f. 修改 `_openNoteFromSearch`**（第 5196-5210 行）：调用 `openEditor(noteId, true, getNoteOpenFullscreen())` 传入全屏参数，与卡片查看行为一致。

### 不需要修改的地方

- **后端**：无需新增绑定，复用 `GetSetting`/`SetSetting`
- **数据库**：无需迁移，`note_open_fullscreen` 键会在首次保存时自动创建
- **`reloadSettings()`**（data-management.js）：`window.loadNoteOpenFullscreenSetting` 已暴露到全局，会自动被调用

## 假设与决策

1. **只控制编辑器面板全屏**：不影响 OS 窗口全屏（F11）。这是用户自然的期望，因为现有的全屏按钮/Ctrl+E 就是编辑器面板全屏。
2. **实时生效**：修改设置后，下一次打开笔记立即生效，无需重启。
3. **现有全屏按钮不受影响**：用户仍可手动通过按钮或 Ctrl+E 切换全屏状态。
4. **快速笔记模式不受影响**：快速笔记的 `openEditor(null, false, true)` 已明确传递 `true`，不会被修改。

## 验证步骤

1. 设置页出现"全屏打开"开关，默认关闭
2. 关闭状态下，点击查看笔记 → 普通模式打开
3. 开启后，点击查看笔记 → 全屏模式打开
4. 开启后，右键编辑 → 全屏模式打开  
5. 开启后，FAB 新建/Ctrl+N → 全屏模式打开
6. 开启后，搜索弹窗打开笔记 → 全屏模式打开
7. 修改开关值，刷新设置页，确认值持久化
