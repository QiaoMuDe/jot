# Tasks

- [x] Task 1: HTML 结构 — 在 `index.html` 的技能下拉菜单中新增"编程"菜单项
  - [x] 在 `#aiChatSkillsDropdown` 内 "翻译" 菜单项之后添加 `.ai-chat-skills-item`（`data-skill="coding"`），包含代码图标 + "编程" 文字
  - [x] 无需方向选择子菜单（点击即激活）

- [x] Task 2: JS 逻辑 — 在 `ai-chat.js` 中新增编程技能支持
  - [x] `SKILL_PROMPTS` 常量中新增 `coding` 条目（单个 system prompt，无方向配置）
  - [x] 菜单点击事件适配：对 `data-skill="coding"` 的菜单项直接激活（`activeSkills.coding = true`），不展开子菜单
  - [x] `renderSkillChips()` 新增 `coding` 分支：显示"编程"文字 + 代码图标，无方向后缀
  - [x] `getSkillSystemPrompts()` 适配无方向配置的技能（直接取 prompt 字符串而非 `prompt[direction]`）

- [x] Task 3: 验证 — 人工确认功能正常
  - [x] 点击"更多技能" → 菜单显示"翻译"和"编程"两项
  - [x] 点击"编程" → 技能直接激活，菜单关闭，chip 栏显示"编程" chip
  - [x] 发送消息 → API 请求 messages 头部包含编程 system prompt
  - [x] 翻译 + 编程同时激活 → chip 栏显示两个 chip，prompt 拼接正确
  - [x] 切换会话 → 技能重置，chip 消失

## Task Dependencies

- [Task 1] 无依赖（纯 HTML）
- [Task 2] 依赖 [Task 1]（JS 操作的 DOM 元素需要先存在）
- [Task 3] 依赖 [Task 1] 和 [Task 2]（需要完整实现后才能验证）
