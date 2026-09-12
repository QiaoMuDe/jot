# 系统主题增加与维护权威文档

本文档是本项目**主题系统的唯一权威参考**。新增、修改或删除任何系统主题时，优先参照本文件执行，以 `variables.css` 中的色值为准。

## 一、主题系统的组成与数据流

系统主题由 **CSS 变量 + `data-theme` 属性** 驱动；选择菜单由配置数据**自动生成**，不需要手动维护 HTML。

启动/切换链路（3 处背景色来源必须保持数值一致，否则用户会看到"窗口底色 → 页面背景"的闪烁）：

| 环节 | 文件 | 作用 |
| --- | --- | --- |
| ① 页面背景 | `variables.css` 的 `--bg` | **唯一权威背景色**，所有颜色以此为准 |
| ② 首帧防闪 | `index.html` 头部内联 `criticalColors` | 完整 CSS 加载前注入 `:root{--bg;--topbar-bg}` |
| ③ 窗口底色 | `main.go` 的 `themeBG()` | WebView2 窗口创建前的初始背景 RGBA |

命名约定：主题 key 使用**小写英文连字符**（如 `tokyo-night`），作为 `data-theme`、配置 key、localStorage 值的统一标识。

## 二、维护涉及的文件

| 文件 | 角色 |
| --- | --- |
| `frontend/src/css/variables.css` | 每个主题的色值定义（唯一权威背景色来源） |
| `frontend/src/js/theme-config.js` | 3 个映射：中文名 / 代码高亮配对 / Mermaid 明暗 |
| `frontend/index.html` | 头部内联 `criticalColors` 首帧配色 |
| `main.go` | `themeBG()` 窗口背景 RGB |

> **不要修改 `frontend/src/main.js`** —— 主题下拉菜单由 `buildThemeDropdown()` / `buildCodeHighlightThemeDropdown()` 依据 `themeLabels` 自动生成，`applyTheme()` 依据 `isDarkTheme` 决定 Mermaid 明暗。

## 三、新增一个主题

假设新增主题 key 为 `my-theme`，需依次完成 4 处改动：

### 1. `frontend/src/css/variables.css` 新增变量块

新增一个完整的 `[data-theme="my-theme"] { ... }`，**复制并按需改写一个相近主题的整个块**（配色、阴影、主题系统变量、语义色、分层阴影都要有），参照 `nord` / `dracula` 等已有块。注意：

- `--bg` 为本次新增主题的权威背景色，是唯一需与 ②③ 两处同步计算的色值；
- `--accent-rgb` 需给定 `r, g, b` 三个数（供 `--selection-bg` 等 `rgba()` 复用）；
- `--tip-think` / `--tip-generate` 消息统计条占比色需按该主题单独取值。

### 2. `frontend/src/js/theme-config.js` 补充 3 个映射（缺一不可）

| 映射 | 说明 | 示例 |
| --- | --- | --- |
| `themeLabels['my-theme']` | 下拉菜单中显示的中文名 | `'迷彩'` |
| `codeHighlightThemePairing['my-theme']` | 推荐搭配的代码高亮主题名 | `'github-dark'` |
| `isDarkTheme['my-theme']` | Mermaid 图用暗色(`true`)还是亮色(`false`)主题 | 暗色主题必须填 `true`，否则 Mermaid 渲染成亮色与新主题不协调 |

> **易漏项**：`isDarkTheme` 常被忽略。新增暗色主题（背景深色）务必填 `true`。

### 3. `frontend/index.html` 在 `criticalColors` 加首帧配色

```
'my-theme': ['#1A1B26', '#1F2335'],   // [背景色, 顶栏色]
```

- 第一位必须等于 `variables.css` 的 `--bg`；
- 第二位必须等于 `--topbar-bg`；
- 二者不一致会引发启动瞬间色差闪烁。

### 4. `main.go` 在 `themeBG()` switch 加 RGB 分支

```go
case "my-theme":
    return 26, 27, 38
```

- 该值与 `--bg` 一致（将十六进制 `#rrggbb` 转成十进制 `r, g, b`）。

### 5. 验证

- 启动后主题出现在设置的下拉菜单中；
- 切换无闪色；
- Mermaid/代码高亮明暗与新主题协调。

## 四、修改已有主题

只改颜色时，优先定位到 `variables.css` 中对应 `[data-theme="..."]` 块，**若只改 `--bg` / `--accent` 等不影响窗口防闪和背景**：仅需同步 `variables.css` 一处即可；**若改动的是 `--bg` / `--topbar-bg`**（背景类），必须同步更新 ② `index.html criticalColors` 与 ③ `main.go themeBG()`，保持三处数值一致。

## 五、删除一个主题

删除时需清理**同样四处**的残留，避免旧 key 悬挂：

1. `variables.css`：删除对应 `[data-theme="..."]` 整个块；
2. `theme-config.js`：删除 `themeLabels` / `codeHighlightThemePairing` /（如存在）`isDarkTheme` 中的对应 key；
3. `index.html`：删除 `criticalColors` 中的对应条目；
4. `main.go`：删除 `themeBG()` 中对应 `case`（若删的是 `default` 主题，需把其它分支的兜底值调整合理）。

历史案例：移除 `one-dark-pro` 时曾漏清理 `index.html` 的 `criticalColors` 与 `main.go` 的 `themeBG`，造成悬挂残留。

## 六、现有主题清单基线

下表为当前各系统主题的权威 `--bg`（取 `variables.css`）。**新增/修改主题后，应同时确保本章「背景色」列与 `index.html` `criticalColors`[0]、`main.go` `themeBG()` 返回值一致**，保持三处同步。

| 主题 key | 中文名 | 明暗 | 背景色（`--bg`） |
| --- | --- | --- | --- |
| `default` | 默认 | 亮 | `#F2EDE3` |
| `light` | 浅色 | 亮 | `#F4F4F7` |
| `dark` | 深色 | 暗 | `#18181C` |
| `nord` | 北极 | 亮 | `#E4E9F0` |
| `tokyo-night` | 夜幕 | 暗 | `#1A1B26` |
| `catppuccin-latte` | 暖咖 | 亮 | `#EDE7E5` |
| `gruvbox-light` | 旧纸 | 亮 | `#F0E8C9` |
| `dracula` | 德古拉 | 暗 | `#232530` |
| `eye-protection` | 护眼 | 亮 | `#E4ECD9` |
| `quiet-light` | 静谧 | 亮 | `#F0EAEC` |
| `ysgrifennwr` | 暖笺 | 亮 | `#F0E5D0` |

> 说明：上表以 `variables.css` 的 `--bg` 为准。若 `criticalColors` / `main.go` 与权威值存在历史偏差，视为待修复项，需按前述流程对齐。