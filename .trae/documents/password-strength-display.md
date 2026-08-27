# 密码强度显示（添加/编辑/详情页）

## Summary

在密码管理的「添加/编辑」对话框和「查看详情」对话框中，为密码字段增加实时强度显示。

## Current State

* **添加/编辑对话框** (`pmEditOverlay`)：密码是 `<input>` 元素，位于 `.pm-input-wrap` 容器内，当前无强度显示

* **查看详情对话框** (`pmDetailOverlay`)：密码是只读 `<span>` 元素，位于 `.pm-detail-value-with-actions` 容器内，当前无强度显示

* 密码生成器已有 4 圆点强度指示器（`.pm-gen-strength`），颜色方案：weak=红、fair=黄、good=绿、strong=accent

## Design

### 添加/编辑对话框

在密码输入框**下方**（`.pm-input-wrap` 之后）添加一行强度指示：

```
[密码输入框                    显示]
[●●●○ 良好]                     ← 新增
```

* 空密码时不显示

* 实时响应 `input` 事件，调用 `CheckPasswordStrength` 更新

* 使用 4 圆点 + 文字标签，复用生成器的颜色方案

### 查看详情对话框

在密码值行的**下方**新增一行强度信息：

```
密码   ••••••••••  [显示] [复制]
       ●●●○ 良好                ← 新增
```

* 在 `.pm-detail-overlay` 的密码行后面插入

* 页面打开时即调用 `CheckPasswordStrength` 显示强度

## Changes

### 1. `frontend/index.html`

**添加/编辑对话框**（约第2001行）：在 `.pm-input-wrap` 的 `</div>` 后、`.pm-field` 的 `</div>` 前，插入：

```html
<div id="pmEditPwdStrength" class="pm-pwd-strength" style="display:none"></div>
```

**查看详情对话框**（约第2045行）：在密码行 `.pm-detail-row` 后插入：

```html
<div class="pm-detail-row" id="pmDetailPwdStrengthRow" style="display:none">
    <span class="pm-detail-label"></span>
    <div id="pmDetailPwdStrength" class="pm-pwd-strength"></div>
</div>
```

### 2. `frontend/src/css/components/password-manager.css`

新增 `.pm-pwd-strength` 样式（在密码相关样式区域附近）：

```css
/* 密码强度指示（添加/编辑 + 详情页共用） */
.pm-pwd-strength {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
    font-size: 11px;
    color: var(--text-muted);
}
.pm-pwd-strength-dots {
    display: flex;
    gap: 2px;
}
.pm-pwd-strength-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--border);
    transition: background 0.2s;
}
.pm-pwd-strength-dot.filled.weak  { background: var(--danger, #ef4444); }
.pm-pwd-strength-dot.filled.fair  { background: #f59e0b; }
.pm-pwd-strength-dot.filled.good  { background: #22c55e; }
.pm-pwd-strength-dot.filled.strong { background: var(--accent); }
```

### 3. `frontend/src/js/password-manager.js`

新增防抖工具函数和通用渲染函数：

```js
/** 防抖：delay ms 内重复调用只执行最后一次 */
function pmDebounce(fn, delay) {
    let timer = null;
    return (...args) => { clearTimeout(timer); timer = setTimeout(() => fn(...args), delay); };
}

/** 渲染密码强度指示（圆点 + 文字标签） */
function pmRenderStrength(el, score) {
    if (score < 0 || el === null) return;
    const labels = ['弱', '一般', '良好', '强', '强'];
    const classes = ['weak', 'fair', 'good', 'strong', 'strong'];
    el.innerHTML = `<span class="pm-pwd-strength-dots">${
        Array.from({length: 4}, (_, i) =>
            `<span class="pm-pwd-strength-dot${i < score ? ' filled ' + classes[score] : ''}"></span>`
        ).join('')
    }</span><span>${labels[score]}</span>`;
}
```

**添加/编辑对话框**：

* 绑定 `pmFieldPassword` 的 `input` 事件，使用 **debounce 300ms** 防抖后调用 `CheckPasswordStrength` + `pmRenderStrength`

* 密码为空时隐藏强度指示（直接隐藏，不调用后端）

**查看详情对话框**：

* `openPmDetail` 中获取密码记录后，调用 `CheckPasswordStrength` + `pmRenderStrength` 显示强度

* 掩码状态下也显示强度（不暴露密码内容）

## Files to Modify

| 文件                                                 | 改动                              |
| -------------------------------------------------- | ------------------------------- |
| `frontend/index.html`                              | 两个对话框各插入一个强度容器 DOM              |
| `frontend/src/css/components/password-manager.css` | 新增 `.pm-pwd-strength` 样式        |
| `frontend/src/js/password-manager.js`              | 新增 `pmRenderStrength` 函数 + 绑定事件 |

## Verification

1. 打开添加对话框，输入密码，观察强度实时变化
2. 纯数字密码应显示"弱"或"一般"，混合密码显示"良好"或"强"
3. 打开详情对话框，确认密码强度正常显示
4. 空密码时强度指示不显示

