# 统一修复内部滚动型视图的"底栏"遮挡（padding-bottom 残留）

## 问题背景

全局规则 `.view { padding: 24px 32px }`（`frontend/src/css/components/main-content.css#L14-L18`）为所有视图提供窗口四边留白。其中 6 个视图采用"内部滚动容器"模式：

```css
#mainContent:has(#viewX.active) { scrollbar-gutter: auto; overflow-y: hidden; }
```

对这些视图，`.view` 的 `padding-bottom: 24px` 会把内部滚动容器的裁切线抬高到窗口底缘上方 24px，下方露出一条 `--bg` 色空白带——内容滚动时像被一条"底栏"遮挡（浏览器实测：裁切线 645.2px，窗口高 669px，差值 23.8px）。

此前待办清单与密码管理已按以下模式修复（`padding-bottom: 0` + 注释），数据管理、设置本次用户已确认存在同款问题。

## 排查结论（全部 6 个内部滚动型视图）

| 视图                       | padding 现状                                          | 结论                                                      |
| ------------------------ | --------------------------------------------------- | ------------------------------------------------------- |
| viewData 数据管理            | `padding: 24px 0 24px 32px`                         | **需修**（用户已确认信笺被遮挡）                                      |
| viewSettings 设置          | `padding: 24px 0 24px 32px`                         | **需修**（用户已确认展开预设下拉时出现）                                  |
| viewCalendar 日历          | `padding: 24px 0 24px 32px`                         | **需修**（排查确认：右侧 `.calendar-notes-list` 内部滚动，同款裁切线 + 空白带） |
| viewTodo 待办              | `#viewTodo.active { padding-bottom: 0 }`            | 已修（todo.css#L551-L555），不动                               |
| viewPasswordManager 密码管理 | `#viewPasswordManager.active { padding-bottom: 0 }` | 已修（password-manager.css#L163-L166），不动                   |
| viewAiChat AI 聊天         | `#viewAiChat.view { padding: 0 }`（全清）               | 无问题（ai-chat.css#L2480-L2482），不动                         |

其余视图（viewGrid 笔记首页、viewTrash 回收站、viewEditor 编辑器）由 `#mainContent` 直接滚动或全高布局，`padding-bottom` 是可滚到的正常收尾留白，**不属于问题，不修**。

修复后底部呼吸感由各内部容器自身 padding 提供（已确认均存在）：

* `.data-panels { padding: 20px 24px }` → 底部 20px

* `.settings-panel { padding: 20px 24px }` → 底部 20px

* `.calendar-notes-panel { padding: 20px 0 }` → 底部 20px

## 修改内容（3 个文件，各加 1 条规则）

修复方式与既有 todo/pm 模式完全一致（新增 `.active` 覆盖规则 + 同款注释），不改动现有 `padding: 24px 0 24px 32px` 行。

### 1. `frontend/src/css/components/data-view.css`

在 `#mainContent:has(#viewData.active)` 规则（L13-L16）之后新增：

```css
/* 面板滚动区贴到窗口底缘：抵消 .view 的 padding-bottom（底部空白带像底栏遮挡信笺），
   底部呼吸感由 .data-panels 自身的 padding-bottom 提供 */
#viewData.active {
    padding-bottom: 0;
}
```

### 2. `frontend/src/css/components/settings-panel.css`

在 `#mainContent:has(#viewSettings.active)` 规则（L118-L121）之后新增：

```css
/* 面板滚动区贴到窗口底缘：抵消 .view 的 padding-bottom（底部空白带像底栏遮挡内容），
   底部呼吸感由 .settings-panel 自身的 padding-bottom 提供 */
#viewSettings.active {
    padding-bottom: 0;
}
```

### 3. `frontend/src/css/components/calendar.css`

在 `#mainContent:has(#viewCalendar.active)` 规则（L10-L13）之后新增：

```css
/* 日历面板贴到窗口底缘：抵消 .view 的 padding-bottom（底部空白带像底栏遮挡），
   底部呼吸感由 .calendar-sidebar / .calendar-notes-panel 自身的 padding-bottom 提供 */
#viewCalendar.active {
    padding-bottom: 0;
}
```

## 假设与决策

* **选择新增** **`.active`** **规则而非直接改 padding 简写**：与 todo.css / password-manager.css 的既有修复模式、注释风格完全一致，diff 最小、语义清晰。

* **不统一重构**：6 个视图 3 种现状（已修/全清/待修），本次只补齐漏掉的 3 个，不做全局 `.view` 基线改动（会牵连笔记首页等直接滚动视图的留白）。

* **内部容器 padding 不调整**：修复后底部留白 20px（原为 20+24=44px），与待办/密码管理修复后的观感标准一致，无需补偿。

## 验证步骤

1. `npm run dev` 启动 Vite（后台），浏览器打开页面（Wails API 缺失的报错忽略）。
2. 用 browser\_evaluate 依次激活 `#viewData` / `#viewSettings` / `#viewCalendar`，测量：

   * `.data-panels` / `.settings-panels` / `.calendar-notes-panel` 的 `getBoundingClientRect().bottom` 应 ≈ `window.innerHeight`（差值 < 1px，修复前为 23.8px）；

   * 窗口最底部 `elementFromPoint` 采样应命中内部滚动容器而非 `DIV.view.active`。
3. 数据管理页信笺滚动中途，落款下方不再有假底栏空白带（裁切线贴窗口底缘）。
4. 设置页展开预设下拉滚动到底，下拉下方无空白带遮挡。
5. 目视确认日历页左右面板贴底、无底部空白带；待办/密码管理/AI 聊天/笔记首页不回归。

