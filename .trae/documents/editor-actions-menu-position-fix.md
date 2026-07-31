# 修复编辑器操作菜单弹出位置

## Summary

用户要求编辑器顶栏的「操作」下拉菜单**出现在操作按钮的左侧**（此前多轮修改后仍显示在按钮右侧/按钮正下方偏右，且子菜单弹出后被面板裁剪）。本计划通过**改变菜单的锚定包含块**（将按钮+菜单包进独立相对定位容器，用 `right` 锚定）从根上解决问题，同时补上 Wails 场景下前端资源更新不生效的缓存治理。

## Current State Analysis

### 现状几何分析（解释"为什么一直弹在右边"）

- 编辑器面板 `.editor-panel` 宽 560px（[editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L28-L38)）。
- 操作栏 `.editor-header-actions` 为 flex 容器、`justify-content: flex-end` 右对齐，宽约 248px（7 个 32px 按钮 + 6 个 4px gap），其左边缘约在面板 x≈300px 处（[editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L105-L110)）。
- 当前菜单 `.editor-actions-menu` 用 `left: 0` 相对**整个操作栏容器**锚定 → 菜单从 x≈300 向右展开 170px 到 x≈470，**菜单主体几乎全在按钮（x 300-332）右侧** → 用户描述为"还是出现在按钮右边"。
- 子菜单 `left: calc(100% + 4px)` 从根菜单右缘继续向右 → x≈474 到 624，**超出面板右边缘** → 这就是用户最初抱怨的"子菜单弹不出来"（[editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L200-L215)）。

### 正确目标布局

- 根菜单：右边缘贴按钮左边缘（菜单主体位于按钮左侧），x≈130-300，面板内 ✓
- 子菜单：从根菜单右缘向右弹出，x≈304-454，面板内 ✓

### 已确认的关键事实

