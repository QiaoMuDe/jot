# AI 会话侧边栏条目右侧空白修复 Plan

## Summary
移除 AI 侧边栏中每个会话条目右侧到边框之间的多余空白，使条目视觉上更贴近侧栏右边界。

## Current State Analysis
造成右侧空白的原因有 3 层，层层叠加：

| 层级 | 选择器 | 当前值 | 贡献空间 |
|------|--------|--------|----------|
| 列表容器右 padding | `.ai-session-list` | `padding: 6px 8px` | 8px |
| 条目自身右 padding | `.ai-session-item` | `padding: 6px 6px` | 6px |
| 删除按钮残留空间 | `.ai-session-item-delete` | `width: 0; overflow: hidden; flex-shrink: 0` | flex-shrink:0 阻止零宽，可能残留 ~1-2px |
| 滚动条区域 | `.ai-session-list` | `overflow-y: auto`，6px 固定滚动条 | 6px（有滚动时） |
| **合计（有滚动条时）** | | | **~20-22px** |
| **合计（无滚动条时）** | | | **~14px** |

**根因确认**: 删除按钮 `flex-shrink: 0` + `width: 0` 的组合在某些浏览器引擎中并未将元素压缩到真正的零宽度，因为 `flex-shrink: 0` 阻止缩小，而 `overflow: hidden` 只隐藏溢出内容但不强制零宽度。这是最隐蔽但最关键的空白源。

## Proposed Changes

### 1. 删除按钮改用 display 切换（彻底消灭占用空间）
- **文件**: `frontend/src/css/components/ai-chat.css`
- **选择器**: `.ai-session-item-delete`
- **操作**:
  - 默认状态：`width: 0` → `display: none`（彻底脱离布局）
  - hover 状态：`display: flex; width: 20px`
- **理由**: `display: none` 是唯一能确保元素完全不参与 flex 布局的方式。`width: 0 + overflow: hidden` 在不同浏览器引擎中表现不一致，`flex-shrink: 0` 可能阻止压缩到真正的零宽

### 2. 减小整个列表的右 padding
- **文件**: `frontend/src/css/components/ai-chat.css`
- **选择器**: `.ai-session-list`
- **操作**: `padding: 6px 8px` → `padding: 6px 4px 6px 8px`
- **理由**: 右 padding 从 8px 减到 4px，减少右侧多余空间，左侧保持 8px 不变（与搜索框/header 对齐）

### 3. 减小条目自身右 padding
- **文件**: `frontend/src/css/components/ai-chat.css`
- **选择器**: `.ai-session-item`
- **操作**: `padding: 6px 6px` → `padding: 6px 4px`
- **理由**: 右 padding 从 6px 减到 4px，同时左侧也减到 4px 保持对称

## 最终效果
- **无滚动条时**: 右侧空白 = 4px (list) + 4px (item) = **8px**（原来是 14px）
- **有滚动条时**: 右侧空白 = 4px (list) + 6px (scrollbar) + 4px (item) = **14px**（原来是 ~20px）
- 删除按钮完全不占布局空间

## Verification
1. 打开 AI 助手 → 查看会话侧边栏条目右侧到边框的间距
2. 确保 hover 条目时删除按钮正常显示（`display: flex; width: 20px`）
3. 确保 active 状态的边框（`border: 1px solid`）内容完整可见
