# AI 模型下拉菜单滚动条修复方案

## 当前问题

`.ai-chat-model-dropdown`（AI 助手页面的模型选择下拉菜单）的滚动条不显示。已经尝试了 `overflow-y: scroll`、`scrollbar-gutter: stable`、`::-webkit-scrollbar` 自定义样式等多种方案，均无效。

## 根因分析

通过对比项目中其他下拉菜单组件的滚动条实现，发现以下关键差异：

| 组件 | `::-webkit-scrollbar` | `scrollbar-width` | `scrollbar-color` | 滚动条状态 |
|------|----------------------|-------------------|-------------------|-----------|
| `.font-family-options` | 仅隐藏 button | `thin` | `var(--scrollbar-thumb) transparent` | ✅ 正常 |
| `.sidebar-notebook-list` | 仅隐藏 button | `thin` | `var(--scrollbar-thumb) transparent` | ✅ 正常 |
| `.search-modal-results` | 仅隐藏 button | `thin` | `var(--scrollbar-thumb) transparent` | ✅ 正常 |
| `.batch-tag-body` | 仅隐藏 button | `thin` | `var(--scrollbar-thumb) transparent` | ✅ 正常 |
| `.shortcuts-body` | 仅隐藏 button | `thin` | `var(--scrollbar-thumb) transparent` | ✅ 正常 |
| `.ai-chat-model-dropdown` | 完整自定义(8px) | ❌ 无 | ❌ 无 | ❌ 不显示 |
| `.theme-select-dropdown` | 完整自定义(8px) | ❌ 无 | ❌ 无 | 用户反馈不显示 |

**结论**：所有滚动条正常工作的组件都**同时**设置了 `scrollbar-width: thin` + `scrollbar-color: ...` 这两个标准 CSS 属性。而缺失这两个属性的组件（包括 AI 模型下拉和设置页主题/服务商下拉）在 WebView2 上滚动条不正常。

## 方案：添加缺失的标准滚动条属性

### 修改文件

**`frontend/src/css/components/ai-chat.css`** — 修改 `.ai-chat-model-dropdown`：

1. 移除 `scrollbar-gutter: stable`
2. 移除全部 `::-webkit-scrollbar` 伪元素自定义样式
3. 改用和 `.font-family-options` 相同的模式：`scrollbar-width: thin` + `scrollbar-color: var(--scrollbar-thumb) transparent` + 仅隐藏 `::-webkit-scrollbar-button`

这种模式在项目中 6 个组件上已验证正常工作。

### 修改后样式

```css
.ai-chat-model-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    margin-bottom: 4px;
    min-width: 160px;
    max-height: 240px;
    overflow-y: auto;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 4px 12px var(--shadow);
    display: none;
    z-index: 20;
    scrollbar-width: thin;
    scrollbar-color: var(--scrollbar-thumb) transparent;
}

.ai-chat-model-dropdown.open {
    display: block;
}

.ai-chat-model-dropdown::-webkit-scrollbar-button {
    width: 0 !important;
    height: 0 !important;
    display: none !important;
}
```

### 同步修复 `.theme-select-dropdown`

同样的缺失问题也存在于设置页的 `.theme-select-dropdown`。但考虑到 `.theme-select-dropdown` 用户未反馈滚动条问题（可能模型较少未触发），作为预防性修复一起处理。

## 验证步骤

1. 构建项目：`go build ./...`
2. 运行，切换到有大量模型的 AI 服务商，触发模型下拉列表溢出
3. 确认滚动条可见且样式正确（6px 细条，`--scrollbar-thumb` 颜色）
4. 切换服务商验证设置页的下拉也同样正常
