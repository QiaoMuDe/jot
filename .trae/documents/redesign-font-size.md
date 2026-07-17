# 字体大小选择器重新设计方案

## 现状

当前外观卡片中的字体大小设置项：

```
[大小]  [小][偏小][默认][大][特大] [16]px
```

由一排预设按钮 + 自定义数字输入组成。用户反馈"整一排按钮看着有点丑"。

## 改动范围

仅改造外观卡片中的"字体大小"设置项（font setting row），字体族选择器保持不变。

### 设计方向: 滑条 + 实时预览

- 去掉预设按钮组（`font-size-presets`）
- 去掉 px 自定义输入框（`font-size-custom`）
- 替换为一个**字号滑条** + 字号数值标签
- 在底部增加**实时预览区**，用当前字体显示示例文本

---

## 具体修改

### 文件 1: `frontend/index.html`

替换字体大小行的 HTML（约第 298-312 行），新结构：

```html
<div class="font-setting-row">
    <label class="font-setting-label">大小</label>
    <span class="font-setting-desc"></span>
    <div class="font-size-slider-wrap">
        <input type="range" id="fontSizeSlider" class="font-size-slider" min="10" max="32" value="16" step="1" />
        <span class="font-size-value" id="fontSizeValue">16px</span>
    </div>
</div>
<div class="font-preview-row" id="fontPreviewRow">
    <div class="font-preview-text" id="fontPreviewText">
        The quick brown fox jumps over the lazy dog
    </div>
</div>
```

### 文件 2: `frontend/src/css/components/dropdowns.css`

1. 移除 `.font-size-presets`、`.font-size-btn`、`.font-size-controls`、`.font-size-custom`、`.font-size-input`、`.font-size-unit` 样式（约第 108-173 行）

2. 新增 `.font-size-slider-wrap`、`.font-size-slider`、`.font-size-value` 样式

3. 新增 `.font-preview-row`、`.font-preview-text` 样式

滑条样式要点：
- 自定义滑条轨道和 thumb（使用 accent 主题色）
- `width: 180px`，垂直居中
- 数值标签在滑条右侧：`font-size: 1rem; font-weight: 600; min-width: 42px`

预览区样式要点：
- 在字体行下方单独一行，`margin-top: 8px; padding-left: 92px`（与编辑器和输入项对齐）
- 浅色圆角背景卡，`padding: 12px 16px`
- 预览文字使用当前 font-family 和 font-size

### 文件 3: `frontend/src/main.js`

调整约第 1546-1570 行的事件绑定：

1. **移除**: `.font-size-btn` click 事件绑定
2. **移除**: `#fontSizeInput` change 事件绑定
3. **新增**: `#fontSizeSlider` input 事件 → 更新 `#fontSizeValue` 显示 + 调用 `applyFontSize()` + 更新预览区
4. **修改**: `applyFontSize(size)` — 增加更新滑块位置的逻辑
5. **修改**: `loadSettings()` — 增加恢复滑块位置逻辑
6. **修改**: `updateFontSettingsUI()` — 移除预设按钮高亮逻辑，改为更新滑块

---

## 实现步骤

1. CSS: 添加滑条和预览区样式，移除旧按钮样式
2. HTML: 替换大小行结构，增加预览行
3. JS: 绑定滑条事件，移除旧按钮/输入事件
4. 验证: 拖动滑条字号即时变化，预览区同步，保存/恢复正确

---

## 验证

1. 滑条拖动到 10-32px 任意位置，页面字号即时变化
2. 预览区文字以当前字体 + 字号渲染
3. 保存后刷新，字号恢复正确
4. 外观卡片整体布局协调
