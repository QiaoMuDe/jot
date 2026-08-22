# Token 显示格式优化

## 概述

当前 `formatTokens` 函数只支持 `K` 单位，当 token 数达到几万甚至几十万时，显示为 `100.0K`、`1500.0K` 等不直观的格式。需要增加 `M` 单位支持，并优化显示精度规则。

---

## 当前状态

### 格式化函数

[frontend/src/js/ai-chat.js#L384-L387](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L384-L387)

```js
function formatTokens(count) {
    if (count >= 1000) return (count / 1000).toFixed(1) + 'K';
    return String(count);
}
```

### 调用位置（4 处）

| 行号 | 用途 | 显示位置 |
|------|------|---------|
| L400 | 会话总 token 数 | 标题栏右侧（`updateContextSize`） |
| L2555 | AI 回复 token 数 | 消息气泡底部操作栏 |
| L2999 | 历史消息 AI 回复 token 数 | 消息气泡底部操作栏（渲染） |
| L3592, L3601 | 用户消息 token 数 | 用户消息气泡底部 |

### 问题

| 实际 token 数 | 当前显示 | 期望显示 |
|---------------|---------|---------|
| 500 | `500` | `500` |
| 1,500 | `1.5K` | `1.5K` |
| 100,000 | `100.0K` | `100K` |
| 1,500,000 | `1500.0K` | `1.5M` |

---

## 变更清单

### 唯一改动：`frontend/src/js/ai-chat.js` — `formatTokens` 函数

```js
function formatTokens(count) {
    if (count >= 1000000) {
        // ≥ 1M: 显示 M 单位，1 位小数，如 1.5M
        return (count / 1000000).toFixed(1) + 'M';
    }
    if (count >= 1000) {
        // ≥ 1K: 显示 K 单位
        // ≥ 100K 时不显示小数（如 100K），< 100K 时显示 1 位小数（如 1.5K）
        if (count >= 100000) {
            return Math.round(count / 1000) + 'K';
        }
        return (count / 1000).toFixed(1) + 'K';
    }
    return String(count);
}
```

### 显示规则

| 范围 | 格式 | 示例 |
|------|------|------|
| < 1,000 | 原始数字 | `500` |
| 1,000 ~ 99,999 | 1 位小数 + K | `1.5K`、`10.0K`、`99.9K` |
| 100,000 ~ 999,999 | 整数 + K | `100K`、`500K` |
| ≥ 1,000,000 | 1 位小数 + M | `1.5M`、`10.0M` |

### 无需改动的文件

| 文件 | 理由 |
|------|------|
| 后端所有文件 | token 计算和存储逻辑不变，仅前端格式化显示 |
| CSS 文件 | 显示格式不变，仅数值文案变化 |
| 其他 JS 文件 | `formatTokens` 仅在 ai-chat.js 中定义和使用 |