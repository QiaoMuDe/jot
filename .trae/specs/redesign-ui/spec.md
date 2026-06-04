# Jot UI 全面重设计 Spec

## Why

当前 Jot 应用的 UI 风格老旧、视觉设计不佳，缺乏统一的品牌感和设计语言。需要一次全面的视觉重构，让应用看起来更现代、专业且有温度。

## Design Direction: 温暖极简主义 (Warm Minimalism)

**风格定位**：以温暖、干净、有质感的极简设计，营造舒适专注的笔记体验。

**设计系统**：

| 设计维度 | 选型 |
|----------|------|
| **模式** | 浅色模式（Light Mode）为主 |
| **风格** | 温暖极简（Warm Minimalism）+ 自然有机感 |
| **配色** | 暖白基调 + 琥珀色点缀 |
| **字体** | DM Sans（英文）+ 系统 sans-serif（中文后备） |
| **圆角** | 卡片 12px，按钮 8px，输入框 10px |
| **阴影** | 柔和暖阴影（warm shadow），使用 rgba(0,0,0,0.06)-rgba(0,0,0,0.10) |
| **动画** | 150-250ms ease-out，微交互动画 |

### 配色方案

```
背景:         #F7F5F0 (暖白)
卡片/面板:    #FFFFFF
主文字:       #2D2A24 (暖黑)
次文字:       #8B867C (暖灰)
主题色:       #D97706 (琥珀色)
主题色浅:     #FDE68A (琥珀色浅)
主题色深:     #B45309
边框:         #E5E0D8
分割线:       #EBE6DE
危险色:       #DC2626
悬停背景:     #F3F0EB
选中高亮:     #FEF3C7 (琥珀色极浅)
☆置顶标记:    #D97706 (琥珀色)
❤️品牌元素:   #D97706
复选框/选中:  #D97706
```

### 字体系统

```
标题:         DM Sans 600 18px (品牌名/大标题)
卡片标题:     DM Sans 500 15px
正文字体:     DM Sans 400 14px
小字/时间:    DM Sans 400 12px
代码/内容:    DM Sans 400 14px
```

### 组件风格

| 组件 | 风格 |
|------|------|
| **Topbar** | 暖白背景，底部 1px 边框，品牌左对齐，操作右对齐 |
| **卡片** | 白色卡片，热身阴影，圆角 12px，hover 上移 -2px |
| **按钮** | 填充按钮（主题色），文字按钮（透明+悬停色），危险按钮（红色） |
| **输入框** | 暖白背景，10px 圆角，聚焦时主题色边框 |
| **模态框** | 居中弹窗，热身阴影，顶部标题栏 |
| **标签 chip** | 小圆角，彩色背景，白色文字 |
| **视图切换** | 页面 title + 内容区域 |

---

## What Changes

### 全局 (Global)
- CSS 变量系统：所有颜色、阴影、圆角、间距统一为语义变量
- 字体替换：从系统字体 → DM Sans（Google Fonts）
- 全局间距：使用 4px 基底（4/8/12/16/20/24/32/48/64）
- 全局过渡：hover/focus 统一 150ms ease-out
- 移除 emoji 图标，统一使用简洁的 SVG/Unicode 符号

### 1. Topbar 重设计
- 品牌 "Jot" 文字使用 DM Sans 600，琥珀色
- 搜索框：暖白背景，圆角 10px，左侧放大镜图标
- 操作按钮组：+ / ✓ / ☰，统一圆角 8px，border 1px
- hover 和 active 状态使用主题色
- 高度 56px，垂直居中

### 2. 卡片网格视图重设计
- 网格布局不变（auto-fill, minmax(280px, 1fr)），间距 16px
- 卡片：白色背景，热身阴影，圆角 12px，padding 16px
- 卡片 hover：上移 2px，阴影加深
- 卡片标题：DM Sans 500 15px，最多 2 行截断
- 卡片内容预览：DM Sans 400 14px，暖灰 #8B867C，最多 3 行截断
- 卡片时间：小字 12px，暖灰
- 标签 chip：小圆角 6px，padding 2px 8px，font-size 11px
- 置顶徽标（📌）：改为 ★（星号），琥珀色
- 批量模式复选框：accent-color 琥珀色
- 选中态：琥珀色极浅背景高亮 #FEF3C7，边框琥珀色

