# 深度研究技能实现方案

## 一、概述

新增"深度研究"技能，突破现有Agent运行上限进行深度研究。该技能与其他技能互斥，激活时临时将最大迭代次数提升至200（若当前设置小于200）。

## 二、当前状态分析

### 技能系统架构

* 技能存储在`ai_prompts`数据库表中，Category为"skill"

* 12个内置技能：翻译、编程、写作、解题答疑、需求规格、文本润色、内容摘要、文案生成、工作总结、提示词生成、人物档案、角色扮演

* 技能通过提示词注入到Agent的Instruction中

* 前端通过"更多技能"下拉菜单选择，互斥模式（一次只能激活一个技能）

### Agent运行上限机制

* 默认最大迭代次数：20次（`internal/agent/agent.go:45`）

* 用户可在设置中调整范围：1-500次

* 通过`ai_agent_max_iterations`配置项控制

## 三、实现方案

### 1. 新增深度研究技能提示词

**文件**: `internal/database/db.go` 的 `InitBuiltinPrompts` 函数（约第560行）

在现有技能列表末尾新增：

```go
{
    Key: "skill_deep_research", Name: "深度研究", Category: "skill", IsBuiltin: true,
    Content: `# Role: 深度研究分析师

## Core Task
对用户提出的研究课题进行深度、系统的研究分析，突破常规对话限制，进行多轮工具调用和信息整合，最终提供全面、深入的研究报告。

## 研究流程规范
1. **问题分解**：将复杂研究课题分解为多个子问题
2. **多轮搜索**：对每个子问题进行多角度、多来源的信息收集
3. **信息整合**：将收集到的信息进行交叉验证和整合
4. **深度分析**：基于整合的信息进行深度分析和推理
5. **报告生成**：生成结构化、有深度的研究报告

## 工具使用策略
- 优先使用本地笔记工具（recall_notes）检索相关知识
- 本地知识不足时，使用联网搜索工具获取最新信息
- 对收集到的信息进行交叉验证，确保准确性
- 分析要有深度，不要停留在表面信息的罗列

## 输出格式
研究报告应包含：
- 研究背景与目标
- 研究方法与过程
- 核心发现与分析
- 结论与建议
- 参考来源

## Guidelines
- 报告结构清晰，逻辑严谨
- 每个论点都要有数据或事实支撑
- 对不同来源的信息进行对比分析
- 给出明确、可行的结论和建议`,
},
```

### 2. Agent运行时临时提升迭代次数

**文件**: `internal/agent/agent.go` 的 `Run` 方法（约第275行）

在读取`maxIterations`之后，添加深度研究技能的特殊处理：

```go
// 读取配置的最大迭代次数（默认 20），防止 ReAct 循环死循环
maxIterations := DefaultMaxIterations
if s.deps.Setting != nil {
    if n, err := strconv.Atoi(s.deps.Setting.Get("ai_agent_max_iterations")); err == nil && n > 0 {
        maxIterations = n
    }
}

// 深度研究技能：临时提升迭代次数至200（若当前设置小于200）
for _, skillID := range req.SkillIDs {
    if skillID == "skill_deep_research" {
        if maxIterations < 200 {
            maxIterations = 200
        }
        break
    }
}
```

### 3. 前端新增深度研究技能选项

**文件**: `frontend/index.html` 的技能菜单（约第1258行）

在现有技能菜单末尾新增：

```html
<div class="ai-chat-skills-item" data-skill="deep_research">
    <span class="ai-chat-skills-icon">🔍</span>
    <div class="ai-chat-skills-info">
        <div class="ai-chat-skills-name">深度研究</div>
        <div class="ai-chat-skills-desc">突破迭代上限，进行深度分析研究</div>
    </div>
</div>
```

### 4. 前端技能标签映射

**文件**: `frontend/src/js/ai-chat.js` 的 `getSkillLabel` 函数（约第2108行）

在switch语句中新增：

```javascript
case 'deep_research': return '深度研究';
```

## 四、修改文件清单

| 文件                           | 修改内容                         |
| ---------------------------- | ---------------------------- |
| `internal/database/db.go`    | 新增`skill_deep_research`技能提示词 |
| `internal/agent/agent.go`    | 深度研究技能临时提升迭代次数至200           |
| `frontend/index.html`        | 新增深度研究技能菜单项                  |
| `frontend/src/js/ai-chat.js` | 新增技能标签映射                     |

## 五、验证步骤

1. 启动应用，检查数据库中是否成功插入`skill_deep_research`技能
2. 在前端AI助手中点击"更多技能"，确认深度研究选项可见
3. 激活深度研究技能，发送研究类问题，验证：

   * 迭代次数是否提升至200（即使设置中配置小于200）

   * 研究报告是否按规范格式输出
4. 验证深度研究技能与其他技能互斥（激活深度研究后，其他技能应被取消）
5. 在设置中将Agent运行上限调整为大于200的值（如300），激活深度研究技能，验证迭代次数仍为300（不覆盖更大的值）

