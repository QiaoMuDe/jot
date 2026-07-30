# 按钮加载态宽度抖动修复 — spinner 绝对定位

## Summary

将 `setBtnLoading` 的 spinner 从 `prepend` + 改文字改为 `position: absolute` + 保持文字不变，消除按钮加载态宽度变化导致的输入框挤占抖动。

## 当前问题

`setBtnLoading(btn, true, '测试中…')` 在按钮内 prepend 一个 14px spinner + 把文字改为 `测试中…`，相比常态 `测试`（2 字）多出约 20px 宽度。按钮在 flex 容器中膨胀，挤占左侧输入框的空间，加载结束再还原，产生视觉抖动。

## 改动方案

### 1. `settings-panel.css` — 调整 `.btn-loading` 和 `.btn-spinner`

```css
/* ── 旧 ── */
.btn-loading {
  position: relative;
  pointer-events: none;
  display: inline-flex !important;
  align-items: center;
  justify-content: center;
  gap: 6px;
  animation: btnPulse 1.2s ease-in-out infinite;
}
.btn-loading .btn-spinner {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  animation: btnSpin 0.8s linear infinite;
}

/* ── 新 ── */
.btn-loading {
  position: relative;
  pointer-events: none;
  display: inline-flex !important;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding-left: 22px;   /* 14px spinner + 左右 padding 留位 */
  animation: btnPulse 1.2s ease-in-out infinite;
}
.btn-loading .btn-spinner {
  position: absolute;
  left: 6px;
  top: 50%;
  transform: translateY(-50%);
  width: 14px;
  height: 14px;
  animation: btnSpin 0.8s linear infinite;
}
```

### 2. `main.js` — 修改 `setBtnLoading` loading 分支

```js
// ── 旧 ──
if (loading) {
    btn.dataset.origText = btn.textContent;
    btn.classList.add('btn-loading');
    btn.disabled = true;
    const spinner = document.createElementNS(...);
    spinner.setAttribute('class', 'btn-spinner');
    ...
    btn.prepend(spinner);
    btn.childNodes[1].textContent = label || '处理中…';
}

// ── 新 ──
if (loading) {
    btn.classList.add('btn-loading');
    btn.disabled = true;
    // spinner 用 position:absolute，不参与按钮宽度计算
    if (!btn.querySelector('.btn-spinner')) {
        const spinner = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        spinner.setAttribute('class', 'btn-spinner');
        spinner.setAttribute('viewBox', '0 0 24 24');
        spinner.setAttribute('fill', 'none');
        spinner.setAttribute('aria-hidden', 'true');
        spinner.innerHTML = '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" opacity="0.3"/>' +
            '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" stroke-dashoffset="-10" opacity="0.85"/>';
        btn.insertAdjacentElement('afterbegin', spinner);
    }
}
```

恢复分支也简化：

```js
// ── 旧 ──
} else {
    btn.classList.remove('btn-loading');
    btn.disabled = false;
    const spinner = btn.querySelector('.btn-spinner');
    if (spinner) spinner.remove();
    if (btn.dataset.origText) btn.textContent = btn.dataset.origText;
    delete btn.dataset.origText;
}

// ── 新 ──
} else {
    btn.classList.remove('btn-loading');
    btn.disabled = false;
    const spinner = btn.querySelector('.btn-spinner');
    if (spinner) spinner.remove();
}
```

不再修改 `btn.textContent`，按钮文字全程不变。

### 3. 调用方 `label` 参数保持不变

所有 `setBtnLoading(btn, true, '测试中…')` 调用的 `label` 参数不再被使用，保留不动，避免不必要的改动。

## 效果

- 常态按钮宽度 = 加载态按钮宽度
- 所有用 `setBtnLoading` 的按钮统一改善（测试、获取模型、Tavily/知乎测试连接）
- API 连接页的输入框、预设弹窗的输入框在按钮加载时不再被挤占

## 验证

1. 肉眼确认 `<button>` 在加载态时宽度不变
2. 点击「测试」→ spinner 出现在按钮文字左侧，不改变文字内容
3. 加载结束 → spinner 移除，按钮恢复常态
