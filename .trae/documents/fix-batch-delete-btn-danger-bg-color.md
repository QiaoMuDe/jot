# 修复批量删除按钮视觉状态问题

## 摘要

批量删除按钮有两个设计缺陷：①背景色过浅（浅色主题 `#FEF2F2` 几乎白色）；②**无状态区分** — 未选中任何笔记时和已选中时的按钮视觉完全一样，缺少"可点/不可点"的状态提示。此外 `border` 声明不完整存在 CSS 特异性冲突。

本次重构按钮的完整交互状态体系：未选中→禁用态，已选中→危险突出态，hover→加深，active→按压反馈。

---

## 修改方案

### 修改 1：`frontend/src/main.js` — `updateBatchBar()` 添加 `.has-selection` class 切换

在 `updateBatchBar()` 末尾添加（文件 ~5064 行）：

```javascript
// 更新删除按钮状态（无选中时禁用，有选中时激活）
if (els.batchDeleteBtn) {
    els.batchDeleteBtn.classList.toggle('has-selection', count > 0);
}
```

这是整个方案的交互基础 — 无选中时按钮表现为禁用态，选中后激活为危险按钮。

### 修改 2：`frontend/src/css/components/main-content.css` — 重写 `.batch-btn.btn-danger` 状态机

替换现有 `.batch-btn.btn-danger` 三个选择器（~453-461 行）为完整状态体系：

```css
/* ── 基础：未选中 → 禁用态 ── */
.batch-btn.btn-danger {
  background: transparent;
  color: var(--text-muted);
  border: 1px solid var(--border);
  cursor: not-allowed;
  opacity: 0.5;
}

/* ── 已选中 → 危险突出态 ── */
.batch-btn.btn-danger.has-selection {
  background: var(--danger-bg);
  color: var(--danger);
  border: 1px solid var(--danger-border);
  opacity: 1;
  cursor: pointer;
}

/* ── hover：背景变为实色危险色，文字变白 ── */
.batch-btn.btn-danger.has-selection:hover {
  background: var(--danger);
  color: #fff;
  border-color: var(--danger);
  box-shadow: 0 2px 8px rgba(220, 38, 38, 0.2);
}

/* ── active（按下）：缩小 + 压暗 ── */
.batch-btn.btn-danger.has-selection:active {
  transform: scale(0.96);
  filter: brightness(0.85);
  box-shadow: none;
}
```

设计理由：
- **无选中态**：`opacity: 0.5` + `text-muted` + `cursor: not-allowed`，明确传达"不可操作"
- **已选态**：`var(--danger-bg)` 浅粉底 + `var(--danger)` 红字 + `var(--danger-border)` 红框，清晰传达危险操作
- **hover**：实色 `var(--danger)` 底 + 白字 + 红色阴影，强烈的"即将执行危险操作"信号
- **active**：`scale(0.96)` 物理按压感 + `brightness(0.85)` 加深

### 修改 3：`frontend/src/css/variables.css` — 全面增强 `--danger-bg` 可见度

所有主题的 `--danger-bg` 提升，确保 `.has-selection` 状态的背景色肉眼可辨：

| 主题 | 行号 | 当前值 | 改为 | 说明 |
|------|------|--------|------|------|
| default | 83 | `#fef2f2` | `#FEE2E2` | Tailwind red-100 浅粉 |
| light | 144 | `#FEF2F2` | `#FEE2E2` | 同上 |
| nord | 260 | `#FEF2F2` | `#FEE2E2` | 同上 |
| eye-protection | 376 | `#FEF2F2` | `#FEE2E2` | 同上 |
| catppuccin-latte | 428 | `#FEF2F2` | `#FEE2E2` | 同上 |
| gruvbox-light | 480 | `#FEF2F2` | `#FEE2E2` | 同上 |
| alice | 745 | `#FDF2F2` | `#FDE2E2` | 微暖粉 |
| lightmind | 799 | `#F6E9E7` | `#F5D8D6` | 更明显的淡粉 |
| quiet-light | 638 | `rgba(220,38,38,0.08)` | `rgba(220,38,38,0.18)` | 不透明度翻倍 |
| ysgrifennwr | 691 | `rgba(199,71,75,0.08)` | `rgba(199,71,75,0.18)` | 不透明度翻倍 |
| dark | 202 | `rgba(248,113,113,0.1)` | `rgba(248,113,113,0.2)` | 不透明度翻倍 |
| tokyo-night | 318 | `rgba(247,118,142,0.1)` | `rgba(247,118,142,0.2)` | 不透明度翻倍 |
| dracula | 532 | `rgba(255,85,85,0.1)` | `rgba(255,85,85,0.2)` | 不透明度翻倍 |
| one-dark-pro | 585 | `rgba(224,108,117,0.1)` | `rgba(224,108,117,0.2)` | 不透明度翻倍 |

---

## 不修改的部分

- `settings-panel.css` 中的 `.btn-danger`（未使用 `--danger-bg`，不受影响）
- `calalog` 等按钮的 border 声明

---

## 交互状态一览

```
┌──────────────────────────────────────────────────────────┐
│  状态           │  视觉表现                              │
├──────────────────────────────────────────────────────────┤
│  无选中 (idle)  │  半透明灰字、灰边框、not-allowed 光标  │
│  有选中 (ready) │  粉底红字红框、正常光标                │
│  hover  (ready) │  实色红底白字、红色阴影                │
│  active (ready) │  缩小 0.96 + 压暗 0.85，无阴影         │
└──────────────────────────────────────────────────────────┘
```

## 验证

1. 进入批量模式 → 批量删除按钮呈**半透明灰色禁用态**
2. 选中 1+ 个笔记 → 按钮变为**粉底红字红框**激活态
3. hover 激活态按钮 → **红底白字** + 阴影
4. 按下按钮 → **缩小 + 压暗**反馈
5. 取消全选回到 0 个 → 按钮**回到禁用态**
6. 切换所有 14 个主题验证颜色正确
