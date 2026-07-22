# Translate Skill Redesign — Language Direction Picker Spec

## Why

当前翻译功能在「更多技能」中使用嵌套子菜单选择方向，操作路径长（4 步），且只支持中/英互译。改造后扁平化为直接激活项，chip 上可视化展示源语言→目标语言，支持更多语言，整体操作更直观高效。

## What Changes

### 前端 HTML/CSS/JS

- **修改** `index.html` — 翻译从具有展开/收起子菜单的菜单项改为普通技能菜单项（移除 chevron 图标和 `#aiChatTranslateOptions` 子菜单）
- **修改** `ai-chat.js` — 
  - 移除 `skillsTranslateOptions` 相关的展开/收起/radio 同步逻辑（~40 行）
  - `activeSkills.translate` 数据结构从 `{ direction: 'to_chinese' }` 改为 `{ source: 'english', target: 'chinese' }`
  - 翻译在技能菜单中点击直接激活（保持上次方向设置或默认 english→chinese）
  - 翻译 chip 改为双语言组件：`[🌐] [English] ⇄ [Chinese] [×]`
  - 点击左侧/右侧语言弹出语言选择浮层
  - 新增语言选择浮层的打开/关闭/选择事件
  - `renderSkillChips()` 中翻译 chip 的 HTML 重构
  - 会话配置保存/恢复兼容旧数据格式（`direction` → `source+target` 迁移）
- **新增** `ai-chat.css` — 翻译 chip 双语言布局样式、语言选择浮层样式
- **移除** `ai-chat.css` — `.ai-chat-skills-options` 相关样式（~50 行，max-height 动画、radio 选中态等）

### 后端 Go

- **修改** `internal/database/db.go` — 将 `skill_translate_cn` 和 `skill_translate_en` 两条内置 prompt 合并为一条通用翻译 prompt，使用 `{source}` 和 `{target}` 占位符
- **修改** `internal/services/ai_service.go` — 在技能 prompt 注入逻辑中，当 `skillId` 为 `skill_translate` 时，根据前端传入的 source/target 替换占位符生成动态 prompt

### 数据结构变更

```
旧: activeSkills.translate = { direction: 'to_chinese' }
新: activeSkills.translate = { source: 'english', target: 'chinese' }
```

## Impact

- Affected specs: AI 对话 — 更多技能 / 翻译功能
- Affected code:
  - `frontend/index.html` — 移除翻译子菜单相关结构
  - `frontend/src/js/ai-chat.js` — JS 逻辑重构（状态管理、chip 渲染、事件绑定）
  - `frontend/src/css/components/ai-chat.css` — 移除子菜单样式、新增 chip + 浮层样式
  - `internal/database/db.go` — 修改内置提示词数据
  - `internal/services/ai_service.go` — 动态提示词注入

## ADDED Requirements

### Requirement: 扁平化翻译菜单项

系统 SHALL 将「翻译」改为与其他技能一致的直接点击激活菜单项。

#### Scenario: 点击翻译直接激活
- **WHEN** 用户打开「更多技能」菜单并点击「翻译」
- **THEN** 翻译技能立即激活（使用上次保存的方向或默认 english→chinese）
- **AND** 菜单关闭，输入区上方显示翻译 chip
- **AND** 不需要额外的方向选择步骤

### Requirement: 翻译 chip 双语言显示

系统 SHALL 在翻译 chip 中以「源语言 ⇄ 目标语言」格式显示翻译方向。

#### Scenario: chip 显示语言方向
- **WHEN** 翻译技能已激活
- **THEN** chip 显示为 `[🌐] [English] ⇄ [Chinese] [×]`
- **AND** 左侧「English」为源语言（可点击）
- **AND** 右侧「Chinese」为目标语言（可点击）
- **AND** 中间箭头为装饰性分隔
- **AND** 右侧 × 按钮取消翻译技能

### Requirement: 语言选择浮层

系统 SHALL 在用户点击左侧或右侧语言时，弹出一个语言选择浮层。

#### Scenario: 点击语言弹出选择
- **WHEN** 用户点击 chip 上的源语言或目标语言
- **THEN** 在点击位置附近弹出一个浮层菜单，列出可选语言
- **AND** 浮层包含：English、中文、日本語、한국어、Français、Deutsch、Español、Русский、العربية、Português
- **AND** 当前语言处于选中态（高亮 + ✓）
- **AND** 用户选择后浮层关闭，chip 更新

#### Scenario: 选择相同语言处理
- **WHEN** 用户选择与另一侧相同的语言
- **THEN** 自动交换两侧语言（保持 source ≠ target）
- **AND** 浮层关闭

### Requirement: 语言选择浮层关闭

系统 SHALL 在以下情况关闭语言选择浮层：
- 用户点击了某个语言选项
- 用户点击浮层外部区域
- 用户再次点击当前激活的语言标签

### Requirement: 向后兼容

系统 SHALL 兼容旧格式的会话配置数据。

#### Scenario: 加载旧会话配置
- **WHEN** 加载已保存的会话配置中包含旧格式 `translate.direction === 'to_chinese'`
- **THEN** 自动转换为 `{ source: 'english', target: 'chinese' }`
- **AND** 不会丢失翻译技能状态

### Requirement: 动态翻译提示词

系统 SHALL 使用一条通用翻译提示词，根据 source/target 动态替换。

#### Scenario: 发送翻译请求
- **WHEN** 翻译技能已激活（source: 'english', target: 'chinese'）
- **AND** 用户发送消息
- **THEN** 后端注入的提示词为：`将以下 English 消息翻译成 Chinese`

## MODIFIED Requirements

### Requirement: 技能激活指示 (修改)

旧：翻译 chip 显示「翻译 → 英文」或「翻译 → 中文」
新：翻译 chip 显示源语言和目标语言（如「English ⇄ Chinese」），两侧语言可点击切换

### Requirement: 更多技能按钮与下拉菜单 (移除翻译子菜单)

从「更多技能」下拉菜单中移除翻译相关的展开/收起子菜单逻辑。翻译与其他技能并列作为普通菜单项。

## REMOVED Requirements

### Requirement: 翻译方向子菜单

**Reason**: 扁平化设计，无需在菜单内嵌套方向选择
**Migration**: 方向选择移至激活后的 chip 上，通过点击语言标签弹出浮层选择

### Requirement: 翻译方向 radio 选项

**Reason**: 用语言选择浮层替代 radio 选项
**Migration**: 方向选择通过 chip 上的语言标签交互实现
