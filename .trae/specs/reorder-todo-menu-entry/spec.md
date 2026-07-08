# 待办清单菜单入口重新排序 Spec

## Why

待办清单目前位于「更多菜单」的最底部（AI 助手之后隔一个分隔线），作为常用功能入口位置过深，且没有快捷键。将其移到「数据管理 → 回收站」分组后，符合「组织管理工具」的语义分组，并分配 Ctrl+6 快捷键。

## What Changes

- **index.html**：将「待办清单」dropdown-item 从菜单末尾（AI 助手之后）移到「回收站」之后、「设置」之前；更新所有受影响菜单项的快捷键 title
- **main.js**：键盘导航 handler 中重新分配 Ctrl+6/7/8 快捷键
- **main.js**：renderShortcutsPage 中更新快捷键说明列表

## Impact

- 影响模块：更多菜单、键盘快捷键导航、快捷键说明弹窗
- 影响文件：
  - `frontend/index.html`
  - `frontend/src/main.js`

## 变更后菜单结构

```
数据管理       (Ctrl+4)
回收站         (Ctrl+5)
待办清单       (Ctrl+6)      ← 从末尾移到此位置
───────────
设置           (Ctrl+7)      ← 原 Ctrl+6
帮助参考                     ← 无快捷键，保持原位
───────────
AI 助手        (Ctrl+8)      ← 原 Ctrl+7
```

## ADDED Requirements

### Requirement: 待办清单快捷键
- **WHEN** 用户按 Ctrl+6
- **THEN** 切换视图到待办清单

### Requirement: 设置快捷键调整
- **WHEN** 用户按 Ctrl+7
- **THEN** 切换视图到设置页

### Requirement: AI 助手快捷键调整
- **WHEN** 用户按 Ctrl+8
- **THEN** 切换视图到 AI 助手

## MODIFIED Requirements

### Requirement: 快捷键说明弹窗
原 Ctrl+6/7 的说明更新为：
- Ctrl+6 → 待办清单
- Ctrl+7 → 设置
- Ctrl+8 → AI 助手
