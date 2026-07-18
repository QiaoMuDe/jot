# 将默认主题从 `:root` 剥离的计划

## 概要

将 `variables.css` 中 `:root` 的双重职责（共享设计令牌 + 默认主题值）分离，`:root` 仅保留与主题无关的共享令牌，默认主题值移至独立的 `[data-theme="default"]` 块。

## 现状分析

`variables.css` 的 `:root` 块（第 3-115 行）目前同时承担两个角色：

### 角色 A：共享设计令牌（与主题无关，应留在 `:root`）
| 分类 | 变量 | 行号 |
|------|------|------|
| 圆角 | `--radius-sm` ~ `--radius-2xl` | 29-33 |
| 字体 | `--font-family`, `--font-size-base`, `--font-heading` ~ `--font-mono` | 36-43 |
| 间距 | `--space-1` ~ `--space-12` | 46-53 |
| 过渡 | `--transition` | 56 |
| 图标尺寸 | `--icon-sm` ~ `--icon-lg` | 59-61 |
| 动画 | `--anim-duration-fast` ~ `--anim-easing-in` | 64-69 |
| 主题切换过渡 | `--theme-transition` | 72-75 |

### 角色 B：默认主题语义值（应搬至 `[data-theme="default"]`）
| 分类 | 变量 | 行号 |
|------|------|------|
| 配色 | `--bg` ~ `--input-bg` | 5-20 |
| 阴影 | `--shadow-sm` ~ `--shadow-xl` | 23-26 |
| 主题系统变量 | `--accent-rgb` ~ `--text-tertiary` | 78-93 |
| 语义色 | `--success` ~ `--info-text` | 96-108 |
| 分层阴影 | `--shadow-elevated` ~ `--shadow-toast` | 111-114 |

### 关键依赖关系
- `--selection-bg`（第 13 行）使用 `rgba(var(--accent-rgb), ...)`，`--accent-rgb` 将移至 `[data-theme="default"]`，所以 `--selection-bg` 也必须从 `:root` 移除
- `--theme-transition`（第 72-75 行）使用 `var(--anim-duration-slow)`，两者都在 `:root` 中保留，没问题
- `--font-heading` 等使用 `var(--font-family)`，都在 `:root` 中保留，没问题

## 修改方案

### 涉及文件
- 仅 `frontend/src/css/variables.css` 一个文件

### 具体变更

#### 变更 1：精简 `:root` 块
- 保留：圆角、字体、间距、过渡、图标尺寸、动画、主题切换过渡（角色 A）
- 删除：配色、阴影、`--accent-rgb`、`--selection-bg`、主题系统变量、语义色、分层阴影（角色 B）
- 删除第 1 行注释 `/* 设计系统变量 — Jot 温暖极简主题 */`，改为 `/* 设计系统共享令牌 */`

#### 变更 2：新建 `[data-theme="default"]` 块
- 在 `:root` 块之后、`[data-theme="light"]` 之前插入
- 内容为原 `:root` 中删除的角色 B 所有变量
- 注释标记：`/* ===== 默认主题（温暖极简） ===== */`

### 为什么可行

1. **CSS 层叠**：`[data-theme="default"]` 特异性高于 `:root`，当 `<html data-theme="default">` 时，主题变量从 `[data-theme="default"]` 取值，共享令牌从 `:root` 继承
2. **所有主题块已定义完整变量**：每个 `[data-theme="..."]` 块已包含所有主题色变量，剥离后 `:root` 的角色 B 定义不再被需要
3. **初始化流程安全**：
   - `index.html` 内联脚本在 CSS 加载前从 `localStorage` 读取主题并设置 `data-theme`，`default` 是有效值
   - 关键色内联脚本注入 `:root{--bg:...;--topbar-bg:...}` 作为防闪烁保护，独立于主题系统
   - `getCurrentTheme()` 回退值为 `'default'`，`applyTheme()` 总是主动设置 `data-theme` 属性

## 验证步骤

1. 重新构建/运行应用
2. 检查默认主题（`data-theme="default"`）下所有 UI 元素颜色是否正确
3. 切换其他主题（light、dark、nord、tokyo-night 等），确认正常
4. 重新启动应用，确认主题初始化无闪烁
5. 检查浏览器控制台无 CSS 变量相关警告

## 风险与注意事项

- **无破坏性变更**：`data-theme` 属性值不变，JS 逻辑不变，其他 CSS 文件不变
- **唯一风险**：若 `<html>` 上无 `data-theme` 属性，`:root` 中不再有 `--bg` 等变量，`var(--bg)` 会回退到 `initial`。但实际代码中 `getCurrentTheme()` 有回退，`applyTheme()` 保证属性被设置，内联关键色脚本覆盖初始渲染，风险极低