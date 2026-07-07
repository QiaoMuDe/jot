# 目录大纲偶现自动打开问题排查与修复

## 总结

**问题**：打开新笔记时，目录大纲（TOC）偶尔会自己打开，而根据设计需求默认应该是关闭的。

**根因**：`_initTocToggle()` 在启动时从 `localStorage('tocSidebarOpen')` 恢复 TOC 状态，当该值残留为 `'true'` 时，会在启动后自动展开 TOC。根据设计需求，TOC **不应持久化状态**，每次打开笔记默认为关闭，用户仅在当前会话中手动展开。

## 修改方案

### 变更：移除 TOC 相关的全部 localStorage 读写

**文件**：[main.js#L3548-L3574](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)

移除两处 localStorage 操作：

1. **`_initTocToggle()` 中的 localStorage 读取** — 不再从 `localStorage('tocSidebarOpen')` 恢复状态，启动时 TOC 始终关闭
2. **点击切换处理器中的 localStorage 写入** — 用户的展开/折叠操作仅影响当前会话的 class，不持久化

```javascript
function _initTocToggle() {
    if (!els.tocSidebar || !els.tocToggleBtn) return;
    // 不再从 localStorage 恢复状态 —— TOC 默认始终关闭
    
    els.tocToggleBtn.addEventListener('click', () => {
        const mdContent = els.mdRendered.textContent.trim();
        if (!mdContent) {
            nm.show('正文暂无内容，无法生成目录', 'info');
            return;
        }
        const hasHeadings = els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6').length > 0;
        if (!hasHeadings) {
            nm.show('当前文档未提取到标题', 'info');
            return;
        }
        const isOpen = els.tocSidebar.classList.toggle('toc-visible');
        els.tocToggleBtn.classList.toggle('active', isOpen);
        // 不再写入 localStorage
    });
}
```

`_closeToc()` 保持不变（仅移除 class，无需操作 localStorage）。

### 原理

- `_closeToc()` 已在合适的时机被调用（切换编辑模式、关闭编辑器、内容为空等），确保 TOC 在笔记切换时关闭
- 移除 localStorage 后，即使之前残留 `'true'` 值也不影响行为
- 用户在当前会话中手动展开 TOC 后，切换笔记/切换模式/关闭编辑器时 `_closeToc()` 自动将其关闭

## 验证步骤

1. 打开 Markdown 笔记 → 确认 TOC 默认关闭
2. 点击 TOC 按钮展开 → 切换到编辑模式 → TOC 收起
3. 切回预览模式 → 再次展开 TOC → 关闭编辑器
4. 重新打开笔记 → 确认 TOC 默认为关闭
5. 重启应用 → 打开笔记 → 确认 TOC 关闭（不受之前 localStorage 残留影响）
6. 非 Markdown 笔记 → 确认 TOC 始终关闭
