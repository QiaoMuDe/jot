# Tasks

- [x] Task 1: 提取 `js/constants.js`
  - 创建 `frontend/src/js/constants.js`
  - 移入：`SVGS` 常量（原 29-78 行）— `export const SVGS = { ... }`
  - 移入：`formatTime`、`highlightText`、`getSummary`、`debounce`（原 504-555 行）— `export function ...`
  - 末尾挂载到 window：`window.SVGS = SVGS; window.formatTime = formatTime; ...`
  - **`resetPagination` 不拆**（留在 main.js）
  - 修改 `main.js`：删除原定义，顶部加 `import { SVGS, formatTime, highlightText, getSummary, debounce } from './js/constants.js';`

- [x] Task 2: 提取 `js/notification.js`
  - 创建 `frontend/src/js/notification.js`
  - 移入：`NotificationManager` 类（原 80-178 行）— `export class NotificationManager { ... }`
    - 内部 `SVGS.checkmark` 等引用改为 `window.SVGS.checkmark`
  - 移入：`let mockNotes = null` + `getMockNotes` + `getMockTags`（原 3622-3671 行）— `export function ...`
  - 末尾挂载到 window
  - 修改 `main.js`：删除原定义，顶部加 `import { NotificationManager, getMockNotes, getMockTags } from './js/notification.js';`

## import 顺序

```js
import { SVGS, formatTime, highlightText, getSummary, debounce } from './js/constants.js';
import { NotificationManager, getMockNotes, getMockTags } from './js/notification.js';
```

## 验证

每个 Task 完成后运行应用，检查控制台无报错 + 对应功能正常。
