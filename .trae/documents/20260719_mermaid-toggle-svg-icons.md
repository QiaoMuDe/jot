# Mermaid 切换按钮添加 SVG 图标

## 当前状态

Mermaid 切换按钮（渲染/源码）当前只有纯文字，而复制按钮有 `SVGS.copy` 图标 + 文字。目标是统一风格，给切换按钮也加上图标。

## 改动方案

### 1. constants.js — 新增两个 Lucide 风格图标

在 `SVGS` 对象末尾添加两个图标：

| 图标名       | 用途     | 图形描述                |
| --------- | ------ | ------------------- |
| `diagram` | "渲染"按钮 | 三个圆点 + 连线构成的流程图/网络图 |
| `code`    | "源码"按钮 | 尖括号 `<>` 代码符号       |

两者均保持现有风格：24x24 viewBox、1.5px stroke、currentColor。

### 2. main.js — `setupMermaidBlock()` 按钮创建

```js
// 现有：toggleBtn.textContent = '渲染';
// 改为：
toggleBtn.innerHTML = SVGS.diagram + ' 渲染';
```

### 3. main.js — `toggleMermaidView()` 切换时更新

```js
// 显示源码视图时：
btn.innerHTML = SVGS.diagram + ' 渲染';

// 显示渲染视图时：
btn.innerHTML = SVGS.code + ' 源码';
```

### 4. editor.css — 无变动（按钮本身用 `inline-flex` + `gap: 4px` 对齐，与复制按钮一致）

现有 CSS `.mermaid-toggle` 使用 `display: inline-flex`、`gap: 4px`，SVG 图标(16px) 放入后自然与文字对齐，无需额外样式调整。

## 验证

1. `vite build` 构建成功
2. 笔记和 AI 消息中的 Mermaid 按钮均显示图标 + 文字
3. 切换视图后图标随文字更新（渲染→源码，源码→渲染）

