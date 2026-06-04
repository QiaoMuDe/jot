# 主题系统 Spec

## Why

当前应用只有一套固定的温暖配色，用户无法根据环境或个人偏好切换主题。通过提取全量 CSS 变量并设计主题系统，用户可以在设置页切换默认/浅色/深色三种主题，提升使用体验。

## What Changes

1. **CSS 变量提取与统一**：将 `style.css` 中所有硬编码颜色值替换为 CSS 变量，在 `app.css` 的 `:root` 中统一管理
2. **三套主题定义**：Default（当前温暖基调）、Light（明亮蓝调）、Dark（深色基调），通过 `[data-theme]` 属性覆盖 CSS 变量
3. **设置页新增主题切换**：在设置页「字体设置」下方新增「主题设置」分区，使用下拉菜单（类似字体族选择器）选择主题
4. **后端持久化**：通过已有 `SettingService` 持久化主题偏好（key: `theme`）
5. **启动加载**：应用启动时从后端读取已保存的主题并应用

## Impact

* Affected specs: 无，独立新增功能

* Affected code: `frontend/src/app.css`（CSS 变量重构 + 主题覆盖）、`frontend/src/style.css`（硬编码颜色替换）、`frontend/index.html`（新增主题下拉菜单 DOM）、`frontend/src/main.js`（主题切换逻辑）、`internal/services/setting_service.go`（复用现有 KV 存储）

***

## ADDED Requirements

### Requirement: CSS 变量重构

系统 SHALL 将所有硬编码颜色值提取为 CSS 自定义属性。

#### Scenario: 硬编码颜色迁移

* **GIVEN** 当前 `style.css` 中存在多处硬编码颜色值

* **WHEN** 主题系统被实现

* **THEN** 所有颜色值应通过 `var(--xxx)` 引用，而非直接写死

需提取的硬编码位置：

* `.btn-danger` / `.batch-btn.btn-danger` 系列：`#fef2f2` / `#fee2e2` / `#fecaca` → `--danger-bg` / `--danger-border`

* `.import-result.success`：`#ecfdf5` / `#065f46` / `#a7f3d0` → `--success-bg` / `--success-text` / `--success-border`

* `.import-result.error`：`#fef2f2` / `#b91c1c` / `#fecaca` → 复用 `--danger-*`

* `.btn-perm-delete`：`#fef2f2` / `#fee2e2` / `#fecaca` → 复用 `--danger-*`

* `.context-menu-item.danger:hover`：`#fef2f2` → 复用 `--danger-bg`

* `.tag-delete-btn` / `.tag-delete-btn:hover`：`rgba(0, 0, 0, 0.2)` / `rgba(0, 0, 0, 0.4)` → `--tag-delete-bg` / `--tag-delete-hover-bg`

* `.undo-toast`：`#2D2A24` / `#F7F5F0` → `--toast-bg` / `--toast-text`

* `.editor-overlay` / `.confirm-dialog-overlay`：`rgba(45, 42, 36, 0.4)` → `--overlay-bg`

* `.note-card.selected` shadow: `rgba(217, 119, 6, 0.15)` → 使用 `rgba(var(--accent-rgb), 0.15)`

* `.loading-spinner`：`#e0e0e0` / `#f97316` → `--border` / `--accent`

* `.segmented-control`：`#f0f0f0` / `#d9d9d9` fallback → `--input-bg` / `--border`

* `app.css` 滚动条颜色：`rgba(0, 0, 0, 0.08)` / `rgba(0, 0, 0, 0.18)` → `--scrollbar-thumb` / `--scrollbar-thumb-hover`

新增 CSS 变量：

* `--accent-rgb`：accent 颜色的 RGB 分量（用于 rgba 构造）

* `--danger-bg`：危险操作背景色

* `--danger-border`：危险操作边框色

* `--success-bg`：成功状态背景色

* `--success-border`：成功状态边框色

* `--success-text`：成功状态文字色

* `--overlay-bg`：蒙层背景色

* `--toast-bg`：Toast 背景色

* `--toast-text`：Toast 文字色

* `--scrollbar-thumb`：滚动条滑块色

* `--scrollbar-thumb-hover`：滚动条滑块 hover 色

* `--tag-delete-bg`：标签删除按钮背景

* `--tag-delete-hover-bg`：标签删除按钮 hover 背景

### Requirement: 三套主题定义

系统 SHALL 提供三套完整的配色主题。

#### Scenario: Default 主题

* **GIVEN** 用户首次使用应用

* **WHEN** 应用启动

* **THEN** 默认使用当前温暖基调配色

Default 主题变量（保持现有温暖基调配色）：

```css
[data-theme="default"] {
  --bg: #F7F5F0;
  --card-bg: #FFFFFF;
  --text-primary: #2D2A24;
  --text-secondary: #8B867C;
  --text-muted: #A39E96;
  --accent: #D97706;
  --accent-rgb: 217, 119, 6;
  --accent-light: #FDE68A;
  --accent-lighter: #FEF3C7;
  --accent-dark: #B45309;
  --border: #E5E0D8;
  --divider: #EBE6DE;
  --danger: #DC2626;
  --hover-bg: #F3F0EB;
  --topbar-bg: #FFFFFF;
  --input-bg: #F3F1ED;
  --overlay-bg: rgba(45, 42, 36, 0.4);
  --toast-bg: #2D2A24;
  --toast-text: #F7F5F0;
  --danger-bg: #fef2f2;
  --danger-border: #fecaca;
  --success-bg: #ecfdf5;
  --success-border: #a7f3d0;
  --success-text: #065f46;
  --scrollbar-thumb: rgba(0, 0, 0, 0.08);
  --scrollbar-thumb-hover: rgba(0, 0, 0, 0.18);
  --tag-delete-bg: rgba(0, 0, 0, 0.2);
  --tag-delete-hover-bg: rgba(0, 0, 0, 0.4);
}
```

