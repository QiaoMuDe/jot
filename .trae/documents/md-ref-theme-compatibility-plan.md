# MD 语法页面主题适配 — 执行报告

## 根因

MD 语法页面的代码块（`pre`）使用**硬编码深色背景** `#1e293b / #e2e8f0`，不跟随主题切换。而编辑器预览区的代码块用 `var(--bg-secondary)` / `var(--text-primary)` 正常跟随主题。

## 改动清单

### [style.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css) — 3 处修复

| 选择器 | 改前（硬编码） | 改后（主题变量） |
|--------|---------------|-----------------|
| `.md-ref-source-panel pre` | `background: #1e293b; color: #e2e8f0;` | `var(--bg-secondary); var(--text-primary)` + 新增 `border: 1px solid var(--border)` |
| `.md-ref-preview pre` | `background: #1e293b; color: #e2e8f0;` | `var(--bg-secondary); var(--text-primary)` + 新增 `border: 1px solid var(--border)` |
| `.md-ref-copy-btn` | `rgba(255,255,255,0.08)` / `rgba(255,255,255,0.15)` / `#94A3B8` | `var(--card-bg)` / `var(--border)` / `var(--text-muted)` → hover: `var(--hover-bg)` / `var(--text-primary)` |
| `.md-ref-card` | `transition` 缺少 bc/bd | 新增 `background-color 0.2s ease, border-color 0.2s ease` |

## 效果

- 切换 6 套主题，**所有组件（卡片、标签、面板、代码块、按钮）** 跟随主题配色
- 代码块由深色固定变为跟随 `--bg-secondary`，配合 `--text-primary` 保证可读性
- 复制按钮由 `rgba(255)` 改为 `--card-bg` / `--border`，在任何主题下都可见

## 验证

`npm run build` 通过。
