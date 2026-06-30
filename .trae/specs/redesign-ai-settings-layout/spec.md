# AI 设置模块布局重构 Spec

## Why

当前 AI 设置模块所有选项平铺在一个 `font-settings` 容器中，8 个设置项（API 地址、服务商、API Key、模型、深度思考、引用截断、联网搜索、卡片召回）挤在一起，没有视觉分组，Tavily 配置更是混杂了 API key、开关、提示文字，整体缺乏信息层级和浏览舒适度。

## What Changes

- 将 AI 设置重构为 **3 个分组卡片**，按功能领域划分
- 为每个分组添加标题和图标，建立清晰的视觉层级
- 优化单个设置项的布局和间距，减少 inline style 的使用
- 将开关（Toggle）与具体输入项就近组合，避免分离
- 联网搜索（Tavily）拆分为独立配置区域，API key + 开关 + 提示文字统一收纳
- 替换 emoji 图标为 SVG 图标
- 新增 scoped CSS 类，替代现有 inline style

## Impact

- Affected specs: settings-panel
- Affected code:
  - `frontend/index.html` — AI 设置 HTML 结构重构
  - `frontend/src/css/components/settings-panel.css` — 新增 scoped 样式
  - `frontend/src/main.js` — 对应的 DOM 元素选择器保持不变

## ADDED Requirements

### Requirement: AI 设置分组
AI 设置页面的所有配置项按 3 组划分：

#### Group 1: API 连接（API 地址、服务商、API Key、模型）
- 标题 "API 连接" + 连接图标
- 每个配置项一行，标签 + 控件对齐

#### Group 2: 对话增强（深度思考、引用截断、卡片召回）
- 标题 "对话增强" + 魔法棒/星星图标
- 深度思考：开关 + 描述文字
- 引用截断：数字输入 + "字符/条" 标签
- 卡片召回：开关 + 描述文字，下方缩进显示召回条数输入

#### Group 3: 联网搜索（Tavily API Key、搜索开关）
- 标题 "联网搜索" + 搜索图标
- Tavily API Key 输入 + 显示/隐藏 + 测试按钮
- 提示文字 "前往 tavily.com 注册获取"
- 搜索开关：默认启用联网搜索

### Requirement: 视觉风格优化
No visual coupling with font settings. 使用独立的 CSS 类 `.ai-settings-group` / `.ai-setting-item` / `.ai-setting-label` 等。

- 分组卡片：卡片背景、圆角、内边距、间距，每个分组上方有标题行
- 设置项行：flex 布局，标签固定宽度 80px，控件弹性填充
- 开关对齐：开关靠右对齐
- 数字输入框宽度固定 100px
- 提示文字 0.78rem、次级色、灰显
- SVG 图标替换 emoji

## MODIFIED Requirements

### Requirement: 开关状态共享
The design already uses localStorage + database for toggle states. No change to the data layer.

## REMOVED Requirements

### Requirement: `font-setting-row` 复用
**Reason**: AI 设置不再复用字体设置的 CSS 类，改用独立的 AI 设置样式体系。
**Migration**: 新增 `.ai-settings-group` / `.ai-setting-item` 等独立 CSS 类替换。
