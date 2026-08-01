# Tasks

- [ ] Task 1: 修改 `executeAction` 函数，支持 `type: 'insert'` 模式
  - 新增 `actionType` 参数（默认 `'transform'`）
  - 当 `actionType === 'insert'` 且无选中文本时，`from = to = 光标位置`
  - 点击菜单项时传递对应 `action.type` 给 `executeAction`

- [ ] Task 2: 新建 `frontend/src/js/editor-actions/md-syntax.js`，导出 MD 语法操作项数组
  - 每个操作项设置 `type: 'insert'`
  - 按分组结构组织：行内样式、标题、列表、块元素、链接/媒体、表格、数学公式
  - 共约 20 项操作，每项实现 `handler` 处理选中包裹和空字符串模板插入
  - 默认导出 `MD_SYNTAX_ACTIONS` 数组

- [ ] Task 3: 在 `editor-actions.js` 中导入 MD 语法操作项并合并到 `EDITOR_ACTIONS`
  - 添加 `import { MD_SYNTAX_ACTIONS } from './editor-actions/md-syntax.js'`
  - 在 `EDITOR_ACTIONS` 定义末尾使用 `...MD_SYNTAX_ACTIONS` 展开

# Task Dependencies

- [Task 2] 依赖于 [Task 1]
- [Task 3] 依赖于 [Task 1] 和 [Task 2]