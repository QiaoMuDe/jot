# 笔记本侧栏三段式配色与样式重设计计划（全主题适配）

## 概述

对笔记本侧栏的 **头部 → 列表 → 底部** 三个区域进行配色与样式重设计，使其层次清晰、视觉协调，并**在全部 6 个主题下均表现良好**。

## 当前状态分析

当前侧栏的三个区域在视觉上区分不够明确，**且该问题在所有主题中都存在**：

| 区域 | 当前样式问题（各主题共性） |
|------|--------------------------|
| **Header** | `color-mix(96% bg-secondary, 4% accent)` 微暖底色太不明显，与列表区边界模糊 |
| **List** | hover 使用 `--hover-bg`，和 `--bg-secondary` 差异太小（不同主题差异 2-8%），反馈微弱 |
| **Footer** | `border-top: 50% transparent` 分隔线太弱；新建按钮透明背景，缺乏存在感 |

## 各主题 CSS 变量对比

所有改动均使用 CSS 变量和 `color-mix`，**无需为每个主题写单独规则**：

| 主题 | `--bg-secondary` | `--card-bg` | `--accent` | `--border` |
|------|-----------------|-------------|-----------|-----------|
| **默认（暖）** | #EDE9E0 暖米 | #FFFFFF | #D97706 琥珀 | #E5E0D8 |
| **Light** | #F0F0F0 冷灰 | #FFFFFF | #2563EB 蓝 | #EEEEEE |
| **Dark** | #181818 深黑 | #1A1A1A | #F59E0B 琥珀 | #2A2A2A |
| **Nord** | #E2E7EE 灰蓝 | #FFFFFF | #5E81AC 钢蓝 | #D8DEE9 |
| **Monokai** | #1F1D20 深紫 | #221F22 | #FF6188 粉 | #3C3A3D |
| **Tokyo** | #15161F 深靛 | #24283B | #7AA2F7 蓝 | #2F354A |

## 实施方案

### 变更 1：侧栏 Header — 强化分区感

**文件**：`frontend\src\style.css`

**具体改动**：

```css
.sidebar-header {
  /* 从 color-mix(96% bg-secondary, 4% accent) 改为： */
  background: var(--card-bg);
  /* border-bottom 从半透明改为： */
  border-bottom: 1px solid var(--border);
}
```

**效果说明**：
- 浅色主题（默认/Light/Nord）：白色 `--card-bg` 浮在 `--bg-secondary` 底色上，分区感鲜明
- 深色主题（Dark/Monokai/Tokyo）：`--card-bg` 比 `--bg-secondary` 略浅，形成微妙层级

### 变更 2：侧栏 List — 增强交互反馈

**文件**：`frontend\src\style.css`

**具体改动**：

```css
.notebook-item:hover {
  /* 从 var(--hover-bg) 改为： */
  background: color-mix(in srgb, var(--accent) 8%, transparent);
}

.notebook-item.active {
  /* 从 color-mix(60% accent-lighter) 改为： */
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  font-weight: 600;
}
```

### 变更 3：侧栏 Footer — 强化功能入口

**文件**：`frontend\src\style.css`

**具体改动**：

```css
.sidebar-footer {
  /* 新增 background： */
  background: var(--card-bg);
  /* border-top 从半透明改为： */
  border-top: 1px solid var(--border);
}

.sidebar-new-btn:hover {
  /* 从 color-mix(60% accent-lighter) 改为： */
  background: color-mix(in srgb, var(--accent) 10%, transparent);
  color: var(--accent);
}
```

### 变更 4：Badge 同步微调

**文件**：`frontend\src\style.css`

```css
.notebook-item:hover .notebook-badge {
  /* 从 var(--text-secondary) 改为： */
  color: var(--accent);
}
```

## 核心设计原理

所有改动严格使用 **CSS 变量 + `color-mix`**，不写入任何硬编码色值：

1. 每个主题的 `--card-bg`、`--bg-secondary`、`--accent` 等变量自动组合出该主题匹配的配色
2. **Header/Footer 使用 `--card-bg`** 形成上下夹心，列表区作为内容凹槽
3. **hover 反馈使用 accent 色光晕**而非灰色底，在所有主题下都醒目且统一
4. 未来新增主题自动适配，无需额外 CSS

## 不修改的边界

- `.collapsed` 折叠态完全隐藏
- 右键菜单、重命名输入框、新建弹窗
- badge 的其他已有样式

## 验证步骤

1. `go build ./...` 编译通过
2. 逐一检查 6 个主题下的三个区域视觉效果
3. 每个主题悬停列表项验证反馈
4. 确保右键菜单、新建弹窗等附属 UI 不受影响
