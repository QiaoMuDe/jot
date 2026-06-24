# Bug修复：纯文本模式全屏后输入框变半高且无法还原

## 问题现象

笔记在纯文本模式下点击全屏 → 内容输入区（CM6 编辑器）变成半高。点击还原和再次全屏，一直是半高，无法恢复正常。

## 根因分析

`toggleEditorFullscreen()` 的阶段 4（L2899-L2911）中，内容淡入后**清除了 `body.style.opacity`**：

```javascript
if (body) {
    body.style.transition = 'opacity 0.12s ease-out';
    body.style.opacity = '1';
    setTimeout(() => {
        body.style.opacity = '';       // ← 清除 inline opacity
        body.style.transition = '';
    }, 130);
}
```

清除后，`.editor-body` 回退到 CSS 的 `opacity: 0`（[style.css#L1052](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css#L1052)）。

所以全屏切换完成后：
- **Panel**: `modalEnter` 的 `forwards` 保持 `opacity: 1; transform: scale(1)` → 可见
- **Body**: CSS `opacity: 0` → **不可见**，但仍在 flex 布局中占位
- 用户看到 editor panel 有一半区域空白（body 透明不可见，但 footer/header 可见）

### 涉及的关键代码

| 文件 | 行 | 内容 |
|------|----|------|
| `style.css` | L1046-1053 | `.editor-body { opacity: 0; ... }` |
| `style.css` | L942-955 | `.editor-panel` 无 CSS transition |
| `main.js` | L2868-2870 | 阶段1：body opacity 淡出到 0 |
| `main.js` | L2875-2876 | 阶段2：清除 body transition，面板做浅退出 |
| `main.js` | L2880-2897 | 阶段3：切换 fullscreen class，重排，modalEnter |
| `main.js` | L2903-2908 | **阶段4：内容恢复淡入后清除 opacity，触发 bug** |

### Markdown 模式不受影响的原因

Markdown 模式下 `.editor-textarea` 在预览模式为 `display: none`，`.md-rendered` 为 `block`。阶段 2 中 `mdRendered.style.display = 'none'`，阶段 4 恢复。但 `display` 切换使浏览器重新计算布局，body 的 `opacity: 0` 对 MD 内容的影响被 `display: none/block` 的强制重排掩盖。纯文本模式没有 mdRendered 的 display 切换，body opacity 的问题直接暴露。

## 修改方案

### 改动 1：CSS — 删除 `.editor-body` 的 `opacity: 0`

**文件**: `frontend/src/style.css` L1052

```
- opacity: 0;
+ /* opacity 由 openEditor 的 viewEnter 动画和 toggleEditorFullscreen 控制 */
```

原因：
- `openEditor()` 通过 `body.style.animation = 'viewEnter ... forwards'` 从 opacity:0→1 淡入，CSS 值被动画覆盖
- `closeEditor()` 清除 `body.style.animation` 后，body 应该保持 `opacity: 1` 以便退出动画期间可见
- `toggleEditorFullscreen()` 通过 inline `body.style.opacity` 控制淡入淡出，不应该依赖 CSS 回退值

### 改动 2：JS — 不清理 `body.style.opacity`

**文件**: `frontend/src/main.js` L2905-L2908

修改阶段 4，去掉清除 opacity 的逻辑：

```javascript
if (body) {
    body.style.transition = 'opacity 0.12s ease-out';
    body.style.opacity = '1';
    setTimeout(() => {
        // 只清理 transition，保留 opacity = 1 防止回退到 CSS opacity: 0
        body.style.transition = '';
    }, 130);
}
```

### 验证步骤

1. 纯文本模式 → 全屏 → 内容区正常显示，不等半高
2. 纯文本模式 ← 退出全屏 → 恢复正常悬浮卡片尺寸
3. 纯文本模式 ↔ 重复全屏/退出 5 次 → 始终正常
4. Markdown 模式 → 全屏/退出 → 正常
5. 快速笔记启动 → 全屏 → 正常
6. 新建笔记 → 编辑 → 全屏 → 正常
7. 关闭编辑器 → 再打开其他笔记 → 正常