1. 源码 CSS（`.editor-actions-menu.dropdown-menu { left: 0 }` 特异性 0,2,0）与 JS inline style（`menu.style.left='0'`）在代码层面均正确，但**锚定的是整个操作栏容器而非按钮本身**，几何上就注定"偏右"。
2. 本项目是 **Wails v2** 桌面应用：生产模式经 `go:embed all:frontend/dist` 在 **`wails build` 编译期**嵌入 dist（[main.go](file:///d:/资源池/下水道/Dev/本地项目/jot/main.go#L15-L16)）。只 `npm run build` 不重新 `wails build`，exe 内嵌的仍是旧资源 → 前端改动"看似不生效"。
3. `Rnx.toml` 的 `[task.frontend]` 有 `if [ ! -d "dist" ]` 守卫，dist 已存在时不会自动重建（[Rnx.toml](file:///d:/资源池/下水道/Dev/本地项目/jot/Rnx.toml#L76-L92)）。
4. `assetserver.Options` 确认支持 `Middleware`（`D:\AppData\gopath\pkg\mod\github.com\wailsapp\wails\v2@v2.12.0\pkg\options\assetserver\options.go` 第 35 行），可给入口页加 no-cache 头，根治 WebView2 缓存旧 `index.html` → 旧哈希资源的问题。

## Proposed Changes

### 1. [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L182-L184)：新增按钮+菜单的独立锚定包装器

**改什么**：把 `editorActionsBtn` 与 `editorActionsMenu` 包进一个 `<div class="editor-actions-wrap">`。

**为什么**：菜单必须相对**按钮本身**定位，而不是相对包含 7 个按钮的整个操作栏容器。包装器宽度恰好等于按钮宽度，使 `right`/`left` 锚定到按钮的几何位置。

**怎么改**：

```html
<div class="editor-header-actions">
    <div class="editor-actions-wrap">
        <button class="editor-header-btn" id="editorActionsBtn" title="操作">…SVG…</button>
        <div class="dropdown-menu editor-actions-menu" id="editorActionsMenu"></div>
    </div>
    <button class="editor-header-btn" id="tocToggleBtn" title="目录">…</button>
    …
</div>
```

### 2. [editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L138-L157)：菜单改为右缘锚定按钮左侧 + 新增包装器样式

**改什么 / 为什么**：
- 新增 `.editor-actions-wrap`：`position: relative`（成为菜单包含块）+ `display: inline-flex`（shrink-to-fit，宽度=按钮宽度，避免块级 div 撑满容器破坏布局）。
- `.editor-actions-menu.dropdown-menu`：`left` 改为 `auto`，`right` 改为 `calc(100% + 4px)`（菜单右缘 = 按钮左缘再左移 4px，形成 4px 间距），`transform-origin` 改为 `top right`（从右上角展开）。对 `left`/`right`/`transform-origin`/`top` 加 `!important`，抵御 `.dropdown-menu` 基类（`left: 8px`、`transform-origin: top right`，见 [topbar.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/topbar.css#L151-L165)）的干扰。

**怎么改**：

```css
/* 操作按钮+菜单锚定包装器 */
.editor-actions-wrap {
  position: relative;
  display: inline-flex;
}

.editor-actions-menu.dropdown-menu {
  position: absolute;
  top: calc(100% + 4px) !important;
  left: auto !important;
  right: calc(100% + 4px) !important;
  transform-origin: top right !important;
  /* 其余样式保持不变 */
}
```

子菜单规则（`left: calc(100% + 4px)`，[editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L200-L215)）**保持不变**——从根菜单右缘向右弹出，位于按钮正下方偏右，空间充足。

### 3. [editor-actions.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/editor-actions.js#L159-L166)：同步按钮点击时的内联定位

**改什么**：内联样式从 `left: 0; right: auto` 改为 `left: auto; right: calc(100% + 4px)`。

**为什么**：与 CSS 保持双保险一致，防止任何层叠问题导致回退到"按钮右侧"。

**怎么改**：

```js
menu.style.left = 'auto';
menu.style.right = 'calc(100% + 4px)';
```

### 4. [main.go](file:///d:/资源池/下水道/Dev/本地项目/jot/main.go#L68-L77)：AssetServer 增加入口页 no-cache 中间件

**改什么 / 为什么**：WebView2 会缓存无哈希的 `index.html`，导致其引用旧的哈希资源（旧 CSS/JS），前端改动即使重新构建也"不生效"。给入口页加 `no-cache` 头根治。

**怎么改**（Wails v2.12 的 `Middleware` 字段）：

```go
AssetServer: &assetserver.Options{
    Assets: assets,
    Middleware: func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 入口页禁用缓存，避免 WebView2 缓存旧资源引用导致前端修改不生效
            if r.URL.Path == "/" || r.URL.Path == "/index.html" {
                w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
            }
            next.ServeHTTP(w, r)
        })
    },
    Handler: …(现有 /images/ 处理保持不变)…,
},
```

### 5. 构建与运行（关键步骤，用户侧生效前提）

1. `cd frontend && npm run build`（重新生成 dist）
2. **重新构建 Go 二进制以嵌入新 dist**：`rnx --run run`（内部执行 `wails build -debug -clean` 后启动），或开发期用 `wails dev`（vite 热更新直接生效）
3. 不要直接运行 `build/bin/` 下的旧 exe（内嵌的是旧 dist）

## Assumptions & Decisions

- **"出现在左边"= 根菜单主体位于按钮左侧**（菜单右缘贴着按钮左缘，带 4px 间距），子菜单保持向右弹出。此解释与用户全部 4 轮反馈一致（"太靠右/弹不出来" → "希望出现在左边"），且能同时满足子菜单不溢出。
- 不对菜单做方向翻转逻辑（如空间不足向左弹），因为固定布局下按钮左/右两侧空间恒充足，保持简单。
- `!important` 仅加在菜单定位关键属性上，作为对多轮"样式不生效"的防御，不影响其他组件。
- 不删除之前创建的 `frontend/vite.config.js`（固定 dev 端口 5174，与 `wails.json` 的 `serverUrl: auto` 兼容，删掉反而可能让端口漂移）。

## Verification

1. `npm run build` 通过，无编译错误。
2. `main.go` 通过 `go build ./...` 或 `golangci-lint run ./...` 静态检查。
3. 应用内手动验证（用户侧）：
   - 新建/编辑笔记 → 顶栏最左侧「操作」按钮（代码图标）点击 → **根菜单出现在按钮左侧**（右缘贴按钮左缘），完全在面板内。
   - 悬停「格式化 ▶」→ 子菜单向右弹出，完整可见、不被裁剪。
   - 点击「JSON 格式化 / JSON 压缩」→ 对选中文本或全文生效，Ctrl+Z 可撤销。
   - 查看模式下操作按钮隐藏。
4. 若用户此前用旧 exe 验证过：确认本次通过 `rnx --run run`（重新 wails build）启动，而非直接跑旧 exe。
