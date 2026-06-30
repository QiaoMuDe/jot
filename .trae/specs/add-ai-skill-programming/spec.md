# AI 助手更多技能 — 编程技能 Spec

## Why

为 AI 助手的"更多技能"系统新增"编程"技能，用户激活后 AI 将扮演资深程序员角色，提供代码编写、调试、架构设计、技术方案评估等编程相关服务，支持多种编程语言和最佳实践。

## What Changes

- **修改** `frontend/index.html` — 在技能下拉菜单中新增"编程"菜单项（无方向选择子菜单，点击即激活）
- **修改** `frontend/src/js/ai-chat.js` — 
  - `SKILL_PROMPTS` 新增 `coding` 条目（单条 system prompt，无方向配置）
  - `renderSkillChips()` 新增 `coding` 类型的 chip 渲染分支（无需方向标签）
  - `getSkillSystemPrompts()` 适配无方向配置的技能
  - 技能菜单点击事件适配无方向配置的技能
- **修改** `frontend/src/css/components/ai-chat.css` — 无新增样式（复用现有技能相关样式）

## Impact

- Affected specs: AI 对话交互 — 更多技能
- Affected code:
  - `frontend/index.html` — 新增"编程"菜单项
  - `frontend/src/js/ai-chat.js` — 新增编程 prompt、更新 skill chip 渲染和 prompt 获取逻辑

## ADDED Requirements

### Requirement: 编程技能激活

系统 SHALL 在"更多技能"下拉菜单中新增"编程"技能项，点击即激活（无需方向选择）。

#### Scenario: 点击激活编程技能
- **WHEN** 用户点击"更多技能"按钮展开菜单
- **AND** 用户点击"编程"菜单项
- **THEN** 技能立即激活，菜单关闭
- **AND** 输入区上方显示 chip "编程"

### Requirement: 编程技能 chip 指示

系统 SHALL 在技能 chip 指示栏中显示编程技能的激活状态。

#### Scenario: 编程技能 chip 显示与取消
- **WHEN** 编程技能已激活
- **THEN** 输入区上方显示 chip，内容为"编程"（无方向后缀）
- **AND** chip 左上角显示代码图标
- **AND** chip 右上角有叉号，点击可取消技能

### Requirement: 编程 system prompt 注入

系统 SHALL 在发送消息时，若编程技能已激活，将编程 system prompt 注入到消息列表头部。

#### Scenario: 编程技能激活时发送消息
- **WHEN** 编程技能已激活
- **AND** 用户发送消息
- **THEN** 请求 API 时在 messages 数组头部插入编程相关的 system prompt

#### Scenario: 多技能共存
- **WHEN** 编程技能和翻译技能同时激活
- **THEN** 两个 system prompt 按技能激活顺序拼接
- **AND** chip 栏同时显示两个 chip

### Requirement: 会话切换重置

系统 SHALL 在切换或新建会话时重置编程技能状态。

#### Scenario: 编程技能不跨会话保持
- **WHEN** 用户切换到其他会话或新建会话
- **THEN** 编程技能自动取消激活，chip 消失

## Coding System Prompt

```
# Role: 资深程序员

## Core Task
为用户提供专业的编程相关服务，包括但不限于代码编写、调试修复、架构设计、技术方案评估和最佳实践建议。

## Guidelines
- 代码质量优先：编写的代码应遵循对应语言的最佳实践，注重可读性、可维护性和性能
- 全面考虑：对逻辑需要分析前置条件、边界情况和异常处理，确保稳健性
- 主动解释：提供代码的同时解释关键设计思路和技术选型理由
- 保持简洁：尽量用简洁高效的代码解决问题，避免过度设计
- 格式规范：代码块标注正确的语言类型，便于高亮显示
```
