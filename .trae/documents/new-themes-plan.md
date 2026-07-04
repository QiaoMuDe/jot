# 新增 6 个主题 — 实施计划

## 摘要

在现有 9 个主题基础上一口气新增 6 个精心挑选的主题（2 暗 4 亮），大幅丰富主题选择。同时清理 `main.go` 中 4 个已删除主题的残留代码。

## 最终确认的 6 个新主题

| # | 英文 key | 中文名 | 类型 | 风格灵感 |
|---|---------|--------|------|---------|
| 1 | `one-dark-pro` | 暗夜 | 暗色 | 暖蓝灰底，Atom 经典，专业通用 |
| 2 | `evergreen-forest` | 深林 | 暗色 | 冷绿底，自然系，与护眼配成绿系亮暗对 |
| 3 | `solarized-light` | 日光 | 亮色 | 暖黄底 + 青绿高亮，色彩科学 |
| 4 | `quiet-light` | 静谧 | 亮色 | 白底极简低对比，从容优雅 |
| 5 | `bluloco-light` | 碧空 | 亮色 | 蓝调清新，色彩丰富清晰 |
| 6 | `ysgrifennwr` | 暖笺 | 亮色 | 奶油暖白底，异国色彩，温暖独特 |

主题 key 命名规则延续现有 `kebab-case` 风格。

### 代码高亮配对方案

| 系统主题 | 推荐代码高亮主题 | 原因 |
|---------|----------------|------|
| `one-dark-pro` | `one-dark-pro` | 原生配对 |
| `evergreen-forest` | `gruvbox-dark` | 暖色高亮在冷绿底上更突出 |
| `solarized-light` | `github-light` | 简约高亮不冲突 |
| `quiet-light` | `vscode-light-plus` | 低对比 UI + 标准高亮 |
| `bluloco-light` | `github-light` | 蓝调 UI + 中性高亮 |
| `ysgrifennwr` | `github-light` | 暖白 UI + 中性高亮 |

## 修改清单

### 1. variables.css — 新增 6 个 `[data-theme]` 块
- **位置**：文件末尾（在 `eye-protection` 之后）
- **内容**：每个主题 49 个 CSS 变量（核心 20 + 语义 17 + 滚动条 3 + 标签 4 + 阴影 4）
- **工作量**：每个主题约 50 行，共约 300+ 行 CSS
- **注意事项**：
  - `--accent-rgb` 必须与 `--accent` 的 RGB 分量一致
  - `--selection-bg` 使用 `rgba(var(--accent-rgb), 0.30)`
  - 阴影颜色匹配主题的 `--text-primary` 或 `--bg` 色系

### 2. main.js — 注册新主题
- **位置**：`themeLabels` 对象（第 1301 行附近）新增 6 个条目
- **位置**：`codeHighlightThemePairing` 对象（第 1289 行）新增 6 个配对

### 3. index.html — 添加 UI 入口
- **位置**：`criticalColors` 对象新增 6 个条目（防闪色）
- **位置**：主题下拉菜单新增 6 个 `<div class="theme-select-item">`

### 4. main.go — 清理残留 + 新增映射
- **清理**：删除 `monokai-pro`、`catppuccin-mocha`、`gruvbox-dark`、`ayu-mirage` 的过时 case
- **新增**：添加 6 个新主题 + `eye-protection` 的背景色映射
- **位置**：`themeBG()` 函数（第 15-41 行）

## 每个主题的配色方案

### 1. one-dark-pro（暗夜）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #282C34 | Atom One Dark 经典深灰底 |
| --card-bg | #2C313A | 略亮于背景 |
| --text-primary | #ABB2BF | 柔和白色 |
| --text-secondary | #828997 | 灰 |
| --accent | #61AFEF | 亮蓝 |
| --accent-rgb | 97, 175, 239 | — |
| --success | #98C379 | 绿 |
| --warning | #E5C07B | 黄 |
| --error / --danger | #E06C75 | 红 |

### 2. evergreen-forest（深林）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #1E2E24 | 深墨绿底 |
| --card-bg | #263A2E | 略亮 |
| --text-primary | #D3C6AA | 米白 |
| --text-secondary | #A0B09A | 灰绿 |
| --accent | #A7C080 | 草绿 |
| --accent-rgb | 167, 192, 128 | — |
| --success | #A7C080 | 草绿 |
| --warning | #DBBC7F | 金黄 |
| --error / --danger | #E67E80 | 红 |

### 3. solarized-light（日光）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #FDF6E3 | 暖黄底 |
| --card-bg | #FEF9ED | 更浅 |
| --text-primary | #586E75 | 柔深青灰 |
| --text-secondary | #839496 | 青灰 |
| --accent | #268BD2 | 亮蓝 |
| --accent-rgb | 38, 139, 210 | — |
| --success | #859900 | 绿 |
| --warning | #B58900 | 黄 |
| --error / --danger | #DC322F | 红 |

### 4. quiet-light（静谧）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #F5F5F5 | 极浅灰白底 |
| --card-bg | #FFFFFF | 纯白卡片 |
| --text-primary | #333333 | 深灰 |
| --text-secondary | #777777 | 中灰 |
| --accent | #4477AA | 柔蓝 |
| --accent-rgb | 68, 119, 170 | — |
| --success | #669944 | 暗绿 |
| --warning | #BB8833 | 土黄 |
| --error / --danger | #CC4444 | 暗红 |

### 5. bluloco-light（碧空）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #F9F9FA | 冷白底 |
| --card-bg | #FFFFFF | 纯白 |
| --text-primary | #383A42 | 深灰 |
| --text-secondary | #9D9D9F | 灰 |
| --accent | #275FE4 | 亮蓝 |
| --accent-rgb | 39, 95, 228 | — |
| --success | #50A14F | 绿 |
| --warning | #C18401 | 金 |
| --error / --danger | #E45649 | 橙红 |

### 6. ysgrifennwr（暖笺）

| 变量 | 色值 | 说明 |
|------|------|------|
| --bg | #F5EDDA | 暖米白底 |
| --card-bg | #FAF4E3 | 更亮暖米 |
| --text-primary | #3A3532 | 暖深棕 |
| --text-secondary | #9C8B7A | 暖灰褐 |
| --accent | #C7474B | 砖红 |
| --accent-rgb | 199, 71, 75 | — |
| --success | #6A8E4A | 橄榄绿 |
| --warning | #B8922E | 芥末黄 |
| --error / --danger | #C7474B | 砖红 |

## 实施顺序

1. **variables.css** — 新增 6 个 `[data-theme]` 块
2. **main.js** — 注册 `themeLabels` 和 `codeHighlightThemePairing`
3. **index.html** — 添加 `criticalColors` 和下拉选项
4. **main.go** — 清理残留 + 新增映射
5. **验证** — 构建、切换、预览

## 验证步骤

1. `npm run build` 通过（零错误零警告）
2. 主题下拉菜单正确列出 9 原有 + 6 新增 = 15 个选项
3. 切换每个新主题，UI 配色正确
4. 切换主题时不闪白（criticalColors 生效）
5. 重启后主题持久化正确（localStorage / Go SettingService）
6. `main.go` 无死代码残留
