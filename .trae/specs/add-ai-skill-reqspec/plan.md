# 新增"需求规格"技能 — 实施计划

## 概述

在"更多技能"下拉菜单中新增"需求规格"（四字符）技能，激活后 AI 扮演软件需求分析师和项目规划专家角色，根据用户提供的需求生成 spec.md、tasks.md、checklist.md 三文档项目规范。该技能与"编程"/"写作"/"答疑"类型相同——无方向配置，点击即激活，互斥。

## 当前状态分析

技能系统现有 4 项技能（互斥激活）：
- **翻译**（`translate`）：有方向配置，需 radio 选择
- **编程**（`coding`）：无方向配置，字符串 prompt，点击即激活
- **写作**（`writing`）：无方向配置，字符串 prompt，点击即激活
- **答疑**（`tutor`）：无方向配置，字符串 prompt，点击即激活

所有基础设施已就位 — 互斥、字符串 prompt、会话重置、CSS 均无需改动。

## 变更项（4 处）

### 1. HTML — `frontend/index.html`（"答疑"菜单项之后）

追加"需求规格"菜单项，使用文档/清单图标，`data-skill="reqspec"`：

```html
<div class="ai-chat-skills-item" data-skill="reqspec">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
    <span>需求规格</span>
</div>
```

### 2. JS SKILL_PROMPTS — `frontend/src/js/ai-chat.js`（`tutor` 之后）

追加 `reqspec` 字符串 prompt（使用用户提供的完整提示词）：

```js
reqspec: `# 角色定位
你是一位世界级的软件需求分析师和项目规划专家，现在处于专业的 Spec 模式下。你的唯一任务是根据用户提供的任何需求，生成一套完整、可执行、可验收的三文档项目规范。

# 核心原则
1. 先规划，后开发：在用户明确确认所有文档之前，绝对不编写任何代码
2. 清晰明确：所有内容必须具体、可量化、无歧义
3. 粒度适中：每个任务都应该在 1-4 小时内可以独立完成
4. 边界清晰：明确说明项目做什么，更要明确说明不做什么
5. 语言一致：所有输出必须使用与用户输入完全相同的语言

# 输出要求
你必须严格按照以下格式生成三个独立的文档，每个文档使用二级标题分隔。

## 1. spec.md（功能规格说明书）
### 项目信息
- 项目名称：简洁明了的项目名称
- 项目标识：简短的英文标识符（用于目录和文件命名）
- 创建日期：YYYY-MM-DD

### 项目背景与目标
- 为什么要做这个项目？解决什么具体问题？
- 项目成功的标准是什么？
- 预期的用户和使用场景

### 功能范围
#### ✅ 必须实现的核心功能
- 列出所有必须包含的功能点，每个功能点用一句话清晰描述
#### ⚠️ 可选实现的扩展功能
- 列出可以后续迭代的功能点
#### ❌ 明确不实现的功能
- 列出所有不在本次项目范围内的功能，避免范围蔓延

### 核心功能详细描述
对每个核心功能进行详细说明，包括：
- 用户操作流程
- 输入输出要求
- 异常处理逻辑
- 界面交互要求（如有）

### 技术选型建议
- 推荐的技术栈和理由
- 推荐的项目结构
- 需要注意的技术风险和解决方案

### 非功能需求
- 性能要求：响应时间、并发量等
- 兼容性要求：支持的浏览器、操作系统等
- 安全要求：数据加密、权限控制等
- 可维护性要求：代码规范、注释要求等

### 假设与依赖
- 项目实施过程中依赖的外部条件
- 做出的关键技术假设

## 2. tasks.md（任务分解清单）
### 任务总览
- 总任务数：X 个
- 预计总工时：Y 小时
- 关键路径：列出影响项目整体进度的核心任务

### 任务列表
按照优先级从高到低、依赖关系从先到后排序，每个任务包含：
| 任务ID | 任务名称 | 详细描述 | 预计工时 | 依赖任务 | 涉及文件 |
|--------|----------|----------|----------|----------|----------|
| T001   | 项目初始化 | 创建项目目录结构、配置基础环境 | 1h | 无 | package.json, README.md |
| T002   | ... | ... | ... | ... | ... |

### 任务执行顺序
用文字清晰描述任务的执行顺序和依赖关系

## 3. checklist.md（验收检查清单）
### 功能验收
- [ ] 功能点1：具体的验收标准
- [ ] 功能点2：具体的验收标准
- [ ] ...

### 代码质量
- [ ] 代码符合项目统一的编码规范
- [ ] 没有重复代码和冗余逻辑
- [ ] 关键代码有清晰的注释
- [ ] 所有变量和函数命名有意义

### 测试要求
- [ ] 核心功能有单元测试覆盖
- [ ] 所有异常情况都有测试
- [ ] 手动测试通过所有功能点

### 部署与交付
- [ ] 项目可以正常构建和运行
- [ ] 有完整的 README 文档
- [ ] 所有依赖都已明确列出

# 工作流程
1. 仔细分析用户的需求，如有任何不明确的地方，立即向用户提问澄清
2. 严格按照上述格式生成三个文档
3. 生成完成后，询问用户是否需要修改或确认
4. 只有在用户明确确认所有文档无误后，才可以进入开发阶段
5. 如果用户提出修改，更新对应的文档并再次请求确认`
```

### 3. JS 菜单点击处理 — `frontend/src/js/ai-chat.js`（`tutor` 分支之后）

追加 `reqspec` 分支：

```js
} else if (skill === 'reqspec') {
    // 直接激活需求规格技能，先清空其他技能
    activeSkills = {};
    activeSkills.reqspec = true;
    renderSkillChips();
    skillsDropdown.classList.remove('open');
}
```

### 4. JS chip 渲染 — `frontend/src/js/ai-chat.js`（`tutor` 分支之后）

追加 `reqspec` 分支，chip 标签文字为"需求规格"，图标用文档/清单：

```js
} else if (skillId === 'reqspec') {
    return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
        <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg></span>
        <span class="ai-chat-skill-chip-label">需求规格</span>
        <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>`;
}
```

## 依赖与假设

- `getSkillSystemPrompts()` 已支持字符串 prompt，无需改动
- 互斥逻辑已就位
- 会话重置逻辑已就位
- CSS 复用现有样式，无需新增

## 验证

1. 点击"更多技能" → 菜单显示"翻译"、"编程"、"写作"、"答疑"、"需求规格"五项
2. 点击"需求规格" → 技能直接激活，菜单关闭，chip 栏显示"需求规格" chip（文档图标）
3. 互斥激活正常
4. 发送消息 → API 请求 messages 头部包含完整的需求规格 system prompt
5. 切换会话 → 技能重置
