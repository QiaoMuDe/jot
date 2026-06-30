# 新增"答疑"技能 — 实施计划

## 概述

在"更多技能"下拉菜单中新增"答疑"（二字符）技能，激活后 AI 扮演解题导师角色，提供学科问题解答、思路讲解、知识点梳理等服务。该技能与"编程"、"写作"类型相同——无方向配置，点击即激活，与其他技能互斥。

## 当前状态分析

技能系统现有 3 项技能（互斥激活）：
- **翻译**（`translate`）：有方向配置，需 radio 选择
- **编程**（`coding`）：无方向配置，字符串 prompt，点击即激活
- **写作**（`writing`）：无方向配置，字符串 prompt，点击即激活

所有基础设施已就位：
- 互斥机制：`activeSkills = {}` 清空后再赋值
- 字符串 prompt 支持：`getSkillSystemPrompts()` 已适配 `typeof promptDef === 'string'`
- 会话重置：`switchSession()` / `createSession()` 中 `activeSkills = {}`
- 无需新增 CSS，复用现有样式

## 变更项（4 处，均与"写作"技能模式一致）

### 1. HTML — `frontend/index.html`（"写作"菜单项之后）

追加"答疑"菜单项，使用问号灯泡图标，`data-skill="tutor"`：

```html
<div class="ai-chat-skills-item" data-skill="tutor">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <span>答疑</span>
</div>
```

### 2. JS SKILL_PROMPTS — `frontend/src/js/ai-chat.js`（`writing` 之后）

追加 `tutor` 字符串 prompt：

```js
tutor: `# Role: 解题导师

## Core Task
帮助用户解答各学科领域的题目和疑问，提供清晰的解题思路、步骤推导和知识点讲解。

## Guidelines
- 先理解题目要求，确认问题类型和已知条件
- 分步骤展示解题过程，逻辑清晰、推导严谨
- 不仅给出答案，更要解释"为什么"和"怎么想到的"
- 对关键知识点进行延伸讲解，帮助用户举一反三
- 对于有多种解法的题目，优先介绍最简洁或最通用的方法
- 如遇用户理解困难，主动换用更通俗的方式重新解释`
```

### 3. JS 菜单点击处理 — `frontend/src/js/ai-chat.js`（`writing` 分支之后）

追加 `tutor` 分支：

```js
} else if (skill === 'tutor') {
    // 直接激活答疑技能，先清空其他技能
    activeSkills = {};
    activeSkills.tutor = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
}
```

### 4. JS chip 渲染 — `frontend/src/js/ai-chat.js`（`writing` 分支之后）

追加 `tutor` 分支，chip 标签文字为"答疑"：

```js
} else if (skillId === 'tutor') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
        <span class="ai-chat-skill-chip-label">答疑</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
}
```

## 依赖与假设

- `getSkillSystemPrompts()` 已支持字符串 prompt，无需改动
- 互斥逻辑已就位，`activeSkills = {}` 自动处理
- 会话重置逻辑已就位，无需改动
- CSS 复用现有样式，无需新增

## 验证

1. 点击"更多技能" → 菜单显示"翻译"、"编程"、"写作"、"答疑"四项
2. 点击"答疑" → 技能直接激活，菜单关闭，chip 栏显示"答疑" chip（问号灯泡图标）
3. 选中其他技能后再点"答疑" → 旧技能被清除，仅显示"答疑" chip
4. 发送消息 → API 请求 messages 头部包含答疑 system prompt
5. 点击 chip 叉号 → 答疑技能取消激活
6. 切换会话 → 技能重置
