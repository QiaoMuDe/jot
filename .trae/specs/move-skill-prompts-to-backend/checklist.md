# Checkpoints

- [x] 首次运行自动创建 `ai_prompts` 表并向其中插入 11 条内置提示词
- [x] 数据库 `ai_prompts` 表中有 `skill_coding` / `skill_writing` 等 key 的正确记录，内容与原前端一致
- [x] 后端 `GetSkillPrompts(["coding", "writing"])` 返回拼接后的提示词字符串
- [x] `CallAIStream` 收到 `skillIds` 后，技能提示词正确注入到 messages 的 system role 中
- [x] 前端 `SKILL_PROMPTS` 常量和 `getSkillSystemPrompts()` 函数已删除
- [x] 前端 `startStreaming()` 不再处理技能提示词拼接，改为将技能 ID 列表传给后端
- [x] 开启技能（如编程）后发送消息 → AI 回复体现技能特性（如代码相关内容）
- [x] 同时开启多个技能（如翻译+编程）→ 多条技能提示词正确合并注入
- [x] 不开启任何技能 → 系统行为与原来一致，无技能提示词注入
- [x] "优化表达"按钮（OPTIMIZE_EXPRESSION_PROMPT）功能不受影响
