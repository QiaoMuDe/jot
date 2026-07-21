# 系统主题下拉菜单支持上下方向键切换

## Summary

在设置页的系统主题下拉菜单中增加键盘方向键（ArrowUp/ArrowDown）支持。方向键切换主题时的行为与鼠标点击完全一致：立即应用并保存。无需预览/确认/取消等复杂状态管理。

## Current State

- 主题下拉菜单由 `buildThemeDropdown()`（[main.js#L1399-L1419](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L1399-L1419)）动态生成 14 个 `.theme-select-item`
- 每个 item 绑定 click 事件：`applyTheme(key)` → `localStorage.setItem()` → `saveSettings()` → 通知
- 下拉菜单本身无 `tabindex`，无法接收键盘事件
- 菜单项通过 `active` class 标记当前已保存的主题

## Proposed Changes

### 1. `frontend/src/main.js` — `buildThemeDropdown()` 函数内

**Why**: 在 dropdown 元素上添加键盘事件监听，支持方向键导航。

**How**:

在 `buildThemeDropdown()` 函数的末尾（生成完所有 item 后），给 `dropdown` 添加 `tabindex` 和 `keydown` 监听：

```javascript
// 使下拉菜单可聚焦以接收键盘事件
dropdown.setAttribute('tabindex', '-1');

dropdown.addEventListener('keydown', (e) => {
    const items = dropdown.querySelectorAll('.theme-select-item');
    if (items.length === 0) return;

    // 找到当前 active 的 item 索引
    const currentIndex = Array.from(items).findIndex(item =>
        item.classList.contains('active')
    );

    let targetIndex;
    if (e.key === 'ArrowDown') {
        e.preventDefault();
        targetIndex = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        targetIndex = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
    } else {
        return; // 非方向键不处理
    }

    // 模拟点击目标 item（与鼠标点击走完全相同的保存路径）
    items[targetIndex].click();
});
```

**原理**：
- 方向键找到当前 `.active` 项的索引，循环取上/下一个 item
- 直接调用 `items[targetIndex].click()`，复用已有 item 的 click handler
- 保持现有行为完全不变：applyTheme → save → notify → 关闭下拉

### 2. `frontend/src/css/components/dropdowns.css`（可选）

**Why**: 给 dropdown 添加键盘焦点指示，让键盘用户能看到焦点在哪里。

**How**: 在 `.theme-select-item` 的已有样式基础上添加 `:focus-visible`：

```css
.theme-select-item:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
}
```

## 涉及的文件

| 文件 | 修改 | 说明 |
|------|------|------|
| `frontend/src/main.js` | ~15 行新增 | 在 `buildThemeDropdown()` 末尾给 dropdown 加 `tabindex` + `keydown` 监听 |
| `frontend/src/css/components/dropdowns.css` | ~5 行（可选） | 新增 `.theme-select-item:focus-visible` 键盘焦点样式 |

## 不变的部分

- 鼠标点击行为：不变
- 鼠标滚轮滚动：不变（`overflow-y: auto` 天然支持）
- 外部点击关闭：不变
- `.active` class 逻辑：不变
- 保存通知：不变
- 无需新增状态变量

## Verification

1. 打开设置页，点击主题下拉菜单
2. 按 ArrowDown 键，应切换到下一个主题并即时应用保存
3. 按 ArrowUp 键，应切换到上一个主题并即时应用保存
4. ArrowDown 到最后一个后再按，应循环到第一个
5. ArrowUp 到第一个后再按，应循环到最后一个
6. 鼠标点击行为不变，仍然正常切换
7. `npm run build` 构建无报错
