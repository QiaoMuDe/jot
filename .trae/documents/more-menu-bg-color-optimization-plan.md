# 更多菜单背景色优化计划

## 概述

按主题色系分组定制 `#moreMenu` 背景色，替换当前一刀切的毛玻璃方案（`rgba(255,255,255,0.82)` + `backdrop-filter`），提升菜单可读性和各主题下的视觉一致性。

## 当前状态分析

### 背景问题

| 问题                                     | 根因                                                       |
| -------------------------------------- | -------------------------------------------------------- |
| 毛玻璃 18% 透明度让背景内容透出，文字阅读困难              | `rgba(255,255,255,0.82)` + `backdrop-filter: blur(24px)` |
| 亮色主题下菜单与 `--topbar-bg: #FFFFFF` 几乎融为一体 | 统一白玻璃方案不考虑各主题色系                                          |
| nord 主题被错误归类为暗色（实际为亮色），获得深色半透明背景       | `[data-theme="nord"]` 列在暗色覆盖组                            |
| 9 个主题缺少专属背景覆盖                          | 仅 dark/tokyo-night/dracula/one-dark-pro/nord 有覆盖         |

### 当前 `#moreMenu` 关键样式

```css
#moreMenu {
  background: rgba(255,255,255,0.82);
  backdrop-filter: blur(24px) saturate(1.2);
  box-shadow: 0 12px 48px rgba(0,0,0,0.10), 0 2px 8px rgba(0,0,0,0.06);
  border-top: 3px solid var(--accent);
  border-radius: var(--radius-xl);
  min-width: 200px;
  padding: 8px;
}

/* 暗色覆盖 — 包含 nord，实际仅应有 dark/tokyo-night/dracula/one-dark-pro */
[data-theme="dark"] #moreMenu,
[data-theme="tokyo-night"] #moreMenu,
[data-theme="dracula"] #moreMenu,
[data-theme="one-dark-pro"] #moreMenu,
[data-theme="nord"] #moreMenu {        /* ⚠️ 错误：nord 是亮色主题 */
    background: rgba(26,26,26,0.88);
}
```

## 改动方案

### 策略

1. **为每个主题新增** **`--more-menu-bg`** **CSS 变量**，精确控制菜单背景色
2. **移除** **`backdrop-filter`**，消除背景内容透过菜单造成的视觉噪音
3. **将** **`#moreMenu`** **background 改为纯色**（使用 `--more-menu-bg`），不透明度 ≥ 0.96
4. **增强阴影**，补偿去除毛玻璃后损失的深度感
5. **移除 nord 的暗色覆盖**，为其指定亮色系专属背景

### 具体色值

**纯白背景组** — 菜单背景比 `--topbar-bg`（纯白）略深 3-5%，产生层次分离：

| 主题               | `--topbar-bg` | `--more-menu-bg` (新增) | 说明   |
| ---------------- | ------------- | --------------------- | ---- |
| default          | `#FFFFFF`     | `#FAFAFA`             | 极淡暖灰 |
| light            | `#FFFFFF`     | `#F8F8F8`             | 极淡冷灰 |
| catppuccin-latte | `#FFFFFF`     | `#F8F8FA`             | 极淡紫灰 |

**暖色背景组** — 菜单背景取 `--card-bg` 暗化 3-5%（约 96% 亮度的近似值）：

| 主题            | `--card-bg` | `--more-menu-bg` (新增) | 说明     |
| ------------- | ----------- | --------------------- | ------ |
| gruvbox-light | `#F9F5D7`   | `#F5F0D0`             | 暖黄略深   |
| ysgrifennwr   | `#FAF4E3`   | `#F5EDD8`             | 暖米略深   |
| alice         | `#FFFCF5`   | `#FAF5EC`             | 奶油略深   |
| lightmind     | `#FAF7EF`   | `#F5F0E5`             | 山林纸面略深 |

**冷色背景组**：

| 主题          | `--card-bg` | `--more-menu-bg` (新增) | 说明     |
| ----------- | ----------- | --------------------- | ------ |
| nord        | `#FFFFFF`   | `#F0F3F8`             | 北极蓝白调  |
| quiet-light | `#FCF8F2`   | `#F7F2EC`             | 静谧暖灰略深 |

**特殊背景**：

| 主题             | `--card-bg` | `--more-menu-bg` (新增) | 说明    |
| -------------- | ----------- | --------------------- | ----- |
| eye-protection | `#D8F5DD`   | `#D0ECD5`             | 豆沙绿略深 |

**暗色背景组** — 提高不透明度至 0.95，加强阴影：

| 主题           | `--card-bg` | `--more-menu-bg` (新增) | 说明     |
| ------------ | ----------- | --------------------- | ------ |
| dark         | `#1A1A1A`   | `rgba(26,26,26,0.95)` | 更深，近纯黑 |
| tokyo-night  | `#24283B`   | `rgba(36,40,59,0.95)` | 靛蓝更深   |
| dracula      | `#21222C`   | `rgba(33,34,44,0.95)` | 深紫更深   |
| one-dark-pro | `#2C313A`   | `rgba(44,49,58,0.95)` | 暗灰更深   |

### 修改文件

#### 1. `frontend/src/css/variables.css`

为每个 `[data-theme="..."]` 块新增 `--more-menu-bg` 变量，色值如上表。

#### 2. `frontend/src/css/components/topbar.css`

1. 将 `#moreMenu` 的背景从硬编码 `rgba(255,255,255,0.82)` + `backdrop-filter` 改为 `var(--more-menu-bg)`，背景色纯色无透明度：

   ```css
   background: var(--more-menu-bg);
   /* 移除 backdrop-filter 行 */
   ```

2. 移除暗色覆盖代码块（`.dropdown-menu` 中的 `[data-theme="dark"] #moreMenu` 至 `[data-theme="nord"] #moreMenu` 段落）

3. 增强阴影（在 `#moreMenu` 样式中）：

   ```css
   box-shadow:
     0 12px 48px rgba(0,0,0,0.12),   /* 远层阴影微增 */
     0 2px 8px rgba(0,0,0,0.08);      /* 近层阴影微增 */
   ```

4. 添加 `border: 1px solid var(--border)` 让菜单边缘更清晰（可选，看效果决定）

#### 3. 无需修改文件

* `frontend/index.html` — HTML 结构不变

* `frontend/src/main.js` — JS 逻辑不变

* `frontend/src/css/animations.css` — 动画不变

### 预期效果

1. **可读性提升** — 纯色背景无透明度，文字清晰不穿透
2. **主题一致性** — 每个主题的菜单背景融入该主题色系，非统一白色
3. **深度感保持** — 增强阴影补偿去除毛玻璃后的层次感
4. **nord 修复** — 从错误的暗色组移到正确亮色组

### 验证步骤

1. 构建：`cd frontend && npm run build` 无报错
2. 依次切换所有 14 个主题，检查：

   * 菜单背景是否自然融入主题色系

   * 菜单是否与顶栏 `--topbar-bg` 有明显视觉分离

   * 文字（`--text-primary`/`--text-secondary`）在菜单上可读性良好

   * 分隔线、分组标签、快捷键提示等内部元素清晰可见
3. 功能验证：菜单打开/关闭/点击菜单项/ESC 关闭均正常

