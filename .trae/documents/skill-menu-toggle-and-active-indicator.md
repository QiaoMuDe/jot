# 技能菜单选中指示 + 点击切换

## Summary

给「更多技能」下拉菜单增加两项交互优化：
1. **选中指示** — 菜单中已激活的技能项显示 ✓ 和 accent 色高亮
2. **点击切换** — 点击已激活的技能项可以取消激活（toggle 行为）

## Current State Analysis

- **菜单结构**: `index.html` 中 `#aiChatSkillsDropdown` 内有 12 个 `.ai-chat-skills-item[data-skill="xxx"]`，每个含 icon SVG + span 文本
- **当前点击处理**: `ai-chat.js` 第 888-974 行，12 个独立的 `else if` 分支，每个都执行 `activeSkills = {}` + 设置新技能
- **无菜单状态更新**: 打开菜单时没有任何代码遍历 activeSkills 并标记对应菜单项
- **无 toggle**: 点击已激活的技能仍然执行清空 + 重新设置（相当于无操作）

## Proposed Changes

### File 1: `frontend/src/css/components/ai-chat.css`

**Add** active state style for skill menu items (after existing `.ai-chat-skills-item:hover` rule, ~line 1077):

```css
.ai-chat-skills-item.active {
    color: var(--accent);
}
.ai-chat-skills-item.active::after {
    content: '✓';
    margin-left: auto;
    font-weight: 600;
    color: var(--accent);
    font-size: 0.72rem;
}
.ai-chat-skills-item.active svg {
    color: var(--accent);
}
```

Why: 最简单的实现方式 — `::after` 伪元素打勾，不需要修改 HTML 结构或 JS 动态插入元素。`active` class 切换即可。

### File 2: `frontend/src/js/ai-chat.js`

#### Change 1: 新增 `updateSkillsMenuActiveState()` 函数（在 `renderSkillChips()` 附近）

```javascript
/**
 * 更新更多技能菜单中的选中状态
 */
function updateSkillsMenuActiveState() {
    if (!skillsDropdown) return;
    const items = skillsDropdown.querySelectorAll('.ai-chat-skills-item');
    items.forEach(item => {
        const skill = item.dataset.skill;
        item.classList.toggle('active', !!activeSkills[skill]);
    });
}
```

#### Change 2: 菜单打开时更新状态

在 `skillsBtn` click handler 中（第 882 行）：打开菜单前先更新选中状态

```javascript
skillsBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    updateSkillsMenuActiveState();  // ← 新增：打开前刷新选中状态
    skillsDropdown.classList.toggle('open');
    if (skillsDropdown.classList.contains('open')) {
        skillsDropdown.scrollTop = 0;
    }
});
```

#### Change 3: 技能切换时更新菜单选中状态

`renderSkillChips()` 中已经会重绘 chip，在它之后调用 `updateSkillsMenuActiveState()` 保持同步。但更合理的是在 `saveCurrentSessionConfig()` 之前/之后调用。

由于 `updateSkillsMenuActiveState()` 是轻量 DOM 操作，可以在每个技能激活/取消后调用。

#### Change 4: 重构技能点击事件为 toggle 模式

将 12 个独立 `else if` 分支（第 893-971 行）重构为一段通用代码：

```javascript
// 点击技能菜单项
skillsDropdown.addEventListener('click', (e) => {
    const item = e.target.closest('.ai-chat-skills-item');
    if (item) {
        const skill = item.dataset.skill;
        
        // toggle：如果已激活则取消，否则激活（互斥）
        if (activeSkills[skill]) {
            delete activeSkills[skill];
        } else {
            activeSkills = {};
            if (skill === 'translate') {
                activeSkills.translate = { source: 'english', target: 'chinese' };
            } else {
                activeSkills[skill] = true;
            }
        }
        
        renderSkillChips();
        updateSkillsMenuActiveState();
        saveCurrentSessionConfig();
        
        if (Object.keys(activeSkills).length === 0) {
            // 没有激活技能时，不关闭菜单（让用户看到取消效果）
            // 但如果有技能激活，则关闭菜单
        } else {
            skillsDropdown.classList.remove('open');
        }
        return;
    }
});
```

注意：设计决策 — 当取消激活最后一个技能时，菜单保持打开（让用户看到取消后的效果）。激活技能时菜单关闭（和现有行为一致）。

## Assumptions & Decisions

1. **互斥原则保持不变** — 技能之间仍是互斥的（激活 A 后点 B → 取消 A 激活 B）
2. **toggle 行为** — 点击已激活的技能 → 取消激活，chip 消失，菜单保持打开
3. **空状态** — 所有技能都取消后，skill bar 自动隐藏（现有 `renderSkillChips()` 机制已支持）
4. **翻译技能** — 点击翻译时直接使用默认 english→chinese，方向通过 chip 上的语言标签切换
5. **菜单选中状态更新时机** — 菜单打开时 + 技能切换后

## Verification

1. 打开菜单 → 所有技能项无选中标记
2. 点击「编程开发」→ 菜单关闭，chip 显示，再次打开菜单 → 「编程开发」显示 ✓
3. 再次点击「编程开发」→ 技能取消，chip 消失，菜单保持打开
4. 激活翻译 → 菜单关闭，chip 显示语言方向，再次打开菜单 → 「翻译」显示 ✓
5. 在菜单打开的状态下激活一个技能，再点另一个 → 第一个取消，第二个激活，菜单关闭
