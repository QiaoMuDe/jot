# 修复确认弹窗按钮不跟随主题色彩的问题

## 概述

保存确认弹窗（新建/编辑笔记时未保存就关闭）中的"不保存"和"确定"按钮使用固定语义色（红、黄），不跟随当前主题的 `--accent` 变化，导致与整体 UI 风格割裂。

## 当前状态分析

### 弹窗按钮结构

```html
<!-- index.html:1822-1838 -->
<div class="confirm-actions">
    <button class="confirm-btn confirm-cancel" id="confirmCancelBtn">取消</button>
    <button class="confirm-btn confirm-third" id="confirmThirdBtn">不保存</button>
    <button class="confirm-btn confirm-ok" id="confirmOkBtn">确定</button>
</div>
```

### 当前 CSS（modals.css）

| 按钮 | 类名 | 当前颜色 | 说明 |
|------|------|---------|------|
| 取消 | `.confirm-cancel` | `var(--card-bg)` + `var(--text-secondary)` | 中性色，合理 |
| 不保存 | `.confirm-third` | `var(--warning)`（琥珀色 #f59e0b） | **固定色，不跟随主题** |
| 确定 | `.confirm-ok` | `var(--danger)`（红色 #ef4444） | **固定色，不跟随主题** |

### JS 逻辑

- `showSaveConfirmDialog()`（main.js:1170）: 显示三方按钮，`确定`=保存、`不保存`=放弃
- `showConfirmDialog()`（main.js:1083）: 隐藏三方按钮，`确定`=确认

### 问题根因

`.confirm-ok` 和 `.confirm-third` 使用语义色值（`--danger`/`--warning`），但这些变量在各主题中值固定（红/黄），**不随主题 `--accent` 变化**。在暖色/冷色/暗色主题中始终显示为红色和黄色。

## 修改方案

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\modals.css`

### 改动 1：确定按钮 → 使用主题色

`.confirm-ok` 从 `var(--danger)` 改为 `var(--accent)`，让"确定"按钮使用当前主题的主色调。

```css
.confirm-ok {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
.confirm-ok:hover {
  background: var(--accent);
  opacity: 0.9;
  border-color: var(--accent);
}
```

### 改动 2：不保存按钮 → 使用主题色

`.confirm-third` 从 `var(--warning)` 改为 `var(--accent)`，使用主题色但降低饱和度以区分于确定按钮。

有两种方案可选：

**方案 A（推荐）：统一使用 accent**
两者都使用 `var(--accent)`，但通过透明度/变体区分主次。

| 按钮 | 样式 | 理由 |
|------|------|------|
| 确定（保存） | `background: var(--accent)` 实心 | 主要操作，视觉权重最高 |
| 不保存（放弃） | `background: transparent` + `color: var(--accent)` + `border-color: var(--accent)` 线框风格 | 次要操作，降低视觉权重但仍跟随主题 |

**方案 B：不保存用 danger**
保留"确定=accent"、"不保存=danger"的语义区分。

选择方案 A，因为：
1. 用户明确要求"跟随主题色彩变色"
2. 两个按钮都跟随主题，风格统一
3. 通过实心/线框区分主次操作，视觉层次清晰

## 验证

1. 切换不同主题（浅色/深色/冷色/暖色），确认两个按钮颜色跟随 `--accent` 变化
2. 确认 `showConfirmDialog`（普通确认框）中"确定"按钮颜色也跟随主题
3. 确认 hover 状态过渡平滑
4. 确认"取消"按钮不受影响
