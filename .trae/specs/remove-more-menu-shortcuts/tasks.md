# Tasks

- [x] Task 1: 从 `frontend/index.html` 移除更多菜单项的 `title` 属性
  - Step 1: 移除笔记首页的 `title="Ctrl+1 笔记首页"`
  - Step 2: 移除展开侧栏的 `title="Ctrl+2 展开侧栏"`（HTML 静态部分）
  - Step 3: 移除批量管理的 `title="Ctrl+3 批量管理"`
  - Step 4: 移除数据管理的 `title="Ctrl+4 数据管理"`
  - Step 5: 移除回收站的 `title="Ctrl+5 回收站"`
  - Step 6: 移除待办清单的 `title="Ctrl+6 待办清单"`
  - Step 7: 移除设置的 `title="Ctrl+7 设置"`
  - Step 8: 移除 AI 助手的 `title="Ctrl+8 AI 助手"`

- [x] Task 2: 从 `frontend/src/main.js` 移除 Ctrl+1~8 全局快捷键处理
  - Step 1: 移除 `switch (e.key)` 中 case '1' ~ case '8' 的整个分支代码（lines ~5756-5792）
  - Step 2: 保留 Ctrl+0 锁屏分支（case '0'），整理成独立的 `if` 块
  - Step 3: 移除相关注释

- [x] Task 3: 从 `frontend/src/main.js` 的 `renderShortcutsPage()` 中移除 Ctrl+1~8 条目
  - Step 1: 移除 `{ key: 'Ctrl + 1', desc: '笔记首页' }` ~ `{ key: 'Ctrl + 8', desc: 'AI 助手' }` 共 8 条（lines ~6036-6043）

- [x] Task 4: 从 `frontend/src/main.js` 的 `updateSidebarMenuItem()` 中移除动态 title 设置
  - Step 1: 移除 `menuItem.title = \`Ctrl+2 ${label}\`;` 行（line ~6652）

# Task Dependencies

- 所有任务无依赖关系，可并行执行
