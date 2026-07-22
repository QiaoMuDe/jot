# 第二步：修复技能激活时跳过基础 prompt 的问题

## 摘要

当前问题：当技能激活时，`CallAIStream` 和 `CallAIStreamRegenerate` 完全跳过基础 prompt 注入（`if len(skillIds) == 0` 条件），导致模型只收到技能 prompt 而无任何回答规范/边界约束。

修复方案：将 `baseSystemPrompt` 拆分为身份层 + 规范边界层，**始终注入规范层和边界层**，仅身份层在技能激活时跳过。

## 当前状态分析

### 常量定义（`app.go` L37-L47）

```go
var baseSystemPrompt = "你是 Jot 智能助手，一款轻量级本地笔记应用的内置 AI。" +  // 身份层
    "\n\n回答规范：" +           // ↓ 规范层 + 边界层（以下统称"规范边界层"）
    "\n1. 结构化优先：..." +
    // ...
    "\n\n约束：" +
    "\n1. 不知道的不要编造..."
```

### 注入逻辑（`app.go` L1619-L1633 和 L2013-L2027，两处结构相同）

```go
// 注入基础身份提示词（仅在无笔记引用且无技能时）
if len(skillIds) == 0 {
    hasSystem := false
    for i := range messages {
        if messages[i].Role == "system" {
            hasSystem = true
            break
        }
    }
    if !hasSystem {
        messages = append([]services.Message{
            {Role: "system", Content: baseSystemPrompt},
        }, messages...)
    }
}
```

### 注入顺序（两处相同）

8 步拼接顺序：基础 prompt → 角色扮演 → 笔记引用 → 追问引用 → 上传文件 → 搜索结果 → 卡片召回 → 技能提示词

### `appendToSystemMessage` 函数（`app.go` L2407-L2415）

查找第一条 system 消息并追加内容，若不存在则在头部新建。

## 建议变更

### 1. 拆分常量定义

将 `baseSystemPrompt` 拆分为三个部分，保持组合关系：

```
baseIdentity        = "你是 Jot 智能助手，一款轻量级本地笔记应用的内置 AI。"
baseNormsBoundaries = "\n\n回答规范：\n1. ...\n2. ...\n3. ...\n4. ...\n\n约束：\n1. ...\n2. ...\n3. ..."
baseSystemPrompt    = baseIdentity + baseNormsBoundaries   // 保持完整，供无技能时使用
```

### 2. 修改注入逻辑

**修改前（两处相同）**：
```
if len(skillIds) == 0  → 注入 baseSystemPrompt（完整三层）
```

**修改后（两处相同）**：
```
if len(skillIds) == 0  → 注入 baseSystemPrompt（完整三层）
                     ↓ 新增 else 分支
else                   → 注入 baseNormsBoundaries（仅规范+边界层）
```

### 3. 最终消息顺序

**无技能时**：
```
[Identity + Norms + Boundaries] → [context blocks] → [user messages]
```

**有技能时**：
```
[Norms + Boundaries] → [context blocks] → [skill prompts] → [user messages]
```

### 4. 涉及文件

- `app.go` — 常量定义拆分（L37-L47）+ 两处注入逻辑修改（L1619-L1633、L2013-L2027）

## 假设与决策

- **不改变 `appendToSystemMessage` 行为**：规范边界层注入后，后续 context blocks 和 skill prompts 通过 `appendToSystemMessage` 追加到同一条 system 消息中，行为不变
- **不修改 skill prompt 内容**：skill prompt 的 `# Role: xxx` 与规范边界层不冲突，前者定义角色，后者定义回答方式
- **不修改前端逻辑**：纯后端改动，前端无感知

## 验证步骤

1. 编译通过（`go build ./...`）
2. 代码审查确认：无技能时注入完整 `baseSystemPrompt`，有技能时注入 `baseNormsBoundaries`
3. 有技能时 `baseNormsBoundaries` 位于 system 消息头部，skill prompt 通过 `appendToSystemMessage` 追加在其后