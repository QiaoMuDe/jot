# 设置页侧边栏导航重构 Spec

## Why

当前设置页采用**卡片列表式布局**，9 个设置卡片在 `max-width: 600px` 的容器内垂直排列。随着设置项增多（当前 9 个卡片），用户需要不断滚动才能找到目标设置项，浏览效率低，信息层级不清晰。重构为侧边栏导航布局后，用户可以一键跳转到目标设置区域，同时视觉上更有结构感和精致感。

## What Changes

### 1. 布局结构改造：从卡片列表 → 侧边栏导航 + 内容面板

```
┌──────────────────────────────────────────────────────────┐
│  ← 返回    设置                                          │
├───────────┬──────────────────────────────────────────────┤
│           │                                              │
│  ◉ 外观   │  ┌─ 外观 ─────────────────────────────────┐ │
│  ◉ 编辑器  │  │  字体 大小 主题 预览                    │ │
│  ◉ API连接 │  │                                         │ │
│  ◉ 对话搜索 │  │                                         │ │
│  ◉ 标签管理 │  └──────────────────────────────────────────┘ │
│  ◉ 笔记列表 │                                              │
│  ◉ 日志设置 │                                              │
│  ◉ 锁屏密码 │                                              │
│  ◉ 回收站   │                                              │
│           │                                              │
└───────────┴──────────────────────────────────────────────┘
    176px              flex: 1
```

### 2. 侧边栏设计

- **宽度**: 176px（与笔记本侧栏一致），使用 `flex-shrink: 0`
- **背景色**: `var(--bg-secondary)`，右侧 `1px solid var(--border)` 分隔线
- **导航项**: 每组一个导航条目，包含 SVG 图标 + 标签文本
  - 图标使用项目现有 `ai-group-header` 中的 SVG 图标（保持一致性），尺寸 16×16
  - 标签文本 `font-size: 0.813rem`，`font-weight: 500`
- **悬停态**: `var(--hover-bg)` 背景，平滑过渡
- **激活态**: 左侧 3px `var(--accent)` 竖条 + `var(--accent-lighter)` 背景 + 字体加粗 `font-weight: 600`
- **圆角**: 导航项自身 `border-radius: var(--radius-md)`，右边缘与分隔线保留 8px 间距
- **滚动**: 侧边栏内容超出时 `overflow-y: auto`

### 3. 内容面板设计

- **背景**: 不再使用白色卡片背景（移除 `.settings-section` 的 `background: var(--card-bg)` + `border-radius` + `box-shadow`），改用 `var(--bg)` 纯色背景
- **面板容器**: 每个设置卡片内容包裹在 `.settings-panel` 容器中
- **面板显隐**: 只显示当前激活的面板，其他面板 `display: none`
- **面板标题**: 保留 `.ai-group-header` 作为面板内部标题
- **面板内边距**: `padding: 24px 32px`

### 4. 面板切换动画 (**NEW**)

面板切换时，旧面板淡出并轻微左移，新面板从右侧淡入，形成流畅的"推入"过渡：

- **切换动画定义**:
  - 旧面板退出: `opacity: 1 → 0` + `transform: translateX(0) → translateX(-12px)`，时长 150ms
  - 新面板进入: `opacity: 0 → 1` + `transform: translateX(12px) → translateX(0)`，时长 200ms
  - 两个阶段串行执行（退出 → 进入），总时长 ~350ms
  - **退出动画使用** `var(--anim-easing-out)`，**进入动画使用** `var(--anim-easing-spring)` 获得弹性入效果
- **动画通过 JS 控制 CSS 类实现**: 面板元素添加/移除 `.panel-exit` / `.panel-enter` / `.panel-active` 类
- **`prefers-reduced-motion` 支持**: 动画条件满足时跳过动画，直接切换显示

### 5. 布局体系统一

在本次重构中，将三种历史布局体系统一为 `.ai-setting-item` 三栏格式：
- ✅ `.font-setting-row`（外观、编辑器卡片）→ 迁移到 `.ai-setting-item`
- ✅ `.settings-item`（日志设置卡片）→ 迁移到 `.ai-setting-item`
- 迁移后删除 `.font-setting-row` / `.font-settings` / `.settings-item` 相关样式

### 6. 后端/JS 逻辑调整

- `loadSettings()` 逻辑不变：一次性加载全部设置值，更新到 DOM
- `saveSettings()` 逻辑不变
- 新增 `switchSettingsTab(name)` 函数处理侧边栏切换
- 用户首次打开设置页时默认选中"外观"面板
- 侧边栏点击事件绑定，切换激活状态 + 显示对应面板 + 触发动画

## Impact

- **Affected specs**: 设置页相关
- **Affected code**:
  - `frontend/index.html` — HTML 结构调整：新增侧边栏 DOM，卡片内容改为面板容器
  - `frontend/src/css/components/settings-panel.css` — 新增侧边栏样式、面板切换动画样式，删除旧布局体系
  - `frontend/src/main.js` — 新增 `switchSettingsTab()` 和面板切换动画逻辑
  - 其余 JS/CSS 模块不受影响

## ADDED Requirements

### Requirement: 侧边栏导航

系统 SHALL 在设置页左侧展示一个垂直导航侧边栏，包含 9 个导航条目。

#### Scenario: 正常显示
- **WHEN** 用户打开设置页
- **THEN** 侧边栏显示所有 9 个导航条目，第一个条目（外观）处于激活状态

#### Scenario: 导航切换
- **WHEN** 用户点击侧边栏中的一个非激活导航条目
- **THEN** 该条目变为激活态（左侧竖条 + 高亮背景），对应的设置面板以动画方式显示，之前的面板以动画方式隐藏

#### Scenario: 长列表滚动
- **WHEN** 侧边栏条目超过容器高度
- **THEN** 侧边栏 `overflow-y: auto` 可滚动

### Requirement: 面板切换动画

系统 SHALL 在切换设置面板时提供平滑的过渡动画。

#### Scenario: 正常切换
- **WHEN** 用户点击导航条目切换到新的设置面板
- **THEN** 当前面板 150ms 淡出并左移，新面板 200ms 从右侧淡入

#### Scenario: 减少动效
- **WHEN** 用户的系统设置为 `prefers-reduced-motion: reduce`
- **THEN** 跳过切换动画，面板直接显示/隐藏

## MODIFIED Requirements

### Requirement: 设置内容显示（原卡片列表）

**修改为**：设置内容不再以垂直卡片列表形式一次性显示全部，而是采用侧边栏导航 + 单面板显示。

## REMOVED Requirements

### 旧卡片列表布局
**Reason**: 卡片列表需要滚动浏览，效率低
**Migration**: 替换为侧边栏导航 + 单面板切换
