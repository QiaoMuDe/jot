# 更多技能菜单每次打开重置滚动位置

## Summary

AI 助手的"更多技能"下拉菜单（`.ai-chat-skills-dropdown`）有 `max-height: 300px; overflow-y: auto`，打开 12 个技能项时可能出现内部滚动。滚动后再次打开菜单，滚动位置保持上一次的状态。需要每次打开时重置到顶部。

## Current State

- 打开/关闭逻辑在 [ai-chat.js#L881-L911](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L881-L911)
- 菜单 CSS 设置 `max-height: 300px; overflow-y: auto`（[ai-chat.css#L991-L1012](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/css/components/ai-chat.css#L991-L1012)）
- `skillsDropdown.scrollTop` 在关闭/打开之间不被重置

## Proposed Change

### 1. `frontend/src/js/ai-chat.js` — 添加 scrollTop 重置

在 `skillsBtn` 的 click handler 中，菜单打开的代码分支里加上一句 `skillsDropdown.scrollTop = 0`。

**当前**（约第 883-891 行）：
```javascript
skillsDropdown.classList.toggle('open');
if (skillsDropdown.classList.contains('open')) {
    if (activeSkills.translate) {
        // 同步翻译方向
    } else {
        // 清除选中态
    }
} else {
    skillsTranslateOptions.classList.remove('open');
}
```

**修改后**：
```javascript
skillsDropdown.classList.toggle('open');
if (skillsDropdown.classList.contains('open')) {
    skillsDropdown.scrollTop = 0;
    if (activeSkills.translate) {
        // 同步翻译方向
    } else {
        // 清除选中态
    }
} else {
    skillsTranslateOptions.classList.remove('open');
}
```

## 涉及的文件

| 文件 | 修改 | 说明 |
|------|------|------|
| `frontend/src/js/ai-chat.js` | +1 行 | 在菜单打开分支中重置 `skillsDropdown.scrollTop = 0` |

## Verification

1. 打开 AI 助手，点击"更多技能"按钮，菜单滚动到顶部
2. 滚到菜单中间/底部，关闭再打开，确认滚动位置回到顶部
3. 构建无报错
