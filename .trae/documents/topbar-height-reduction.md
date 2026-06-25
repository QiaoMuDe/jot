# 降低 topbar(自定义窗口标题栏)高度

## Summary

将 Jot 编辑器顶部自定义绘制窗口标题栏(`#topbar`)的固定高度从 `56px` 降低到 `48px`,同步调整所有依赖该高度值的元素(侧栏 header、编辑器全屏遮罩、编辑器全屏高度)。改动全部在 `frontend/src/style.css` 一文件内,4 处数值 + 1 处注释,5 处 CSS 规则。JS / HTML / Wails 配置均无需改动。

## Current State Analysis

### 当前结构
- **HTML 元素** [index.html:34-65](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html#L34-L65) — `<header id="topbar">` 包含 4 个子区域:left(☰ 菜单+品牌名)、search(搜索框)、actions(最小化/最大化/关闭)
- **CSS 高度** [style.css:9](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L9) — `#topbar { height: 56px; }`
- **内部最大元素** [style.css:99-115](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L99-L115) — `.topbar-btn { width: 34px; height: 34px; }`,搜索框 `padding: 8px 0` + 文字 0.813rem ≈ 26px,brand-name 1.063rem ≈ 17px
- 顶部 56px 给按钮上下各留 11px padding,内部空间充裕但确实偏大

### 依赖 56px 高度的 4 处(必须同步)

| 行号 | 规则 | 用途 |
|------|------|------|
| [L9](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L9) | `#topbar { height: 56px; }` | **主目标** |
| [L937](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L937) | `.editor-overlay.fullscreening { top: 56px; }` | 编辑器全屏时的遮罩层,需要紧贴 topbar 下方 |
| [L983](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L983) | `.editor-panel.fullscreen { height: calc(100vh - 56px); }` | 编辑器全屏高度,扣除 topbar 占用的视口 |
| [L3476](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L3476) | `.sidebar-header { height: 56px; }` | 侧栏顶部,注释明确「高度与 topbar 对齐」 |
| [L3469](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L3469) | 注释 `/* 侧栏头部 — 高度与 topbar 对齐（56px）*/` | 需同步更新注释 |

### 不需要改的地方

- [L1999](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L1999) `width: 56px` — 是**宽度**(侧栏收起后宽度),与 topbar 高度无关,只是数字巧合
- 内部元素(button 34px、search 26px、brand 17px)全部 < 48px,无需缩放
- HTML 结构无变化
- JS 无变化(Wails 拖拽区域、按钮事件、双击最大化等都已用 class/id 定位,不依赖高度值)

## Proposed Changes

### 单文件改动: `frontend/src/style.css`

| # | 行号 | 规则 | 改动 | 原因 |
|---|------|------|------|------|
| 1 | [L9](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L9) | `#topbar { height: 56px; }` | → `height: 48px;` | 主目标,降低 8px 节省顶部空间 |
| 2 | [L937](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L937) | `.editor-overlay.fullscreening { top: 56px; }` | → `top: 48px;` | 保持遮罩层紧贴新 topbar 下方,避免全屏时有 8px 空隙 |
| 3 | [L983](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L983) | `.editor-panel.fullscreen { height: calc(100vh - 56px); }` | → `height: calc(100vh - 48px);` | 编辑器全屏时,高度 = 视口 - topbar,需同步 |
| 4 | [L3476](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L3476) | `.sidebar-header { height: 56px; }` | → `height: 48px;` | 侧栏 header 与 topbar 视觉对齐,必须同步 |
| 5 | [L3469](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/style.css#L3469) | 注释 | → `/* 侧栏头部 — 高度与 topbar 对齐（48px）*/` | 注释保持与代码同步,避免后续维护困惑 |

## Assumptions & Decisions

- **保持 34px 按钮不变**:48px - 34px = 14px 上下总 padding,每个按钮上下各 7px,符合 Mac/Windows 11 现代应用标准,视觉平衡
- **不动搜索框和 brand 文字字号**:内部元素已 < 48px,字号调整会引发次生改动(content 区高度,字体加载等),收益小、风险大
- **不引入 CSS 变量**:虽然可以用 `--topbar-height: 48px` 集中管理,但本项目其他多处硬编码(56px 散落在 4 处),用变量需要批量改 4 处用法 + 4 处注释,改动面更大。本次仅做数值同步,后续如要再调,可考虑提取为变量
- **不修改 Wails `Frameless` 行为**:topbar 始终作为拖拽区域,改高度不影响拖拽交互
- **不修改响应式断点**:topbar 56→48 后,在 640px 以下的小屏视图仍然合理(搜索框会自动收窄),不需特殊处理

## Verification

1. **常规视图**:`wails dev` 启动,目视确认 topbar 高度明显降低,搜索框/按钮/品牌名垂直居中,无错位
2. **侧栏对齐**:检查侧栏 header 高度与 topbar 严格一致,分割线对齐
3. **全屏模式**:打开一条笔记 → `Ctrl+\` 或点全屏按钮,确认:
   - 编辑器面板紧贴 topbar 下方,无 8px 空隙
   - 遮罩层覆盖范围正确(top: 48px, height: 100vh - 48px)
4. **窗口拖拽**:鼠标按住 topbar 空白处拖动,确认窗口跟随移动;双击 topbar 空白处可最大化/还原
5. **窗口控制按钮**:最小化/最大化/关闭三个按钮可正常点击
6. **搜索框交互**:点击搜索框 focus,无错位;输入文字无变形
7. **侧栏展开/收起**:切换侧栏状态,sidebar-header 与 topbar 始终对齐
8. **多分辨率**:分别测试窗口在 800×600、1280×720、1920×1080 下的显示效果

## Files Changed

- `frontend/src/style.css` — 5 处(4 处数值 + 1 处注释)
