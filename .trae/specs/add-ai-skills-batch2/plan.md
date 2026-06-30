# 新增 4 项四字技能 — 实施计划

## 概述

批量新增"文本润色"、"内容摘要"、"文案生成"、"工作总结" 4 项技能，全部为四字名称、无方向配置、点击即激活、互斥。完成后更多技能共计 9 项。

## 当前状态分析

技能系统现有 5 项技能：
- **翻译**（`translate`）：有方向配置
- **编程**（`coding`）：字符串 prompt
- **写作**（`writing`）：字符串 prompt
- **解题答疑**（`tutor`）：字符串 prompt
- **需求规格**（`reqspec`）：字符串 prompt

基础设施已就位，本次仅需在现有模式上追加。

## 变更项

### 1. HTML — `frontend/index.html`（"需求规格"菜单项之后）

追加 4 个菜单项，按顺序插入 `#aiChatSkillsDropdown` 末尾：

```html
<div class="ai-chat-skills-item" data-skill="polish">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
    <span>文本润色</span>
</div>
<div class="ai-chat-skills-item" data-skill="summary">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
    <span>内容摘要</span>
</div>
<div class="ai-chat-skills-item" data-skill="copywriting">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path dM18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
    <span>文案生成</span>
</div>
<div class="ai-chat-skills-item" data-skill="report">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
    <span>工作总结</span>
</div>
```

### 2. JS SKILL_PROMPTS — `frontend/src/js/ai-chat.js`（`reqspec` 之后）

追加 4 个字符串 prompt：

```js
polish: `# Role: 文本润色专家

## Core Task
对用户提供的文本进行润色优化，修正语病、优化表达、提升可读性，不改原文核心意思。

## Guidelines
- 保持原文风格和语气基调不变
- 修正语法错误、标点误用和逻辑不通顺之处
- 优化冗余表达，使句子更简洁流畅
- 对长句适当拆分，对短句适当合并，提升阅读节奏
- 专业术语和专有名词保持原样不做替换
- 只输出润色后的文本，不添加解释或评价`,
summary: `# Role: 内容摘要专家

## Core Task
提取用户提供文本的核心要点，生成结构清晰、重点突出的摘要。

## Guidelines
- 把握全文主旨，识别关键论点和支撑论据
- 摘要在原文 1/3 长度以内，用尽可能少的文字传达核心信息
- 按逻辑顺序组织摘要内容（总→分 或 时间顺序等）
- 使用原文中的关键术语和概念，保持准确性
- 保持客观中立，不添加个人评论或引申
- 如文本类型特殊（论文、新闻、故事等），适配对应的摘要风格`,
copywriting: `# Role: 创意文案专家

## Core Task
根据用户需求创作各类营销文案，包括广告语、产品描述、品牌故事、推广文案、社群文案等。

## Guidelines
- 明确文案目标和受众，确保内容精准触达
- 标题/开头要有吸引力，能激发继续阅读的欲望
- 语言简洁有力，避免空泛套话
- 突出卖点和差异化优势，转化为用户利益点
- 根据平台/媒介适配文案风格和长度
- 如有需要，主动提供多个版本供用户选择`,
report: `# Role: 工作总结专家

## Core Task
协助用户生成各类工作总结文档，包括日报、周报、月报、述职报告、项目复盘等。

## Guidelines
- 先了解工作内容的时间范围和核心职责
- 按成果导向组织内容，突出关键产出和价值
- 使用数据量化成果，避免模糊表述
- 结构化呈现：按项目/时间/优先级分类均可
- 对问题和不足客观描述，侧重改进方案而非抱怨
- 对下一步计划给出可执行的时间节点和关键目标`
```

### 3. JS 菜单点击处理 — `frontend/src/js/ai-chat.js`（`reqspec` 分支之后）

追加 4 个分支，顺序与 HTML 一致：

```js
} else if (skill === 'polish') {
    activeSkills = {};
    activeSkills.polish = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
} else if (skill === 'summary') {
    activeSkills = {};
    activeSkills.summary = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
} else if (skill === 'copywriting') {
    activeSkills = {};
    activeSkills.copywriting = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
} else if (skill === 'report') {
    activeSkills = {};
    activeSkills.report = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
}
```

### 4. JS chip 渲染 — `frontend/src/js/ai-chat.js`（`reqspec` 分支之后）

追加 4 个 chip 渲染分支：

```js
} else if (skillId === 'polish') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg></span>
        <span class="ai-chat-skill-chip-label">文本润色</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
} else if (skillId === 'summary') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></span>
        <span class="ai-chat-skill-chip-label">内容摘要</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
} else if (skillId === 'copywriting') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path dM18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></span>
        <span class="ai-chat-skill-chip-label">文案生成</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
} else if (skillId === 'report') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg></span>
        <span class="ai-chat-skill-chip-label">工作总结</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
}
```

## 验证

1. 更多技能菜单显示全部 9 项：翻译、编程、写作、解题答疑、需求规格、文本润色、内容摘要、文案生成、工作总结
2. 各项点击均正确激活，互斥清除正常
3. 各项 chip 显示正确的四字名称和对应图标
4. 发送消息时 prompt 注入正确
5. 切换会话重置
