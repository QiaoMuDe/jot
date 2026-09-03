# 在用户消息下方添加发送时间戳

## 需求概述

在AI助手模块的用户消息气泡下方，除了现有的 token 计数，额外显示消息发送时间戳。使用智能简化显示规则，保持单行紧凑布局。

## 当前状态分析

| 层                                | 当前情况                                                                | 是否需要修改 |
| -------------------------------- | ------------------------------------------------------------------- | :----: |
| 数据库模型 `AIMessage`                | 已有 `CreatedAt time.Time` 字段，数据已存在                                   |  ❌ 不需要 |
| 服务层 `Message` 结构体                | 缺少 `CreatedAt` 字段                                                   |  ✅ 需要  |
| `LoadAISessionMessagesPaginated` | 转换时未复制 `CreatedAt`                                                  |  ✅ 需要  |
| `SaveAIMessage`                  | 返回值 `SaveAIMessageResult` 缺少 `CreatedAt`                            |  ✅ 需要  |
| 前端加载历史消息                         | `chatHistory` map 缺少 `created_at`                                   |  ✅ 需要  |
| 前端发送新消息                          | `sendUserText` `handleRegenerate` 创建 chatHistory 条目时缺少 `created_at` |  ✅ 需要  |
| 前端渲染                             | `createMsgActions` 只显示 token，不显示时间                                  |  ✅ 需要  |
| 前端更新 token                       | 流式结束后更新用户消息 token 显示，不涉及时间                                          |  ❌ 不需要 |

## 实现方案

### 智能时间显示规则

| 时间范围    | 显示格式               | 示例                              |
| ------- | ------------------ | ------------------------------- |
| **今天**  | `HH:MM`            | `128 tokens · 14:35`            |
| **今年内** | `MM-DD HH:MM`      | `128 tokens · 09-01 14:35`      |
| **跨年**  | `YYYY-MM-DD HH:MM` | `128 tokens · 2024-08-15 14:35` |

### 具体修改步骤

#### 1. 后端：internal/services/ai\_service.go

- **修改**：在 `Message` 结构体（第22行）添加 `CreatedAt time.Time json:"created_at"` 字段

- **修改**：在 `LoadAISessionMessagesPaginated` 的赋值（第516行），添加 `CreatedAt: m.CreatedAt`

#### 2. 后端：app.go

- **修改**：在 `SaveAIMessageResult` 结构体（第2972行）添加 `CreatedAt string `     json:"createdAt"\`\` 字段

- **修改**：在 `SaveAIMessage` 方法（第3006行）返回时，将 `time.Now().Format(time.RFC3339)` 填入返回结果

#### 3. 前端：frontend/src/js/ai-chat.js

- **修改**：`loadSession` 中重建 `chatHistory`（第1697行），map 中添加 `created_at: msg.created_at || null`

- **修改**：`sendUserText` 中 `chatHistory.push()`（第2452行）添加 `created_at: new Date().toISOString()`（前端拿到后端返回的 createdAt 后使用后端值）

- **修改**：`handleRegenerate` 中新建用户消息 `chatHistory.push()`（第5286-5288行）同样添加 `created_at` 字段

- **新增**：在 `createMsgActions` 函数中，为用户消息添加时间显示：

  - 如果 `tokens > 0`，格式：`formatTokens(tokens) + ' tokens · ' + formatSmartTime(createdAt)`

  - 如果 `tokens = 0`，格式：`formatSmartTime(createdAt)`

  - 使用同一个 span 还是分开两个 span？复用现有 `.user-tokens` 元素，直接拼接内容

- **新增**：添加 `formatSmartTime(isoString)` 工具函数，实现上述智能显示规则

#### 4. CSS 样式

现有 `.ai-msg-actions` 已是 flex 布局，新添加的时间会自动跟在 token 后面，不需要额外 CSS 修改。字体颜色继承现有样式，保持一致即可。

## 决策记录

1. **时间格式选择**：智能简化显示，当天只显示时分，旧消息补上月日，跨年补上年份，保持单行紧凑
2. **新增字段位置**：所有数据链路都添加 `created_at`，确保从数据库 → 服务层 → API → 前端 chatHistory 全程可用
3. **返回格式**：使用 ISO 8601 字符串 `time.RFC3339` 传递，前端直接解析，避免时区问题
4. **新建消息 createdAt**：后端 `SaveAIMessage` 生成时间戳并返回给前端，保证时间准确性
5. **显示位置**：直接拼接到 token 所在的 span 中，不新增 DOM 元素，简化实现

## 验证步骤

1. 编译 Go 代码检查语法：`go build ./...`
2. 重新生成 Wails 绑定（如果需要）
3. 测试加载历史消息，每个用户消息下方都应有时间显示
4. 测试发送新消息，新消息应立即显示当前时间
5. 测试重新生成（handleRegenerate），新建的用户消息应有正确时间
6. 测试显示规则：今天、昨天、去年的消息格式正确
7. 测试 token=0 的边界情况（虽然很少见）

## 文件清单

| 文件                                | 变更类型                                         |
| --------------------------------- | -------------------------------------------- |
| `internal/services/ai_service.go` | 新增字段 + 赋值                                    |
| `app.go`                          | 新增字段到返回结构                                    |
| `frontend/src/js/ai-chat.js`      | 新增工具函数 + 修改多处 map/push + 修改 createMsgActions |

