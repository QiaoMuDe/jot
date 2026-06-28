# Tasks

- [x] Task 1: 创建 `frontend/src/js/data-management.js` 文件，包含所有数据管理函数
  - 从 `main.js` 复制以下函数定义到新文件：
    - `animateCountUp`（~1880-1895）
    - `loadDataStats`（~1900-1949）
    - `resetDatabase`（~1954-1992）
    - `vacuumDatabase`（~1997-2010）
    - `openDataDir`（~2015-2026）
    - `exportData`（~2031-2045）
    - `importData`（~2050-2070）
    - `loadBackupInfo`（~2075-2092）
    - `backupToDir`（~2097-2120）
    - `restoreFromDir`（~2125-2156）
  - 所有函数通过 `export` 暴露（ES module 方式）
  - 函数内部通过 `const { ... } = window;` 在函数体内解构引用外部依赖

- [x] Task 2: 从 `main.js` 中移除迁移的函数定义
  - 删除 `/* ===== 数据管理函数 ===== */` 注释块及其下的所有函数定义（1872-2157 行）
  - 保留 `switchView` 中 `case 'data': loadDataStats()` 的调用不变
  - 保留事件绑定中对数据管理函数的调用不变
  - 在底部添加了 `window.xxx` 导出供 data-management.js 使用

- [x] Task 3: 通过 ES module `import` 引入 data-management.js
  - 在 main.js 顶部添加 `import { ... } from './js/data-management.js'`
  - 无需修改 index.html（ES module 自动处理依赖加载顺序）

# Task Dependencies

- [Task 1] 必须在 [Task 2] 和 [Task 3] 之前完成 ✅
- [Task 2] 和 [Task 3] 可并行执行 ✅（实际串行执行但相互独立）
