# 拆分 main.js — 提取独立静态模块 Spec

## Why

`frontend/src/main.js`（约 5840 行）过于庞大。将其中完全无状态依赖的常量、工具函数、通知类、模拟数据提取到独立文件，小幅降低复杂度。

## 安全分析 — 为什么不会影响使用

| 提取内容 | 风险 | 原因 |
|---------|------|------|
| `SVGS` 图标常量 | **零风险** | 纯 const 对象，无运行时行为。使用 `export` + `import { SVGS }` 方式引入，main.js 中所有 `SVGS.xxx` 引用**写法不变** |
| 4 个工具函数 | **零风险** | 纯函数（输入→输出，无副作用），不访问 `state`/`els`/DOM。main.js 中调用写法不变 |
| `NotificationManager` 类 | **零风险** | 纯类定义，不访问外部变量（内部用 `window.SVGS` 取图标）。main.js 中 `new NotificationManager()` 写法不变 |
| 模拟数据函数 | **零风险** | 纯函数，返回静态数组。main.js 调用写法不变 |

**`resetPagination` 不拆分** — 它访问了 `currentPage`、`totalNotes` 等 main.js 局部变量，留在原处。

## What Changes

新建 2 个模块文件：

| 新文件 | 内容 | 行数 | 导出方式 |
|--------|------|------|---------|
| `js/constants.js` | `SVGS` + `formatTime` + `highlightText` + `getSummary` + `debounce` | ~90 行 | `export` named + `window.xx` 供其他模块 |
| `js/notification.js` | `NotificationManager` 类 + `mockNotes`/`getMockNotes`/`getMockTags` | ~150 行 | `export` named + `window.xx` 供其他模块 |

迁移策略：
- 使用 **ES module named export/import** — main.js 中所有引用写法保持不变
- 同时挂载到 `window` — 供 `data-management.js`、`trash-page.js` 等其他模块通过 `window.SVGS` 访问
- `notification.js` 中 `SVGS` 引用改为 `window.SVGS`（通过 import 顺序保证 `constants.js` 先加载）
- 不修改 HTML/CSS

## Impact

- Affected code:
  - `frontend/src/main.js` — 删除 ~240 行，添加 2 行 named import
  - `frontend/src/js/constants.js` — **新文件**，~90 行
  - `frontend/src/js/notification.js` — **新文件**，~150 行
- No breaking changes

## ADDED Requirements

### Requirement: constants.js

```js
export const SVGS = { ... };           // 全部 ~30 个 SVG 图标
export function formatTime(isoString) { ... }
export function highlightText(text, keyword) { ... }
export function getSummary(text, maxLen = 100) { ... }
export function debounce(fn, delay) { ... }

// 同时挂载到 window，供其他模块使用
window.SVGS = SVGS;
window.formatTime = formatTime;
// ...
```

### Requirement: notification.js

```js
export class NotificationManager { ... }   // 内部通过 window.SVGS 引用图标
export function getMockNotes() { ... }
export function getMockTags() { ... }

window.NotificationManager = NotificationManager;
window.getMockNotes = getMockNotes;
window.getMockTags = getMockTags;
```

### Requirement: main.js 变更

删除约 240 行（29-78、80-178、504-555、3622-3671），顶部添加：

```js
import { SVGS, formatTime, highlightText, getSummary, debounce } from './js/constants.js';
import { NotificationManager, getMockNotes, getMockTags } from './js/notification.js';
```

其余代码**一字不改** — 所有 `SVGS.*`、`formatTime()` 等引用方式完全不变。
