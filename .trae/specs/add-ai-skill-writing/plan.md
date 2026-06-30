# 新增"写作"技能 — 实施计划

## 概述

在"更多技能"下拉菜单中新增"写作"技能，激活后 AI 扮演专业写作助手。该技能与"编程"技能类型相同——无方向配置，点击即激活，与翻译/编程互斥。

## 当前状态分析

现有技能系统结构（已完成 3 项改造）：
- **翻译**（`translate`）：有方向配置（`{ direction: 'to_chinese' | 'to_english' }`），需 radio 选择后激活
- **编程**（`coding`）：无方向配置，点击直接激活，`SKILL_PROMPTS` 中为字符串
- **互斥机制**：`activeSkills = {}` 清空后再赋值，确保一次只能激活一个技能
- `getSkillSystemPrompts()` 已适配字符串类型和对象类型两种 prompt 结构
- 技能状态在切换/新建会话时自动重置（`switchSession()` / `createSession()` 中 `activeSkills = {}`）

## 变更项

### 1. HTML — `frontend/index.html`（第 737-740 行后）

在"编程"菜单项之后追加"写作"菜单项：

```html
<div class="ai-chat-skills-item" data-skill="writing">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
    <span>写作</span>
</div>
```

- 使用编辑/铅笔图标，无方向选择子菜单
- 遵循 `data-skill="writing"` 命名约定

### 2. JS SKILL_PROMPTS — `frontend/src/js/ai-chat.js`（第 122 行后）

在 `coding` 之后追加 `writing` 条目：

```js
writing: `# Role: 专业写作助手

## Core Task
协助用户完成各类写作任务，包括但不限于文章、报告、邮件、文案、方案、故事创作等。

## Guidelines
- 根据用户需求明确文体和风格，确保内容贴合场景
- 结构清晰、逻辑连贯，段落过渡自然
- 用词准确、表达流畅，避免冗余啰嗦
- 如有需要，主动提供多个版本供用户选择
- 尊重用户意图，以用户的想法为基础进行润色和扩展`
```

### 3. JS 菜单点击处理 — `frontend/src/js/ai-chat.js`（第 497 行附近）

在 `coding` 分支的 `}` 之后、`return;` 之前追加 `writing` 分支：

```js
} else if (skill === 'writing') {
    // 直接激活写作技能，先清空其他技能
    activeSkills = {};
    activeSkills.writing = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
}
```

### 4. JS chip 渲染 — `frontend/src/js/ai-chat.js`（第 889 行附近）

在 `coding` 分支的 `}` 之后追加 `writing` 分支：

```js
} else if (skillId === 'writing') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></span>
        <span class="ai-chat-skill-chip-label">写作</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
}
```

## 依赖与假设

- `getSkillSystemPrompts()` 已支持字符串类型 prompt，无需改动
- 技能互斥逻辑已就位，`activeSkills = {}` 在激活时自动处理
- 会话切换重置逻辑已就位，无需改动
- 无需新增 CSS 样式，复用现有 `.ai-skill-*` 样式

## 验证

1. 点击"更多技能" → 菜单显示"翻译"、"编程"、"写作"三项
2. 点击"写作" → 技能直接激活，菜单关闭，chip 栏显示"写作" chip（编辑图标）
3. 选中"翻译"后再点"写作" → 翻译被清除，仅显示"写作" chip
4. 发送消息 → API 请求 messages 头部包含写作 system prompt
5. 点击 chip 叉号 → 写作技能取消激活
6. 切换会话 → 技能重置
