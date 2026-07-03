# 三项调整计划

## 总览

对 jot 项目进行 3 项独立调整：修改引用笔记默认截断字数为 5000、AI 输入框支持 Ctrl+Enter 换行、更换首页 AI 浮动按钮图标。

---

## 当前状态分析

### 1. 引用截断默认值

默认值 1000 散布在 4 个位置：

| 文件 | 行号 | 当前值 |
|------|------|--------|
| `frontend/index.html` | 525 | `<input value="1000" ...>` |
| `app.go` | 582-585 | 注释+默认返回 `1000` |
| `frontend/src/main.js` | 2255 | 校验失败时重置为 `1000` |
| `internal/services/note_service.go` | 134 | `maxPerNote := 1000` |

### 2. AI 输入框键盘事件

`frontend/src/js/ai-chat.js` 第 1681-1686 行 `onInputKeydown`：
- `Enter` → 发送消息
- `Shift+Enter` → 换行（textarea 默认行为，未拦截）
- `Ctrl+Enter` → 无特殊处理

### 3. 首页 AI 图标

`frontend/index.html` 第 1456 行：`#fabAI` 按钮使用笑脸 SVG（圆形 + 眼睛 + 微笑嘴巴），显得有些幼稚。

---

## 具体变更

### 变更 1：引用截断默认值 1000 → 5000

涉及 4 个文件，均为简单数值替换：

**文件 1：`frontend/index.html` 第 525 行**
- 修改：`value="1000"` → `value="5000"`

**文件 2：`app.go` 第 581-585 行**
- 修改：函数注释和返回值中的 `1000` → `5000`

**文件 3：`frontend/src/main.js` 第 2255 行**
- 修改：重置兜底值 `1000` → `5000`

**文件 4：`internal/services/note_service.go` 第 134 行**
- 修改：`maxPerNote := 1000` → `maxPerNote := 5000`
- 同步更新上方注释 `默认 1000` → `默认 5000`

### 变更 2：AI 输入框支持 Ctrl+Enter 换行

**文件：`frontend/src/js/ai-chat.js` 第 1681-1686 行**

修改 `onInputKeydown` 函数，使 `Ctrl+Enter` 在光标位置插入换行：

```javascript
function onInputKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey && !e.ctrlKey) {
        e.preventDefault();
        onSend();
    }
    if (e.key === 'Enter' && e.ctrlKey) {
        e.preventDefault();
        // 在光标位置插入换行
        const start = inputEl.selectionStart;
        const end = inputEl.selectionEnd;
        inputEl.value = inputEl.value.substring(0, start) + '\n' + inputEl.value.substring(end);
        inputEl.selectionStart = inputEl.selectionEnd = start + 1;
        // 触发 input 事件使发送按钮状态更新
        inputEl.dispatchEvent(new Event('input', { bubbles: true }));
    }
}
```

效果：
- `Enter` → 发送（不变）
- `Shift+Enter` → 换行（不变）
- `Ctrl+Enter` → 换行（新增）

### 变更 3：更换首页 AI 图标

**文件：`frontend/index.html` 第 1456 行**

将笑脸 SVG 替换为 sparkle/星星 AI 图标（Lucide 风格，与项目现有图标风格一致）：

新 SVG：
```html
<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5z"/>
    <path d="M18.5 14.5L16 16l2.5 1.5L20 20l1.5-2.5L24 16l-2.5-1.5z"/>
    <path d="M6 14.5L4 16l2.5 1.5L8 20l1.5-2.5L12 16l-2.5-1.5z"/>
</svg>
```

该图标是一个星星/火花图案，常用于 AI 功能标识，且保持 24x24 viewBox、1.5px stroke、currentColor。

---

## 验证步骤

1. **引用截断**：打开 AI 设置面板，确认`引用截断`输入框默认显示 5000
2. **Ctrl+Enter 换行**：在 AI 输入框中按下 Ctrl+Enter，确认插入换行而非发送消息
3. **AI 图标**：确认首页浮动按钮显示为星星/火花图案而非笑脸
