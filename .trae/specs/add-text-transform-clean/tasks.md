# Tasks

- [x] Task 1: 在 `EDITOR_ACTIONS` 中新增「文本转换」分组（7 个操作项）
  - 大写: `text.toUpperCase()`
  - 小写: `text.toLowerCase()`
  - 首字母大写: 正则 `/\b\w/g` 匹配单词首字母转大写
  - 驼峰式: 先按空格/分隔符拆分，首单词小写，后续单词首字母大写
  - 蛇形式: 先按驼峰边界拆分，再 `_` 连接并转小写
  - 行反转: `lines.reverse().join('\n')`
  - 字符反转: `str.split('').reverse().join('')`

- [x] Task 2: 在 `EDITOR_ACTIONS` 中新增「文本清理」分组（5 个操作项）
  - 去除多余空格: `text.replace(/\s+/g, ' ').trim()`
  - 去除空行: `text.split('\n').filter(l => l.trim()).join('\n')`
  - 行尾空格清理: `text.split('\n').map(l => l.trimEnd()).join('\n')`
  - Tab 转空格: `text.replace(/\t/g, '  ')`
  - 空格转 Tab: `text.replace(/  /g, '\t')`

- [x] Task 3: 运行 `npm run build` 验证构建无错误

# Task Dependencies
- Task 1, Task 2 可并行
- Task 3 依赖 Task 1 + Task 2