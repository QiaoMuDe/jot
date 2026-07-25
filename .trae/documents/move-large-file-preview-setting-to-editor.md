# 将大文件预览阈值设置项移到编辑器区域

## 问题

"大文件预览阈值"设置项（`ai_large_file_preview_threshold`）当前位于设置页的「对话与搜索」区域，但它控制的是 **编辑器行为**（`.md` 笔记超过阈值时自动切纯文本模式），与对话/搜索无关。应将其移到「编辑器」设置区域。

## 当前状态

### 对话与搜索区域（约第 514-521 行）
```html
                    <!-- 大文件纯文本预览阈值 -->
                    <div class="ai-setting-item">
                        <span class="ai-setting-label">大文件预览阈值</span>
                        <span class="font-setting-desc">超过此字符数的 .md 笔记自动切换为纯文本模式（字符）</span>
                        <div class="ai-setting-control" style="flex:none;margin-left:auto;">
                            <input type="number" id="aiLargeFilePreviewThreshold" class="settings-input" value="10000" min="1" max="100000" style="width:100px;" />
                        </div>
                    </div>
```

### 编辑器区域（约第 349-405 行）
```html
                <!-- 纯文本 MD 高亮 -->
                <div class="settings-section">
                    <div class="ai-group-header">...编辑器...</div>
                    <div class="font-settings">
                        <div class="font-setting-row">...语法高亮...</div>
                        <div class="font-setting-row">...全屏打开...</div>
                        <div class="font-setting-row">...自动换行...</div>
                        <div class="font-setting-row">...代码高亮主题...</div>
                        <div class="code-preview" id="codePreview"></div>
                    </div>
                </div>
```

## 修改方案

### 改动 1: 从「对话与搜索」区域移除

在 `frontend/index.html` 中，移除第 514-521 行（大文件预览阈值的整个 `.ai-setting-item` 块）。

### 改动 2: 插入到「编辑器」区域

在「自动换行」设置项之后、「代码高亮主题」之前，新增一行：

```html
                        <div class="font-setting-row">
                            <label class="font-setting-label">大文件预览阈值</label>
                            <span class="font-setting-desc">超过此字符数的 .md 笔记自动切换为纯文本模式（字符）</span>
                            <input type="number" id="aiLargeFilePreviewThreshold" class="settings-input" value="10000" min="1" max="100000" style="width:100px;margin-left:auto;" />
                        </div>
```

注意：使用 `.font-setting-row` 而非 `.ai-setting-item`，以保持编辑器区域样式一致。`margin-left:auto` 让输入框右对齐。

## 无需改动

- `main.js` 中的 `loadSettings()` / `saveSettings()` / change 事件绑定：均通过 `#aiLargeFilePreviewThreshold` ID 引用，无需修改
- 后端：`SettingsConfig` / `db.go` 等均不涉及 UI 布局，无需修改

## 验证

1. 打开设置页，确认「编辑器」区域出现"大文件预览阈值"输入框
2. 确认「对话与搜索」区域不再有"大文件预览阈值"设置项
3. 修改该值，触发 change 事件 → 自动保存并显示通知
4. 重新打开设置页，值应持久化