# Tasks

- [x] Task 1: 安装 npm 依赖: `js-yaml`、`smol-toml`、`sql-formatter`、`js-beautify`
  - 运行 `npm install js-yaml smol-toml sql-formatter js-beautify` 安装 4 个依赖包
  - 验证 `package.json` 和 `package-lock.json` 更新正确

- [x] Task 2: 新增零依赖格式化函数（XML/HTML/CSV）
  - 新增 `frontend/src/js/formatters/xml-formatter.js`：XML 格式化/压缩（基于 DOMParser 递归遍历节点）
  - 新增 `frontend/src/js/formatters/html-formatter.js`：HTML 格式化/压缩（基于 DOMParser + 自定义序列化器，保留 HTML 标签名大小写）
  - 新增 `frontend/src/js/formatters/csv-formatter.js`：CSV 格式化（列对齐，padEnd 按列宽对齐）
  - 每个模块导出 `format(text)` 和 `minify(text)` 两个函数（CSV 仅导出 `format`）

- [x] Task 3: 重构 `editor-actions.js` — 更新操作注册表与错误处理
  - 修改 `EDITOR_ACTIONS` 数组，分组从单一「格式化」拆分为：JSON、XML、HTML、CSS、JavaScript、SQL、CSV、YAML、TOML
  - 每个操作项新增 `errorLabel` 字段（如 `'JSON'`、`'XML'`、`'YAML'`）
  - XML/HTML/CSV 操作引用 Task 2 的 formatter 模块
  - CSS/JS 操作引用 `js-beautify`（`css_beautify` / `js_beautify`）
  - SQL 操作引用 `sql-formatter`（`format`）
  - YAML 操作引用 `js-yaml`（`load` / `dump`）
  - TOML 操作引用 `smol-toml`（`parse` / 自定义序列化）
  - 修改 `executeAction` 的 catch 块：使用 `e.message` 作为通知文案，回落为 `不是合法的 ${action.errorLabel}`

- [x] Task 4: 验证
  - 运行 `npm run build` 确认无编译错误
  - 编辑模式打开操作菜单，确认 9 个分组子菜单正确显示
  - 逐个测试 17 个操作项，确认格式化/压缩结果正确
  - 测试非法内容操作，确认通知文案显示对应的格式名称（如「不是合法的 XML」）
  - 确认 JSON 格式化/压缩等既有功能不受影响

# Task Dependencies
- Task 2 依赖 Task 1（无直接依赖，但建议先安装依赖）
- Task 3 依赖 Task 1 + Task 2（需要引用 formatter 模块和第三方库）
- Task 4 依赖 Task 1-3