# Tasks

- [ ] Task 1: 新建 `format.js`，迁移格式化操作项与 SQL 辅助函数
  - 导入 js-beautify、js-yaml、smol-toml、sql-formatter 及 `../formatters/` 下三个格式化模块
  - 迁移 18 个格式化操作项（JSON/XML/HTML/CSS/JS/SQL/CSV/YAML/TOML）
  - 迁移 `compactSQL`、`convertSQLCase` 辅助函数
  - 默认导出 `FORMAT_ACTIONS` 数组

- [ ] Task 2: 新建 `text-transform.js`，迁移文本转换操作项
  - 迁移 7 项（大写、小写、首字母大写、驼峰式、蛇形式、行反转、字符反转）
  - 默认导出 `TEXT_TRANSFORM_ACTIONS` 数组

- [ ] Task 3: 新建 `text-clean.js`，迁移文本清理操作项
  - 迁移 5 项（去除多余空格、去除空行、行尾空格清理、Tab 转空格、空格转 Tab）
  - 默认导出 `TEXT_CLEAN_ACTIONS` 数组

- [ ] Task 4: 新建 `encode-decode.js`，迁移编码解码操作项
  - 迁移 6 项（Base64 编/解码、URL 编/解码、HTML 编/解码）
  - 默认导出 `ENCODE_DECODE_ACTIONS` 数组

- [ ] Task 5: 精简 `editor-actions.js`
  - 删除已迁移的格式化/文本转换/文本清理/编码解码操作项定义
  - 删除 `compactSQL`、`convertSQLCase` 辅助函数
  - 导入四个新模块并在 `EDITOR_ACTIONS` 末尾展开
  - 保持渲染与执行引擎逻辑不变

# Task Dependencies

- Task 1~4 相互独立，可并行
- [Task 5] 依赖于 [Task 1][Task 2][Task 3][Task 4]