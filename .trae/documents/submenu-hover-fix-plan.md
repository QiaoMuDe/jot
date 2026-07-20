# 子菜单 hover 效果修复计划

## 问题

`#moreMenu` 中的子菜单（帮助参考 → 快捷键说明/MD 语法/关于）在鼠标悬停时，图标不响应 `translateX(2px)` 和 accent 色变化。主菜单条目的 hover 效果正常。

## 根因

`.dropdown-submenu` 的定位偏移在 trigger 和子菜单之间留下了 **4px 间隙**：

```css
.dropdown-submenu {
    left: calc(100% + 4px);  /* ← 这里！和 trigger 之间有 4px 空隙 */
    ...
}
```

```
  ┌──────────────────┐     ┌────────────────┐
  │ 帮助参考          │ ░░░░│ 快捷键说明     │
  │                  │ 4px │ MD 语法        │
  │                  │ gap │ 关于           │
  └──────────────────┘     └────────────────┘
         trigger               submenu
```

当鼠标从 trigger（帮助参考）右边缘向右移动到子菜单时，必须经过这 4px 空白区。在这一瞬间：
1. 鼠标离开 trigger → `:hover` 丢失
2. `dropdown-submenu-trigger:hover .dropdown-submenu` 规则失效
3. `pointer-events` 立即恢复为 `none`（该属性不参与 transition）
4. 子菜单开始关闭（opacity/transform transition 0.12s）
5. 鼠标到达子菜单位置时，子菜单已经不可交互

这就是为什么**主菜单项 hover 正常**（它们直接是 `#moreMenu` 的子元素，没有间隙问题），而**子菜单项 hover 永远无法「稳定生效」**。

## 改动

### 文件：`frontend/src/css/components/topbar.css`

**修改 `.dropdown-submenu` 的 `left` 值：**

```css
/* 旧 */
left: calc(100% + 4px);

/* 新 */
left: 100%;
```

移除 4px gap，让子菜单紧贴 trigger 右侧。鼠标可以平滑地从 trigger 移动到子菜单上，`:hover` 状态不会中断。

同时将 `top` 从 `-6px` 改为 `-6px`（不变），保留其他所有样式。

### 不做的事

- ❌ 不加 `::after` 桥接伪元素（没必要，直接消除间隙最简单）
- ❌ 不改 JS 逻辑
- ❌ 不改 HTML 结构
- ❌ 不影响主菜单图标 hover（`translateX(2px)` 不变）

## 验证

1. 鼠标悬停"帮助参考" → 子菜单应正常弹出
2. 鼠标平滑右移到子菜单条目上 → 图标应有 `translateX(2px)` + accent 色
3. 鼠标移出子菜单 → 子菜单应正常关闭
4. 主菜单其他条目的 hover 效果不受影响
