# 移除三个主题：山林、爱丽丝、暗夜

## Summary

移除三个系统主题：`lightmind`（山林）、`alice`（爱丽丝）、`one-dark-pro`（暗夜），涉及 JS 配置和 CSS 变量定义。

## 需要修改的文件

### 1. `frontend/src/js/theme-config.js`

删除以下三个映射：

**themeLabels** (第10、17-18行)：

* `'one-dark-pro': '暗夜'`

* `'alice': '爱丽丝'`

* `'lightmind': '山林'`

**codeHighlightThemePairing** (第23、35-36行)：

* `'one-dark-pro': 'one-dark-pro'`

* `'alice': 'github-light'`

* `'lightmind': 'monokai-dimmed'`

**isDarkTheme** (第46、53-54行)：

* `'one-dark-pro': true`

* `'alice': false`

* `'lightmind': false`

### 2. `frontend/src/css/variables.css`

删除以下三个主题的 CSS 变量块：

* `[data-theme="one-dark-pro"]` (约第566-617行)

* `[data-theme="alice"]` (约第726-777行)

* `[data-theme="lightmind"]` (约第780-831行)

## Verification

* 检查主题选择器中不再显示这三个主题

* 检查其他主题仍正常工作

