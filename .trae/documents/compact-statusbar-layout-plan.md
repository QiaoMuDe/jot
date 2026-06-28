# 状态栏紧凑布局方案

## 摘要

将编辑器底部状态栏左侧（字数、字符、后缀名）的间距收窄，同时确保内容增多时布局正常不溢出。

## 当前状态分析

`.editor-footer-left` (`flex: 1`) 内包含两个元素：
- `.word-count` — 显示 `12345 个字数 | 12345 个字符`（`white-space: nowrap`）
- `.file-ext` — 显示 `.txt`（`white-space: nowrap`，`::before` 已有 `| ` 分隔符）

两者之间 `gap: 12px`，导致状态栏视觉松散：

```
[12345 个字数 | 12345 个字符]  --- 12px gap ---  [| .txt]
                                                   ↑ ::before 分隔符
```

## 修改方案

**只改一处 CSS**，不动 HTML 和 JS。

### 文件：`frontend/src/css/components/editor.css`

**修改**：将 `.editor-footer-left` 的 `gap` 从 `12px` 改为 `4px`

```css
.editor-footer-left {
  display: flex;
  align-items: center;
  gap: 4px;    /* 12px → 4px */
  flex: 1;
}
```

### 效果

```
[12345 个字数 | 12345 个字符]  | .txt
                                  ↑ 4px gap + ::before "| " 天然紧凑
```

## 内容增长后的表现

- `.word-count` 和 `.file-ext` 均有 `white-space: nowrap` → 数字再多也不换行
- `.editor-footer-left` 有 `flex: 1` → 可随内容自然扩展宽度
- `.editor-footer-right` 也有 `flex: 1` → 左右平衡，不会挤压右侧按钮
- 数字从 `1` 位增到 `6` 位（如 `123456 个字数 | 123456 个字符`）也只多 ~80px 宽度，在桌面端编辑器面板宽度下完全可容纳

## 验证

1. 打开编辑器，检查状态栏左侧间距是否收紧
2. 输入大量文字使字数/字符数增长到万级（如 `12345 个字数 | 67890 个字符`），确认布局不换行、不溢出、右侧按钮正常
3. 切换到查看模式，确认 `.editor-edit-time` 显示时布局仍正常
