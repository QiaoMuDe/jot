# 重命名"更多技能"菜单项（编程→编程开发，写作→创意写作）

## 修改内容

根据用户选择的方案：
- **翻译** → 保持不变
- **编程** → **编程开发**
- **写作** → **创意写作**

## 需要修改的文件及位置

### 1. `frontend/index.html`

菜单显示文本，第 890 行和第 894 行：

```html
<!-- 第 888-891 行 -->
<div class="ai-chat-skills-item" data-skill="coding">
    <svg>...</svg>
    <span>编程</span>              ← 改为 <span>编程开发</span>
</div>

<!-- 第 892-895 行 -->
<div class="ai-chat-skills-item" data-skill="writing">
    <svg>...</svg>
    <span>写作</span>              ← 改为 <span>创意写作</span>
</div>
```

### 2. `frontend/src/js/ai-chat.js`

芯片（chip）标签文本，`renderSkillChips()` 函数中，第 1533 行和第 1539 行：

```js
// 第 1530-1535 行 (coding)
<span class="ai-chat-skill-chip-label">编程</span>       ← 改为 <span>编程开发</span>

// 第 1536-1541 行 (writing)
<span class="ai-chat-skill-chip-label">写作</span>       ← 改为 <span>创意写作</span>
```

## 影响范围

- 仅修改 2 个文件中的 4 处纯文本
- 不涉及 CSS、JS 逻辑、HTML 结构
- CSS 样式无需调整（文本长度变化不大，原有间距足够）
