# AI 会话侧边栏重新设计 Spec

## Why
当前 AI 会话侧边栏 UI 简陋、留白散乱，整体观感不精致，与 Jot 温暖极简的整体设计调性存在差距。需要在不扩大侧边栏宽度的前提下，通过精细化间距、视觉层次和交互细节提升品质感。

## What Changes
- 压缩并重构侧边栏各区块的间距体系，消除不均匀/过多的留白
- 优化 header 区域（标题 + 新建按钮）的视觉层次
- 重新设计搜索框，使其更紧凑、更融入上下文
- 重构会话条目样式，添加活跃指示器、优化悬停/选中态
- 优化 footer 清空按钮样式
- 增加条目数量角标（可选）
- 优化空状态样式
- 所有改动均复用现有的 `--bg`, `--card-bg`, `--text-*`, `--accent`, `--border`, `--hover-bg` 等 CSS 变量，与多主题兼容

## Impact
- Affected specs: AI Chat session sidebar UI
- Affected code: `frontend/src/css/components/ai-chat.css`, `frontend/index.html`（仅空状态文本微调）
- No JS logic changes needed（纯 CSS 重设计）

## ADDED Requirements

### Requirement: 侧边栏整体间距压缩
The sidebar SHALL use a tighter, consistent spacing system.

#### Scenario: 各区块间距
- **WHEN** 侧边栏渲染
- **THEN** header padding 从 `10px 12px` 压缩到 `8px 10px`
- **THEN** search-wrap padding 从 `8px 10px 8px` 压缩到 `6px 10px 4px`
- **THEN** list padding 保持 `6px 4px 6px 8px`
- **THEN** footer padding 从 `8px 10px` 压缩到 `6px 10px`

### Requirement: 重构 Header 区域
The sidebar header SHALL have a more refined appearance.

#### Scenario: 标题与新建按钮
- **WHEN** 侧边栏 header 渲染
- **THEN** 标题"会话"字号提升到 `0.82rem`，weight 保持 600
- **THEN** 新建按钮尺寸保持 `26px`，hover 时使用 `var(--accent)` 着色
- **THEN** header 底部用更细的 `1px solid var(--border)` 分隔

### Requirement: 重新设计搜索框
The search input SHALL be more compact and visually integrated.

#### Scenario: 搜索框样式
- **WHEN** 搜索框渲染
- **THEN** padding 从 `5px 8px` 压缩到 `4px 8px`
- **THEN** height 缩小以匹配新间距
- **THEN** placeholder 颜色使用 `var(--text-muted)`

### Requirement: 会话条目新增活跃指示器
Each session item SHALL have a visual indicator for active state.

#### Scenario: 活跃/非活跃条目
- **WHEN** 条目为活跃状态（已选中）
- **THEN** 左侧显示 2px 宽的 `var(--accent)` 竖条（绝对定位，`left: 0`）
- **THEN** 背景使用 `color-mix(in srgb, var(--accent) 10%, transparent)`
- **WHEN** 条目为非活跃状态
- **THEN** 左侧无竖条
- **WHEN** 条目 hover
- **THEN** 背景使用 `var(--hover-bg)`，删除按钮显示

### Requirement: 会话条目间距紧凑
Session items SHALL have compact but comfortable spacing.

#### Scenario: 条目内边距
- **WHEN** 条目渲染
- **THEN** padding 为 `6px 4px`（已实现，保持不变）
- **THEN** 条目之间无额外 margin（由上下 padding 自然分隔）

### Requirement: 清空按钮样式优化
The clear button SHALL have a more refined appearance.

#### Scenario: 清空按钮
- **WHEN** footer 渲染
- **THEN** 按钮使用 `var(--card-bg)` 背景，`var(--text-secondary)` 文字
- **THEN** hover 时背景变 `var(--hover-bg)`，文字变 `var(--text-primary)`
- **THEN** 圆角使用 `var(--radius-sm)`

## MODIFIED Requirements

### Requirement: 空状态样式
The empty state SHALL be centered and visually soft.

#### Scenario: 无会话/无搜索结果
- **WHEN** `filteredSessions.length === 0`
- **THEN** 显示居中文字，颜色为 `var(--text-muted)`，字号 `0.82rem`，padding 上 `24px`
- **THEN** 文字内容保持"暂无会话"

### Requirement: 宽度调整为 230px
The sidebar width SHALL be slightly widened for better readability.

#### Scenario: 侧边栏宽度
- **WHEN** 侧边栏展开
- **THEN** width 从 `220px` 调整为 `230px`
- **WHEN** 侧边栏折叠
- **THEN** width 为 `0`
- **WHEN** 侧边栏展开时折叠按钮位置从 `left: 220px` 调整为 `left: 230px`

## REMOVED Requirements
无删除项
