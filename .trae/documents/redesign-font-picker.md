# 字体选择器重新设计方案

## 现状

当前外观卡片中的字体设置分为两行：
1. **字体** — 下拉选择器（trigger + dropdown + 搜索）
2. **大小** — 预设按钮组（小/偏小/默认/大/特大）+ 自定义 px 输入

用户反馈"整一排按钮看着有点丑"，需要重新设计字体设置组件。

## 设计方向

**风格定位**: 紧凑型排版工具，像一个微缩的「字型调板」

**视觉改进要点**:
1. 字体族选择器改为更优雅的预览式触发按钮，右侧显示字体名称示例
2. 字体大小控件改为一个**尺寸滑条** + 实时预览字号，替换现有平面按钮组
3. 增加一个**实时预览区域**，用所选字体和大小显示示例文本

---

## 改动范围

### 文件 1: `frontend/index.html`

**外观卡片中替换以下结构**（约第 285-312 行）：

```
旧结构:
  .font-setting-row (字体) → .font-family-select > trigger + dropdown
  .font-setting-row (大小) → preset buttons + custom input

新结构:
  .font-setting-row (字体) → .font-picker > .fp-trigger（字体预览DOM） + .fp-dropdown（列表+搜索）
  .font-setting-row (大小) → .font-size-slider + .font-size-display + .font-size-label
  .font-preview-row (新增) → 实时预览区
```

### 文件 2: `frontend/src/css/components/dropdowns.css`

替换以下类（约第 1-173 行）：
- 移除 `.font-size-presets`、`.font-size-btn`、`.font-size-controls`、`.font-size-input`、`.font-size-unit`
- 新增 `.font-picker`、`.fp-trigger`、`.fp-dropdown`、`.fp-search`、`.fp-options`
- 新增 `.font-size-slider`、`.font-size-display`、`.font-preview-area`

### 文件 3: `frontend/src/main.js`

调整以下函数（约第 389、1243-1568 行）：
- `initFontSettings()` — 替换大小预设按钮事件为滑条事件
- `applyFontSize()` — 更新预览区
- 移除 `.font-size-btn` 相关事件绑定

---

## 设计细节

### 字体族选择器
- **触发器**: 显示当前字体名称 + 一个用该字体渲染的短示例字符（如 "Aa"）+ 箭头
- **下拉面板**: 搜索框 + 字体列表，每项以自身字体的`font-family`渲染
- **交互**: 点击触发开/关，搜索实时过滤，键盘导航，点击外部关闭

### 字体大小控制
- **滑条**: `input[type="range"]` 范围 10-32px，步长 1px
- **当前值显示**: 滑条右侧显示数字 + "px"，可点击编辑
- **去除旧按钮组**: 不再使用 5 个预设按钮

### 实时预览区
- 位于字体设置行下方，灰底圆角卡片
- 显示示例文本："The quick brown fox jumps over the lazy dog"
- 使用当前选择的 font-family 和 font-size 渲染
- 只有预览功能，不可编辑

---

## 实现步骤

1. CSS: 重写字体选择器样式，新增滑条样式，新增预览区样式
2. HTML: 替换现有字体/大小结构，新增预览行
3. JS: 绑定滑条事件，更新预览区
4. 验证: 字体/大小选择后实时生效，预览区同步更新，保存/恢复正确

---

## 验证

1. 字体族下拉可搜索、可选择，选择后页面字体即时更新
2. 滑条拖动到不同位置，字号即时变化，预览区同步
3. 预览区文字使用所选字体+字号渲染
4. 保存后刷新，设置恢复正确
