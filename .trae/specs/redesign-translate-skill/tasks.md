# Tasks

- [x] Task 1: HTML 结构调整 — 翻译从子菜单改为普通菜单项
  - [ ] 移除 `index.html` 中翻译菜单项的 chevron 图标（`.ai-chat-skills-chevron`）
  - [ ] 移除 `#aiChatTranslateOptions` 整个子菜单结构（两个 radio label）
  - [ ] 翻译菜单项不需要额外属性，与其他技能（如 coding/writing）结构一致

- [x] Task 2: CSS 样式改造 — 移除旧子菜单样式，新增翻译 chip 双语言布局 + 语言选择浮层
  - [ ] 移除 `.ai-chat-skills-options`、`.ai-chat-skills-option`、`selected::before` 等旧样式（~50 行）
  - [ ] 新增翻译 chip 双语言样式：`ai-chat-skill-chip-translate` flex 布局，左右语言可点击（hover 高亮），中间箭头装饰
  - [ ] 新增语言选择浮层样式：圆角卡片、阴影、列表项 hover/selected 态、入场动画
  - [ ] 浮层定位在 chip 语言标签下方，避免溢出

- [x] Task 3: JS 核心逻辑重构 — 扁平化激活、chip 渲染、语言浮层事件
  - [ ] 移除 `skillsTranslateOptions` 相关变量和全部展开/收起/radio 同步逻辑
  - [ ] 修改 `activeSkills.translate` 数据结构为 `{ source: 'english', target: 'chinese' }`
  - [ ] 翻译菜单项点击直接激活（`skill === 'translate'` 与其他技能走相同分支）
  - [ ] 新增 10 种语言常量表（含语言名称与对应代码）
  - [ ] `renderSkillChips()` 中翻译 chip 渲染为双语言格式
  - [ ] chip 上语言标签的 click 事件：打开语言选择浮层
  - [ ] 语言选择浮层的选项点击处理（含相同语言自动交换逻辑）
  - [ ] 点击外部关闭浮层
  - [ ] 会话配置保存/加载兼容旧数据格式（`direction` → `source+target` 迁移）

- [x] Task 4: 后端 Prompt 改造 — 合并为动态翻译提示词
  - [ ] 修改 `internal/database/db.go`，用一条通用 `skill_translate` prompt 替代 `skill_translate_cn` + `skill_translate_en`
  - [ ] 通用 prompt 使用 `{source}` 和 `{target}` 占位符
  - [ ] 修改 `internal/services/ai_service.go`，当技能 key 为 `skill_translate` 时，解析前端传入的 source/target，替换占位符
  - [ ] 前端发送 skillIds 时，translate 对应的 key 改为 `skill_translate`（不再区分 `skill_translate_cn/en`）

## Task Dependencies

- [Task 1] 无依赖
- [Task 2] 依赖 [Task 1]（CSS 基于新的 HTML 结构）
- [Task 3] 依赖 [Task 1] 和 [Task 2]（JS 操作 DOM 元素需 HTML/CSS 就绪）
- [Task 4] 可独立于前端任务并行开发
