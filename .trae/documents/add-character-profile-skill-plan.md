# 新增"人物档案生成"技能 - 计划

## 摘要

在 AI 助手的技能系统中增加一项"人物档案生成"技能。用户只需输入一段人物描述，AI 即可自动生成结构化的角色档案。技能名称定为 `character`，数据库 key 为 `skill_character`。

## 当前状态分析

技能系统由以下 4 层组成，新增一个"无方向/无子选项"的技能（如编程、写作等）需要修改全部 4 层：

### 1. 后端提示词存储

* `internal/database/db.go` 的 `initBuiltinPrompts()` 函数中维护一个 `[]models.AIPrompt` 切片

* 每条记录包含 `Key`/`Name`/`Category`/`Content`/`IsBuiltin` 字段

* 普通技能的 Key 命名规则：`skill_xxx`（如 `skill_coding`）

* Category 固定为 `"skill"`

### 2. 前端技能菜单 HTML

* `frontend/index.html` 第 904-956 行：`.ai-chat-skills-dropdown` 内的 `.ai-chat-skills-item` 列表

* 每个菜单项包含：`data-skill="xxx"`、一个 SVG 图标、显示名称

* 翻译技能（`translate`）有额外子选项（`.ai-chat-skills-options`）；普通技能没有

### 3. 前端 JS 交互逻辑

* `frontend/src/js/ai-chat.js` 第 841-899 行：`skillsDropdown` 的 `click` 事件处理

* 每个技能有独立的 `else if` 分支：`activeSkills = {}; activeSkills.xxx = true; renderSkillChips(); dropdown.close()`

* `frontend/src/js/ai-chat.js` 第 1499-1570+ 行：`renderSkillChips()` 函数

* 每个技能有独立的 `else if` 分支，渲染 chip 的 SVG 图标和标签文本

* 普通技能默认为 `true`，无额外配置对象（与 `translate` 不同，translate 存 `{ direction }`）

* 点击 chip 的移除按钮调用 `delete activeSkills[skill]` 并重新渲染

### 4. 前端 CSS 动画延迟

* `frontend/src/css/components/ai-chat.css` 第 695-705 行

* 当前有 11 个 `.ai-chat-skills-item:nth-child(n)` 规则，从 0.06s 到 0.46s 递增

* 新增技能需追加第 12 项的 `transition-delay: 0.50s`

## 变更内容

### 修改 1: `internal/database/db.go` — 新增内置提示词

在 `initBuiltinPrompts()` 的 `prompts` 切片末尾追加一条：

```go
{
    Key: "skill_character", Name: "人物档案", Category: "skill", IsBuiltin: true,
    Content: `# Role: 角色档案设计师

## Core Task
根据用户提供的角色描述，生成一份结构化的、详尽的人物档案。

## Output Format
严格按照以下 Markdown 结构输出，确保每个部分清晰可读：

### 基础信息
- **姓名**：
- **年龄**：
- **性别**：
- **职业**：
- **外貌特征**：
- **性格标签**：（3-5 个关键词，如"内向、敏感、固执"）

### 背景故事
- **出身**：
- **成长经历**：
- **关键事件**：
- **当前状态**：

### 性格分析
- **核心特质**：（2-3 个最深层的性格特点）
- **优点**：（列出 2-3 个）
- **弱点**：（列出 2-3 个）
- **动机**：（驱动角色行动的核心动力）
- **内心冲突**：（如果有的话）

### 人际关系
- **重要人物**：（与角色相关的人物关系描述）
- **社交状态**：

### 能力与技能
- **专长**：
- **弱点/短板**：

### 经典语录
- 1-2 句能体现角色性格的台词

## Guidelines
- 如果用户描述不够详细，可以基于合理逻辑进行适当补充，但要标注"（推测）"
- 始终以输出上述结构为准，不要偏离格式
- 用词生动形象，避免干巴巴的罗列
- 如果用户要求为特定世界观（如奇幻、科幻、古风）设计角色，适当调整用词风格以匹配世界观
- 最终只输出档案本身，不要添加解释或额外说明`,
},
```

### 修改 2: `frontend/index.html` — 新增技能菜单项

在 `.ai-chat-skills-dropdown` 末尾（`promptgen` 之后），追加：

```html
<div class="ai-chat-skills-item" data-skill="character">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
    <span>人物档案</span>
</div>
```

使用用户头像/人物图标（`<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>`），与其他技能图标风格一致。

### 修改 3: `frontend/src/js/ai-chat.js` — 新增 click 处理分支

在 `skillsDropdown` 的 `click` 事件中，`promptgen` 分支之后（第 898-899 行之间）追加：

```javascript
} else if (skill === 'character') {
    activeSkills = {};
    activeSkills.character = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
}
```

### 修改 4: `frontend/src/js/ai-chat.js` — 新增 renderSkillChips 分支

在 `renderSkillChips()` 函数中，`promptgen` 分支之后追加：

```javascript
} else if (skillId === 'character') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg></span>
        <span class="ai-chat-skill-chip-label">人物档案</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
}
```

### 修改 5: `frontend/src/css/components/ai-chat.css` — 新增动画延迟

在 `.ai-chat-skills-item:nth-child(11)` 规则之后追加：

```css
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(12) { transition-delay: 0.50s; }
```

## 无需修改的环节

* `startStreaming()` 中的 skill ID 映射逻辑：`'skill_' + id` 对 `character` 自动生成 `skill_character`，无需特殊处理

* `app.go` 后端注入逻辑：`GetSkillPrompts` 通用查表，无需修改

* Wails 绑定：无需重新生成，`CallAIStream` 签名不变

## 假设与决策

* 采用"无方向"技能模式（与编程/写作相同），不设子选项

* 提示词内容使用结构化输出格式（Markdown 分段），确保结果清晰

* 技能名称定为"人物档案"，数据 key 为 `character`，DB key 为 `skill_character`

* 图标使用 Feather 的 `user` 图标（人物轮廓），与现有技能图标风格一致

## 验证步骤

1. 确认数据库 `ai_prompts` 表中有 `skill_character` 记录，内容完整
2. 确认前端技能菜单出现"人物档案"选项，点击后显示选中态
3. 确认技能 chip 显示正确，可移除
4. 确认开启"人物档案"后发送消息，AI 回复使用结构化档案格式
5. 确认切换其他技能时，"人物档案"被正确清除

