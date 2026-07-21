# 移动"待办清单"菜单项到 AI 助手分组

## Summary

将更多菜单中的"待办清单"从 Group 2（数据管理/回收站/待办清单/设置）移动到 Group 3（AI 助手），放在 AI 助手的上方，使它们处于同一个分组内。

## Current State

当前更多菜单 HTML 结构（[index.html#L71-L90](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html#L71-L90)）：

```
Group 1:  笔记首页 · 展开侧栏 · 批量管理
          ─── divider ───
Group 2:  数据管理 · 回收站 · 待办清单 · 设置
          ─── divider ───
Group 3:  AI 助手
          ─── divider ───
Group 4:  快捷键说明 · MD 语法 · 关于
```

## Proposed Changes

仅修改 `frontend/index.html` 第 77-85 行：

### 改动前（当前代码）
```html
<!-- Group 2: 数据管理 · 回收站 · 待办清单 · 设置 -->
<div class="dropdown-item" data-action="data">...</div>    <!-- 行76 -->
<div class="dropdown-item" data-action="trash">...</div>   <!-- 行77 -->
<div class="dropdown-item" data-action="todo">...待办清单</div>  <!-- 行78-82 -->
<div class="dropdown-item" data-action="settings">...</div>     <!-- 行83 -->
<div class="dropdown-divider"></div>                            <!-- 行84 -->
<!-- Group 3: AI 助手 -->
<div class="dropdown-item" data-action="ai-chat">...AI 助手</div>  <!-- 行85 -->
```

### 改动后
```html
<!-- Group 2: 数据管理 · 回收站 · 设置 -->
<div class="dropdown-item" data-action="data">...</div>
<div class="dropdown-item" data-action="trash">...</div>
<div class="dropdown-item" data-action="settings">...</div>
<div class="dropdown-divider"></div>
<!-- Group 3: 待办清单 · AI 助手 -->
<div class="dropdown-item" data-action="todo">...待办清单</div>
<div class="dropdown-item" data-action="ai-chat">...AI 助手</div>
```

具体操作：将第 78-82 行的 `<div class="dropdown-item" data-action="todo">...</div>` 从 divider 之前移到 divider 之后、AI 助手之前。

## Assumptions & Decisions

- 只改 HTML 顺序，不改 CSS 或其他文件
- 待办清单的图标和文案保持不变
- 不涉及 JS 逻辑变更（data-action 仍为 `todo`，已有的事件监听自动适配新位置）

## Verification

1. 确认 `frontend/index.html` 中待办清单位于 AI 助手之前、同一个 divider 组内
2. `npm run build` 构建无报错
3. 手动检查更多菜单渲染顺序符合预期
