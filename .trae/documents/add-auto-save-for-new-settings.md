# 为新设置项添加自动保存机制

## 问题

刚才新增的 `ai_web_search_max_chars`（联网搜索结果截断）和 `ai_large_file_preview_threshold`（大文件纯文本预览阈值）两个设置项，虽然在 `loadSettings()` 中加载、在 `saveSettings()` 中收集，但**没有绑定** **`change`** **事件监听器**。用户修改这两个值后，必须手动点击设置页的"保存"按钮才能生效，不符合其他设置项"修改即自动保存"的交互模式。

## 当前状态

* 旧 `aiRefMaxChars` 的 `change` 事件监听已在 Task 8 中移除

* 其他类似设置项（如 `maxFileSize`、`aiSearchResultLimit`、`aiBaseURL` 等）都有 `change → saveSettings() → 通知` 模式

* `aiWebSearchMaxChars` 和 `aiLargeFilePreviewThreshold` 在 `saveSettings()` 中已被正确收集，只是缺少触发存档的时机

## 修改方案

在 `main.js` 中，在 `maxFileSize` 的 change 事件绑定（约第 2522 行结束）之后，新增两个 change 事件绑定：

### 改动 1: 新增 `aiWebSearchMaxChars` 的 change 事件

位置：`main.js` 第 2522 行后（`maxFileSize` 事件块结束后）

```javascript
    // ── 联网搜索结果截断自动保存 ──
    const webSearchMaxChars = document.getElementById('aiWebSearchMaxChars');
    if (webSearchMaxChars) {
        webSearchMaxChars.addEventListener('change', async () => {
            const val = parseInt(webSearchMaxChars.value);
            if (isNaN(val) || val < 1) {
                webSearchMaxChars.value = 5000;
                nm.show('搜索结果截断字数必须大于 0，已重置为 5000', 'warning');
                return;
            }
            if (val > 50000) {
                webSearchMaxChars.value = 50000;
                nm.show('搜索结果截断字数不能超过 50000，已重置为 50000', 'warning');
                return;
            }
            await saveSettings();
            nm.show('搜索结果截断字数已保存', 'success');
        });
    }
```

### 改动 2: 新增 `aiLargeFilePreviewThreshold` 的 change 事件

位置：`main.js` 在 `webSearchMaxChars` 事件块结束后

```javascript
    // ── 大文件预览阈值自动保存 ──
    const largeFileThreshold = document.getElementById('aiLargeFilePreviewThreshold');
    if (largeFileThreshold) {
        largeFileThreshold.addEventListener('change', async () => {
            const val = parseInt(largeFileThreshold.value);
            if (isNaN(val) || val < 1) {
                largeFileThreshold.value = 10000;
                nm.show('大文件预览阈值必须大于 0，已重置为 10000', 'warning');
                return;
            }
            if (val > 100000) {
                largeFileThreshold.value = 100000;
                nm.show('大文件预览阈值不能超过 100000，已重置为 100000', 'warning');
                return;
            }
            await saveSettings();
            nm.show('大文件预览阈值已保存', 'success');
        });
    }
```

## 验证

1. 打开设置页，修改"搜索结果截断"输入框的值，触发 `change` 事件 → 应自动保存并显示通知
2. 打开设置页，修改"大文件预览阈值"输入框的值，触发 `change` 事件 → 应自动保存并显示通知
3. 超出范围的值应自动重置为边界值并显示警告