#### Scenario: Light 主题 — 白色为主，极致清爽

* **GIVEN** 用户在设置页选择「浅色」主题

* **WHEN** 选择完成

* **THEN** 应用切换为以白色为基底的清爽蓝调

配色说明：背景使用极浅灰白 #FAFAFA，卡片和顶栏使用纯白 #FFFFFF，边框极淡 #EEEEEE，营造大量留白的清爽感。强调色用亮蓝 #2563EB。

```css
[data-theme="light"] {
  --bg: #FAFAFA;
  --card-bg: #FFFFFF;
  --text-primary: #1A1A1A;
  --text-secondary: #8C8C8C;
  --text-muted: #B0B0B0;
  --accent: #2563EB;
  --accent-rgb: 37, 99, 235;
  --accent-light: #DBEAFE;
  --accent-lighter: #EFF6FF;
  --accent-dark: #1D4ED8;
  --border: #EEEEEE;
  --divider: #F0F0F0;
  --danger: #DC2626;
  --hover-bg: #F5F5F5;
  --topbar-bg: #FFFFFF;
  --input-bg: #F5F5F5;
  --overlay-bg: rgba(0, 0, 0, 0.25);
  --toast-bg: #1A1A1A;
  --toast-text: #FFFFFF;
  --danger-bg: #FEF2F2;
  --danger-border: #FECACA;
  --success-bg: #F0FDF4;
  --success-border: #BBF7D0;
  --success-text: #166534;
  --scrollbar-thumb: rgba(0, 0, 0, 0.06);
  --scrollbar-thumb-hover: rgba(0, 0, 0, 0.14);
  --tag-delete-bg: rgba(0, 0, 0, 0.2);
  --tag-delete-hover-bg: rgba(0, 0, 0, 0.4);
}
```

#### Scenario: Dark 主题 — 纯黑为底，低蓝光护眼

* **GIVEN** 用户在设置页选择「深色」主题

* **WHEN** 选择完成

* **THEN** 应用切换为纯黑基底 + 浅灰卡片配色

配色说明：背景使用纯黑 #0D0D0D，卡片用 #1A1A1A，文本用冷白 #E8E8E8。避免纯黑导致对比过强，用深灰 #2A2A2A 做边框。强调色用暖琥珀 #F59E0B 在暗背景下保持可读性。

```css
[data-theme="dark"] {
  --bg: #0D0D0D;
  --card-bg: #1A1A1A;
  --text-primary: #E8E8E8;
  --text-secondary: #9E9E9E;
  --text-muted: #6B6B6B;
  --accent: #F59E0B;
  --accent-rgb: 245, 158, 11;
  --accent-light: #92400E;
  --accent-lighter: #78350F;
  --accent-dark: #FBBF24;
  --border: #2A2A2A;
  --divider: #2A2A2A;
  --danger: #F87171;
  --hover-bg: #252525;
  --topbar-bg: #141414;
  --input-bg: #252525;
  --overlay-bg: rgba(0, 0, 0, 0.7);
  --toast-bg: #2D2A24;
  --toast-text: #F7F5F0;
  --danger-bg: rgba(248, 113, 113, 0.1);
  --danger-border: rgba(248, 113, 113, 0.25);
  --success-bg: rgba(52, 211, 153, 0.1);
  --success-border: rgba(52, 211, 153, 0.25);
  --success-text: #6EE7B7;
  --scrollbar-thumb: rgba(255, 255, 255, 0.08);
  --scrollbar-thumb-hover: rgba(255, 255, 255, 0.16);
  --tag-delete-bg: rgba(255, 255, 255, 0.25);
  --tag-delete-hover-bg: rgba(255, 255, 255, 0.4);
}
```

### Requirement: 设置页主题切换 UI

系统 SHALL 在设置页提供主题切换下拉菜单。

#### Scenario: 主题选择器展示

* **GIVEN** 用户进入设置页

* **WHEN** 主题设置分区渲染

* **THEN** 应显示一个下拉菜单，包含三个选项：「默认」「浅色」「深色」

#### Scenario: 主题选择

* **GIVEN** 用户点击主题下拉菜单

* **WHEN** 选择一个主题

* **THEN** 应用立即切换为该主题，并将选择持久化到后端

### Requirement: 主题持久化

系统 SHALL 通过 SettingService 持久化用户的主题偏好。

#### Scenario: 保存主题

* **GIVEN** 用户选择了新主题

* **WHEN** 选择完成

* **THEN** 调用 `SetSetting('theme', 'light|dark|default')` 保存

#### Scenario: 加载主题

* **GIVEN** 应用启动

* **WHEN** 初始化设置

* **THEN** 调用 `GetSetting('theme')` 读取并应用已保存的主题，若无则使用 default

### Requirement: `:root` 主题分层结构

系统 SHALL 采用三层 CSS 变量结构：

1. `:root` 层：所有 CSS 变量的声明（保留 default 主题值作为 fallback）
2. `[data-theme="light"]` 层：仅覆盖颜色相关变量
3. `[data-theme="dark"]` 层：仅覆盖颜色相关变量

