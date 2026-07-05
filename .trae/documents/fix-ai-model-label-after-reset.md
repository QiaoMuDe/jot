# 修复：重置出厂设置后 AI 模型标签未清空

## Summary

重置出厂设置后，设置页的 AI 模型标签仍显示上次选择的模型名，而不是应有的 `"-- 请先获取模型列表 --"`。

## Current State

`frontend/src/main.js` 中 `loadAISettings()` 函数（\~1654-1663 行）：

```js
let cfg;
try {
    cfg = await window.go.main.App.GetAIConfig();
    els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
    if (cfg.model) {
        els.aiModelLabel.textContent = cfg.model;
        addModelDropdownItem(cfg.model, true);
    }
} catch (e) {
    console.warn('loadAISettings: model loading error', e);
}
```

当 `cfg.model` 为空时，`if` 被跳过，label 保持旧值不变。

其他触发路径（切换服务商 \~1910 行、切换预设 \~2413 行）都正确地执行了 `els.aiModelLabel.textContent = '-- 请先获取模型列表 --'`，唯独 `loadAISettings()` 缺少这个重置逻辑。

## Proposed Changes

### 1. `frontend/src/main.js` — `loadAISettings()`

在 `if (cfg.model)` 之后添加 `else` 分支：

```js
if (cfg.model) {
    els.aiModelLabel.textContent = cfg.model;
    addModelDropdownItem(cfg.model, true);
} else {
    els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
}
```

同时，当 model 为空时，搜索框也应该隐藏（与切换服务商时一致）：

```js
} else {
    els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
    const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
    if (wrap) wrap.style.display = 'none';
}
```

## Assumptions & Decisions

* 只在 `loadAISettings()` 中修复。`reloadSettings()` 和进入设置页都会经过此函数，覆盖两种入口。

* label 重置文案 `"-- 请先获取模型列表 --"` 与切换服务商/预设时的文案保持一致。

## Verification

1. 确认 `frontend/src/main.js` 中 `loadAISettings()` 的 `if (cfg.model)` 后有 `else` 分支重置 label
2. 搜索框在无模型时隐藏

