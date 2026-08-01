# 修复格式化操作问题

## 现状分析

### 1. SQL 压缩 — 不是真正的压缩
当前 handler 先 `sqlFormat` 格式化再去掉 `\n{3,}`，结果和"SQL 格式化"几乎一样，用户感觉"没反应"。

### 2. YAML 格式化 — 输出看起来像 JSON
YAML 格式化 handler 用 `yaml.dump(parsed, { indent: 2, lineWidth: 120, noRefs: true })`，输出确实是 YAML 格式。但 YAML 压缩 handler 用了 `flowLevel: 0`，输出是 flow 风格（类似 `{key: value}`），看起来像 JSON，用户可能测试了压缩误以为是格式化的问题。

### 3. TOML 压缩 — 和格式化完全一样
TOML 格式化和压缩的 handler 代码完全一样（`smolToml.parse` + `smolToml.stringify`），点击压缩无区别。

## 修改方案

### 修改 1：SQL 压缩 — 改为真正的压缩
将 handler 改为移除所有多余空白，压缩为紧凑单行。

**文件**：`frontend/src/js/editor-actions.js`，SQL 压缩 handler

### 修改 2：YAML 压缩 — 移除 `flowLevel: 0`
将 YAML 压缩的 `flowLevel: 0` 改为 `flowLevel: 1`（仅 1 层以上嵌套使用 flow 风格），保持 YAML 格式但更紧凑。YAML 格式化不变。

**文件**：`frontend/src/js/editor-actions.js`，YAML 压缩 handler

### 修改 3：TOML 压缩 — 改为真正的压缩
将 handler 改为 `smolToml.stringify` 后去除多余空白行，使之与格式化产生不同结果。

**文件**：`frontend/src/js/editor-actions.js`，TOML 压缩 handler

## 验证
- `npm run build` 无编译错误
- 测试 SQL 压缩：输入多行 SQL，输出应为紧凑单行
- 测试 YAML 格式化：输入 YAML，输出应为 block 风格 YAML
- 测试 YAML 压缩：输入 YAML，输出应为更紧凑的 YAML
- 测试 TOML 压缩：输入 TOML，输出应比格式化更紧凑