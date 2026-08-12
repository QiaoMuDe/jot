# 全局快捷键 macOS 双兼容（Ctrl/Cmd）计划

## Summary

按方案 A 将 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 中 11 处纯 `e.ctrlKey` 判断的全局快捷键改为 `(e.ctrlKey || e.metaKey)` 双兼容，使 macOS 用户按 Cmd+字母 即可触发，Windows 行为零变化。同时同步 1 处配套的编辑器放行逻辑（L6591），避免 mac 上编辑器内 Cmd+Home/End 行为不一致。

## Current State Analysis

[handleKeyboardNavigation](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6303) 中 11 处全局快捷键为纯 `e.ctrlKey` 判断，macOS 上按 Cmd+字母 无效：

| # | 快捷键 | 位置 | 现条件 |
|---|---|---|---|
| 1 | Ctrl+S 编辑器保存 | [L6321](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6321) | `e.ctrlKey && (e.key === 's' \|\| 'S')` |
| 2 | Ctrl+F 搜索 | [L6330](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6330) | `e.ctrlKey && e.key === 'f'` |
| 3 | Ctrl+H 查找替换 | [L6354](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6354) | `e.ctrlKey && e.key === 'h'` |
| 4 | Ctrl+N 新建笔记 | [L6363](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6363) | `e.ctrlKey && e.key === 'n'` |
| 5 | Ctrl+L 编辑/预览切换 | [L6372](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6372) | `e.ctrlKey && (e.key === 'l' \|\| 'L') && ...` |
| 6 | Ctrl+J AI 侧栏 | [L6380](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6380) | `e.ctrlKey && (e.key === 'j' \|\| 'J') && ...` |
| 7 | Ctrl+E 编辑器全屏 | [L6389](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6389) | `e.ctrlKey && (e.key === 'e' \|\| 'E')` |
| 8 | Ctrl+P 启动器 | [L6398](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6398) | `e.ctrlKey && (e.key === 'p' \|\| 'P')` |
| 9 | Ctrl+Q 退出程序 | [L6426](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6426) | `e.ctrlKey && (e.key === 'q' \|\| 'Q')` |
| 10 | Ctrl+Home 滚动顶部 | [L6598](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6598) | `e.ctrlKey && e.key === 'Home'` |
| 11 | Ctrl+End 滚动底部 | [L6604](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6604) | `e.ctrlKey && e.key === 'End'` |

**配套放行逻辑（第 12 处）**：[L6589-6593](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6589-L6593) 在编辑器打开时让 Ctrl+Home/End 交给编辑器原生处理。若不同步改为双兼容，mac 上编辑器打开时按 Cmd+Home/End 会落到全局滚动（L6598/6604），与 Windows 行为不一致——必须同步修改。

**不改动**：Ctrl+A/D/数字（[L6518](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6518)、[L6538](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6538) 已双兼容）；输入框内 Enter/Ctrl+Enter（待办 [L6180](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6180)、AI 文本 [L9951](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9951)、聊天 [ai-chat.js L1826](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1826)）属文本输入辅助，本次不涉及。

## Proposed Changes

仅修改 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 一个文件，12 处 `e.ctrlKey` → `(e.ctrlKey || e.metaKey)`：

1. **L6321**：`if (e.ctrlKey && (e.key === 's' || e.key === 'S'))` → `if ((e.ctrlKey || e.metaKey) && (e.key === 's' || e.key === 'S'))`
2. **L6330**：`if (e.ctrlKey && e.key === 'f')` → `if ((e.ctrlKey || e.metaKey) && e.key === 'f')`
3. **L6354**：`if (e.ctrlKey && e.key === 'h')` → `if ((e.ctrlKey || e.metaKey) && e.key === 'h')`
4. **L6363**：`if (e.ctrlKey && e.key === 'n')` → `if ((e.ctrlKey || e.metaKey) && e.key === 'n')`
5. **L6372**：`if (e.ctrlKey && (e.key === 'l' || e.key === 'L')` → `if ((e.ctrlKey || e.metaKey) && (e.key === 'l' || e.key === 'L')`
6. **L6380**：`if (e.ctrlKey && (e.key === 'j' || e.key === 'J')` → `if ((e.ctrlKey || e.metaKey) && (e.key === 'j' || e.key === 'J')`
7. **L6389**：`if (e.ctrlKey && (e.key === 'e' || e.key === 'E'))` → `if ((e.ctrlKey || e.metaKey) && (e.key === 'e' || e.key === 'E'))`
8. **L6398**：`if (e.ctrlKey && (e.key === 'p' || e.key === 'P'))` → `if ((e.ctrlKey || e.metaKey) && (e.key === 'p' || e.key === 'P'))`
9. **L6426**：`if (e.ctrlKey && (e.key === 'q' || e.key === 'Q'))` → `if ((e.ctrlKey || e.metaKey) && (e.key === 'q' || e.key === 'Q'))`
10. **L6591**：`((e.ctrlKey && (e.key === 'Home' || e.key === 'End'))` → `(((e.ctrlKey || e.metaKey) && (e.key === 'Home' || e.key === 'End'))`
11. **L6598**：`if (e.ctrlKey && e.key === 'Home')` → `if ((e.ctrlKey || e.metaKey) && e.key === 'Home')`
12. **L6604**：`if (e.ctrlKey && e.key === 'End')` → `if ((e.ctrlKey || e.metaKey) && e.key === 'End')`

顺带更新受影响行的行内/相邻注释（如 L6301 函数注释、L6589 放行注释）提及 "Ctrl/Cmd"，保持文档与实际行为一致（仅注释，不改逻辑）。

## Assumptions & Decisions

- **决策**：采用双兼容 `(e.ctrlKey || e.metaKey)`，与项目已有 Ctrl+A/D/数字 的风格一致，无需平台检测（方案 B 被排除）。
- **决策**：输入框内 Enter/Ctrl+Enter（待办换行、AI 换行、聊天换行）**不改**——它们是文本输入辅助，mac 上按 Ctrl+Enter 同样可用，且 mac 上 Cmd+Enter 在聊天类应用常映射"发送"，改动涉及产品语义决策，超出本次范围。
- **假设**：macOS 上 CM6 编辑器内的 Cmd+F/H 与全局 handler 的响应顺序与 Ctrl 场景一致（Wails WebView 内实测验证，见 Verification）。

## Verification steps

1. `GetDiagnostics` 检查 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 无语法错误。
2. `cd frontend && npm run lint` 通过（项目现有 lint 任务）。
3. Windows 本机运行应用，逐一验证 Ctrl+S/F/H/N/L/J/E/P/Q、Ctrl+Home/End 行为与修改前一致（回归）。
4. macOS 实机（如有）验证 Cmd 版本生效；无 mac 环境时标注为待用户实机验证。
