# 修复 MCP 表单表单项排列错乱（label 消失 / 字段重叠）

## 摘要

MCP 服务器新增/编辑对话框里，stdio（本地进程）与 sse/http（远程）两种传输模式的输入组排列错乱：字段标题（label）消失、参数与环境变量重叠。根因是输入组用了 `display: grid` + 均分行高的 1fr 布局，各字段内容高度不均时，字段被压缩导致内部 label/输入框被 flex 收缩。修复方案：改用 `flex column + max-height` 实现平滑展开/收起，字段高度始终等于内容高度，杜绝压缩与重叠。

## 现状分析

表单输入组结构（index.html）：

```html
<div class="mcp-stdio-group" id="mcpServerStdioGroup">   <!-- 3 个 .mcp-form-field -->
<div class="mcp-url-group collapsed" id="mcpServerUrlGroup">  <!-- 2 个 .mcp-form-field -->
```

每个字段：

```html
<div class="mcp-form-field">   <!-- display:flex; flex-direction:column; gap:6px -->
  <label class="mcp-form-label">命令</label>
  <input/textarea class="settings-input">
</div>
```

当前输入组样式（settings-panel.css 1950-1985）：

```css
.mcp-stdio-group { display: grid; grid-template-rows: repeat(3, 1fr); row-gap: 14px; overflow: hidden; }
.mcp-url-group   { display: grid; grid-template-rows: repeat(2, 1fr); row-gap: 14px; overflow: hidden; }
.mcp-stdio-group.collapsed { grid-template-rows: repeat(3, 0fr); opacity: 0; margin-bottom: 0; }
.mcp-url-group.collapsed   { grid-template-rows: repeat(2, 0fr); opacity: 0; margin-bottom: 0; }
.mcp-stdio-group > *, .mcp-url-group > * { min-height: 0; }
```

### 根因

1. `grid-template-rows: repeat(N, 1fr)` 将 N 行高度**均分**。
2. grid item（`.mcp-form-field`）默认 `align-self: stretch`，高度被强制为均分行高。
3. 各字段内容高度不均（命令「label+单行输入框」约 66px；参数/环境变量「label+2 行 textarea」约 100px）。均分后，部分行高小于字段内容需求。
4. `.mcp-form-field` 是 `display: flex; flex-direction: column`，其子项（label、input/textarea）默认 `flex-shrink: 1`——当字段高度不足时，flex 容器会把 label / textarea **压缩**：label 收缩到 0 高度（表现为「标题没了，只剩输入框」），textarea 与 label 挤压重叠。
5. `min-height: 0`（为 0fr 折叠加的 hack）进一步允许字段压缩到 0。

即：grid 均分行高 + flex 子项收缩，双重压缩导致排列错乱。

## 修改方案

只改一个文件：`frontend/src/css/components/settings-panel.css`（输入组相关样式块 1950-1985）。

### 具体改动

将 grid 方案整体替换为 `flex column + max-height` 折叠动画：

```css
/* stdio / url 输入组：切换传输方式时平滑展开/收起（flex column + max-height 过渡 + 透明度）
   flex 布局下字段高度始终等于内容高度，不会压缩 label/输入框（grid 1fr 均分行高会压扁内容） */
.mcp-stdio-group,
.mcp-url-group {
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
  max-height: 400px;   /* 大于展开内容高度（stdio 组约 294px / url 组约 180px），留余量容纳 textarea 增长 */
  opacity: 1;
  transition: max-height 0.25s ease, opacity 0.2s ease, margin 0.25s ease;
}

/* 折叠状态：高度归零、淡出且不占间距 */
.mcp-stdio-group.collapsed,
.mcp-url-group.collapsed {
  max-height: 0;
  opacity: 0;
  margin-bottom: 0;
}

/* 子项禁止收缩，确保 label / 输入框始终完整显示 */
.mcp-stdio-group > *,
.mcp-url-group > * {
  flex-shrink: 0;
}
```

要点说明：

* `display: flex; flex-direction: column`：字段垂直排列，每个字段高度 = 自身内容高度，不参与均分，**不可能被压缩**。

* 子项 `flex-shrink: 0`：双保险，防止任何路径下 label/输入框被收缩。

* `max-height: 400px`：作为展开/收起动画的载体（0 ↔ 400px），足够覆盖两组展开高度；textarea 内容增长时也不会截断（组高度由内容撑开，max-height 只是上限）。

* 删除原 `min-height: 0` 规则（它是 grid 0fr 折叠专用，flex 布局下不需要且有害）。

* 保留 `.mcp-server-form-body` 的 margin 方案（折叠组 `margin-bottom: 0` 不残留间距）不变。

* reduced-motion 块（2048-2070）无需改动：其中已有 `.mcp-stdio-group/.mcp-url-group { transition: none; }`，继续生效。

### 不改动的内容

* HTML（index.html）：`collapsed` 类与 DOM 结构不变。

* JS（main.js）：`setMCPFormTransport()` / `updateMCPServerTransportGroups()` 的 class 切换逻辑不变（只是 CSS 载体从 grid 换成 flex）。

## 假设与决策

* **决策**：放弃 grid 0fr/1fr 折叠动画（WebView2 渲染不可靠、内容不均导致压缩），改用 flex + max-height。max-height 过渡在接近上限时速度略慢（ease 曲线），视觉上「先快后慢」，属可接受的标准做法。

* **假设**：stdio 组展开高度（约 294px）远小于 max-height 400px 上限，动画表现正常。

* 不改动传输方式下拉（theme-select），本次问题与其无关。

## 验证步骤

1. `cd frontend && npm run build` —— 构建通过。
2. `wails build` —— 重新编译 `build\bin\jot.exe`。
3. 重启应用，打开「设置 → MCP 服务器 → 添加/编辑」：

   * stdio 模式下：命令、参数、环境变量三个字段的标题与输入框均完整显示、无重叠、无压缩。

   * 切换到 sse/http：URL、请求头字段排列正常。

   * 来回切换：展开/收起平滑，无闪烁、无残留空隙。