### 3. 编辑器模态框重设计
- 模态框：居中，最大宽 640px，热身阴影，圆角 16px
- 标题区域：顶部品牌色条 3px
- 标题输入：DM Sans 600 20px，无边框，聚焦时底部 1px 边框
- 内容文本域：DM Sans 400 15px，行高 1.7，暖灰文字
- 底部栏：标签选择器 + 颜色选择器 + 字数统计 + 保存按钮
- 自动保存指示器：右侧绿色小圆点 + "已保存" 文字
- 关闭按钮：右上角 X 图标

### 4. 搜索视图重设计
- 搜索结果列表：左侧内容预览 + 右侧时间和标签
- 搜索结果高亮：琥珀色背景 #FDE68A
- 空结果：居中显示温暖插画感提示
- 搜索框聚焦时主题色边框

### 5. 设置页面重设计
- 简洁列表布局
- 条目：标题 + 说明文字 + 右侧操作
- 分割线使用暖色

### 6. 数据管理页面重设计
- 统计卡片：三列网格，暖白卡，热身阴影，圆角 12px
- 每个统计卡：图标 + 数字 + 标签
- 操作按钮组：水平排列，间距 12px
- 导入/结果 toast 使用主题色

### 7. 回收站视图重设计
- 列表布局（同搜索风格）
- 每条：标题 + 删除时间 + 操作按钮
- 按钮：恢复（主题色文字）+ 永久删除（危险色文字）
- "全部恢复" / "全部清空" 为顶部操作栏

### 8. 批量操作栏重设计
- 暖白背景 + 暖色边框
- 信息文字 "已选 X 条"
- 按钮：批量删除（危险色）+ 退出批量（文字按钮）
- 动画：slideDown 150ms ease-out

### 9. 右键菜单重设计
- 圆角 10px，热身阴影
- 菜单项 padding 8px 16px，hover 暖色背景
- 危险操作（删除）hover 红色背景

### 10. Toast/通知重设计
- 底部居中固定
- 圆角 10px，热身阴影
- "已撤销" 等操作反馈使用主题色

---

## Impact

- **Affected specs**: 所有已有 spec（UI 全覆盖）
- **Affected code**:
  - `frontend/src/style.css` — 完全重写样式系统
  - `frontend/src/app.css` — 可能不再需要或简化
  - `frontend/src/main.js` — 少量样式类名调整
  - `frontend/index.html` — 少量结构调整（添加字体引用）
- **不涉及**：后端 Go 代码、模型、业务逻辑

---

## ADDED Requirements

### Requirement: 设计系统 CSS 变量
The system SHALL define CSS custom properties for all design tokens.

#### Scenario: CSS 变量加载
- **GIVEN** 页面加载
- **WHEN** CSS 文件被解析
- **THEN** `:root` 中定义了所有颜色、阴影、圆角、间距、字体变量

### Requirement: 字体加载
The system SHALL load DM Sans font from Google Fonts.

#### Scenario: 字体引用
- **GIVEN** 页面加载
- **WHEN** HTML 被解析
- **THEN** Google Fonts 加载 DM Sans (400, 500, 600)
- **AND** CSS 中使用 `font-family: 'DM Sans', system-ui, sans-serif`
- **AND** 字体使用 `font-display: swap` 防止 FOIT

---

## MODIFIED Requirements

### Requirement: 卡片渲染样式（原 renderCardGrid）
卡片 HTML 结构保持不变，CSS 类名和样式完全更新：

- `note-card` → 新样式（热身阴影、12px 圆角、白色背景）
- `selected` → 琥珀色极浅高亮边框
- `batch-mode` → 复选框替换置顶按钮

### Requirement: 编辑器样式（原 editor 模态框）
编辑器 HTML 结构基本不变，CSS 样式完全更新：

- `.editor-panel` → 最大 640px，16px 圆角
- `.editor-title-input` → DM Sans 600，聚焦时底部边框
- `.editor-content-textarea` → DM Sans 400，行高 1.7

---

## REMOVED Requirements

无功能被移除。仅 CSS 样式完全替换。
