# 启动器网格 (Launcher Grid) Spec

## Why
在现有的"更多"下拉菜单之外，新增一个全屏浮层式启动器网格。通过 `Ctrl+P` 快捷键一键呼出，支持搜索过滤和方向键导航，为习惯键盘操作的用户提供更高效的导航方式。与现有"更多"菜单**并存**，互不干扰。

## What Changes
- 新增浮层式启动器组件（overlay），通过快捷键 `Ctrl+P` 呼出
- 启动器内包含搜索框（顶部）和 4 列网格卡片（展示 13 个功能项）
- 网格卡片使用与"更多"菜单一致的 SVG 图标
- 支持键盘方向键导航 + Enter 执行 + Esc 关闭
- 打开时自动聚焦搜索框
- 包含入场/离场过渡动画
- 全局 ESC 处理中增加启动器优先关闭逻辑
- 现有的"更多"下拉菜单**完全不变**

## Impact
- 新增 CSS 文件: `frontend/src/css/components/launcher.css`（~200 行）
- 修改 HTML: `frontend/index.html`（新增启动器 DOM 结构）
- 修改 JS: `frontend/src/main.js`（新增启动器逻辑 + 全局快捷键处理）
- 现有"更多"菜单行为不受影响

## ADDED Requirements

### Requirement: 启动器浮层
启动器为全屏浮层，居中显示，类似 macOS Spotlight / Raycast 风格。

#### Scenario: 通过快捷键打开
- **WHEN** 用户按下 `Ctrl+P`
- **THEN** 全屏遮罩淡入（backdrop-filter: blur），启动器面板从中心缩放 + 淡入
- **AND** 搜索框自动获得焦点，光标在输入框中闪烁

#### Scenario: 搜索过滤
- **WHEN** 用户在搜索框中输入文字
- **THEN** 网格卡片实时过滤，仅显示名称包含输入文字（substring）的项
- **AND** 显示的第一个卡片获得选中状态（键盘焦点样式）

#### Scenario: 键盘导航
- **WHEN** 用户在启动器内按下方向键（↑↓←→）
- **THEN** 按网格布局（4 列）顺序移动选中状态，支持行/列循环
- **AND** 选中的卡片有高亮样式
- **WHEN** 用户按下 Enter
- **THEN** 执行选中功能项对应的操作
- **AND** 播放离场动画后关闭启动器

#### Scenario: 鼠标操作
- **WHEN** 用户点击某个卡片
- **THEN** 执行对应功能
- **AND** 关闭启动器
- **WHEN** 用户点击遮罩区域
- **THEN** 关闭启动器

#### Scenario: 关闭启动器
- **WHEN** 用户按下 ESC（在全局 handleKeyboardNavigation 中处理）
- **THEN** 离场动画播放（淡出 + 缩小），动画结束后隐藏
- **WHEN** 执行功能项后
- **THEN** 同上关闭

### Requirement: 功能项列表
启动器网格展示 13 个功能项，与"更多"菜单一致。

| data-action | 中文名称 | 说明 |
|---|---|---|
| home | 笔记首页 | 切换 grid 视图 |
| sidebar-toggle | 展开侧栏 | 切换笔记本侧栏折叠 |
| batch-mode | 批量管理 | 进入批量管理模式 |
| data | 数据管理 | 切换 data 视图 |
| trash | 回收站 | 切换 trash 视图 |
| settings | 设置 | 切换 settings 视图 |
| calendar | 笔记日历 | 切换 calendar 视图 |
| todo | 待办清单 | 切换 todo 视图 |
| ai-chat | AI 助手 | 切换 ai-chat 视图 |
| help | 快捷键说明 | 打开快捷键弹窗 |
| md-ref | MD 语法 | 切换 md-ref 视图 |
| about | 关于 | 打开关于弹窗 |

### Requirement: 启动器样式

#### 布局
- 遮罩：全屏 fixed，z-index: 2100（高于 search-modal 的 2000）
- 面板：居中，宽度 ~520px
- 搜索框：顶部内边距，带搜索图标和提示文字
- 网格：4 列 grid，gap: 10px，每个卡片 ~110px 宽、~80px 高
- 卡片内容：24×24 SVG 图标 + 12px 文字标签（纵向排列，图标在上文字在下）

#### 主题
- 使用现有 CSS 变量，与 search-modal 风格保持一致
- 遮罩背景：`var(--overlay-bg)`
- 面板背景：`var(--card-bg)`
- 卡片默认：透明背景
- 卡片 hover/选中：`var(--hover-bg)` 背景 + 轻微上移效果

#### 动画
- 入场：遮罩 `opacity 0→1`（0.2s ease-out），面板 `scale(0.92)→scale(1)` + `translateY(-8px)→translateY(0)` + `opacity 0→1`（0.28s cubic-bezier(0.16, 1, 0.3, 1)）
- 离场：遮罩 `opacity 1→0`，面板 `scale(1)→scale(0.92)` + `opacity 1→0`（0.15s ease-in）
- 卡片 stagger：每个卡片延迟 20ms 依次出现，使用 `nth-child` 设置 `animation-delay`

## MODIFIED Requirements

### Requirement: 全局键盘快捷键
在 `handleKeyboardNavigation()` 中新增两个修改点：

1. **新增 `Ctrl+P` 分支**
   - 位置：在所有 `Ctrl+` 快捷键处理之后，在 F11 处理之前
   - **WHEN** 用户按下 `Ctrl+P`
   - **THEN** `e.preventDefault()`（阻止浏览器默认打印）
   - **AND** 如果启动器已打开则关闭，否则打开启动器

2. **修改 ESC 处理逻辑**
   - 在当前 ESC 处理分支中，新增启动器优先关闭判断
   - 位置：`#1` 引用笔记选择器浮层判断之后，搜索弹窗判断之前
   - **WHEN** 按下 ESC 且启动器处于打开状态
   - **THEN** 关闭启动器（播放离场动画）

## REMOVED Requirements
无。现有"更多"菜单完全保留，不受影响。
