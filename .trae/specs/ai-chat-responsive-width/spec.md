# AI 对话区响应式宽度适配 Spec

## Why
AI 助手界面的消息列表固定在 `max-width: 900px`，在宽屏桌面窗口下利用率不足 50%，且输入区已撑满全宽造成视觉不一致。需要让对话区随窗口宽度弹性缩放，兼顾宽屏的空间利用和可读性。

## What Changes
- 将 `.ai-chat-messages` 的 `max-width: 900px` 改为 `clamp()` 响应式约束
- 为 `.ai-chat-input-area` 增加与消息区一致的宽度约束，保持视觉对齐
- 调整消息气泡的 `max-width` 百分比约束，确保宽屏下不出现过宽文本
- 考虑侧栏折叠状态对可用宽度的影响

## Impact
- Affected specs: AI 助手相关样式
- Affected code: `frontend/src/css/components/ai-chat.css`

## ADDED Requirements

### Requirement: 响应式宽度约束
系统 SHALL 根据窗口宽度动态调整 AI 对话区的可用宽度。

#### Scenario: 正常窗口 (1200px 宽)
- **WHEN** 窗口宽度约 1200px
- **THEN** 消息列表宽度适配到约 750-850px

#### Scenario: 宽屏窗口 (1920px+)
- **WHEN** 窗口宽度 1920px 以上
- **THEN** 消息列表宽度适配到合理上限（约 1000-1100px），不无限拉伸

#### Scenario: 窄屏窗口 (<900px)
- **WHEN** 窗口宽度小于 900px
- **THEN** 消息列表宽度跟随窗口缩小，保留合适边距

### Requirement: 输入区与消息区宽度对齐
系统 SHALL 使输入区宽度与消息列表宽度保持一致。

#### Scenario: 输入区对齐
- **WHEN** 窗口宽度变化
- **THEN** 输入区宽度与消息列表同步变化，保持左右边距一致

### Requirement: 消息气泡宽度约束
系统 SHALL 确保单条消息在宽屏下不出现过宽的情况。

#### Scenario: 宽屏窗口下长消息
- **WHEN** 用户在宽屏下发送长消息
- **THEN** 消息气泡宽度受 `max-width` 百分比限制，保持可读性

## MODIFIED Requirements

### Requirement: 现有消息列表宽度
**修改前**: `max-width: 900px` 固定值
**修改后**: `max-width: clamp(800px, min(92vw, 100%), 1600px)` 响应式值，同时输入区也应用相同约束
