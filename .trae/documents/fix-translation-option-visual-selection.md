# 修复翻译方向选中后菜单内视觉状态未更新问题

## 问题描述

在"更多技能"菜单中，展开翻译方向选择区，点击"翻译为英文"后关闭菜单。再次打开菜单时，"翻译为中文"选项仍然显示为选中状态（紫色文字 + 勾号），与实际生效的翻译方向不符。

## 根因分析

### 问题定位文件

- `frontend/index.html` — HTML 结构（第 878-886 行）
- `frontend/src/js/ai-chat.js` — 交互逻辑（第 884-900 行）
- `frontend/src/css/components/ai-chat.css` — 选中态样式（第 878-892 行）

### 问题机制

1. **视觉选中机制**：翻译方向选项的 Radio `<input>` 设置了 `display: none`（`ai-chat.css` 第 890 行），选中状态通过外层 `<label>` 的 `.selected` 类来视觉呈现（紫色文字 + 左侧 `✓` 勾号）。

2. **默认状态**：`index.html` 第 879 行硬编码了 `class="... selected"` 在"翻译为中文"的 `<label>` 上。

3. **点击处理逻辑**（`ai-chat.js` 第 884-900 行）：当用户点击某个方向选项时，代码仅做了：
   - `radio.checked = true` — 更新 Radio 的选中状态
   - `activeSkills.translate = { direction: dir }` — 更新 JS 状态
   - `renderSkillChips()` — 更新 chip 指示条
   - 关闭菜单
   
   **缺少关键步骤**：从未更新 `<label>` 上的 `.selected` 类。

4. **后果**：无论用户选择什么方向，"翻译为中文"的 `<label>` 始终保留 `.selected` 类，视觉上始终显示为选中态。

## 修改方案

### 修改文件

`frontend/src/js/ai-chat.js`，第 884-900 行

### 修改内容

在翻译方向选项的点击处理逻辑中，增加 `.selected` 类的更新：

```js
// 点击方向选项
const option = e.target.closest('.ai-chat-skills-option');
if (option) {
    const radio = option.querySelector('input[type="radio"]');
    if (radio) {
        radio.checked = true;
        // 更新视觉选中状态：清除所有选项的 selected 类，再给当前选项添加
        const parent = option.closest('.ai-chat-skills-options');
        if (parent) {
            parent.querySelectorAll('.ai-chat-skills-option').forEach(el => {
                el.classList.remove('selected');
            });
        }
        option.classList.add('selected');

        // 激活翻译技能, 先清空其他技能
        activeSkills = {};
        const dir = radio.value; // 'to_chinese' or 'to_english'
        activeSkills.translate = { direction: dir };
        renderSkillChips();
        // 关闭整个菜单
        skillsDropdown.classList.remove('open');
        if (skillsTranslateOptions) skillsTranslateOptions.classList.remove('open');
    }
    return;
}
```

### 修改说明

- 新增的 5 行代码：先清除 `.ai-chat-skills-options` 内所有 `<label>` 的 `.selected` 类，再给当前点击的选项添加 `.selected` 类。
- 保持其他逻辑不变。

## 影响范围

- 仅修改 JS 中翻译方向选择的事件处理
- 不涉及其他技能（编程、写作等），它们没有方向子菜单
- 不涉及 CSS 或 HTML 结构变更

## 验证方式

1. 打开"更多技能"菜单，展开翻译方向选择区
2. 默认"翻译为中文"显示为选中态（紫色 + 勾号）
3. 点击"翻译为英文"，菜单关闭
4. 再次打开菜单 → 观察翻译方向选择区中"翻译为英文"是否显示为选中态
5. 重复切换方向，确认选中态始终与最后一次选择一致
6. 同时确认 chip 指示条中的文本（"翻译 → 英文"/"翻译 → 中文"）与菜单中的选中态一致
