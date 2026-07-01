# 模型搜索改进：条件显示 + 关键字高亮

## 目标

1. 模型列表 ≤ 1 个时隐藏搜索框（≥2 个时才显示）
2. 搜索时在模型条目中高亮匹配的关键字（保留过滤功能）

## 修改文件

### 1. `frontend/src/js/ai-chat.js` — AI 聊天工具栏下拉

**`renderModelDropdown()`** 中，渲染完列表后控制搜索框可见性：

```js
// 在函数末尾追加
const searchWrap = modelDropdown.querySelector('.ai-model-search-wrap');
if (searchWrap) {
    searchWrap.style.display = modelList.length > 1 ? '' : 'none';
}
```

**`openModelDropdown()`** 聚焦搜索前检查可见性：

```js
const search = modelDropdown.querySelector('.ai-model-search');
if (search && search.offsetParent !== null) setTimeout(() => search.focus(), 50);
// 改为条件聚焦（如果 display:none，offsetParent 为 null）
```

**搜索 input 事件处理** — 从纯隐藏改为高亮 + 隐藏：

```js
modelSearch.addEventListener('input', () => {
    const query = modelSearch.value.trim();
    modelDropdown.querySelectorAll('.theme-select-item').forEach(item => {
        const model = item.dataset.model;
        if (!query) {
            // 清空搜索 → 恢复 textContent，全部显示
            item.textContent = model;
            item.style.display = '';
            return;
        }
        const lowerModel = model.toLowerCase();
        const lowerQuery = query.toLowerCase();
        const idx = lowerModel.indexOf(lowerQuery);
        if (idx !== -1) {
            // 匹配 → 高亮 + 显示
            const before = model.substring(0, idx);
            const match = model.substring(idx, idx + query.length);
            const after = model.substring(idx + query.length);
            item.innerHTML = before + '<mark class="ai-search-highlight">' + match + '</mark>' + after;
            item.style.display = '';
        } else {
            // 不匹配 → 恢复 textContent + 隐藏
            item.textContent = model;
            item.style.display = 'none';
        }
    });
});
```

### 2. `frontend/src/main.js` — 设置页下拉

**`initAISettings()`** 中：

搜索 input 事件改为和 AI 聊天同样的高亮逻辑（使用 `dataset.modelValue`）：

```js
settingSearch.addEventListener('input', () => {
    const query = settingSearch.value.trim();
    dropdown.querySelectorAll('.theme-select-item').forEach(item => {
        const model = item.dataset.modelValue;
        if (!query) {
            item.textContent = model;
            item.style.display = '';
            return;
        }
        const lowerModel = model.toLowerCase();
        const lowerQuery = query.toLowerCase();
        const idx = lowerModel.indexOf(lowerQuery);
        if (idx !== -1) {
            const before = model.substring(0, idx);
            const match = model.substring(idx, idx + query.length);
            const after = model.substring(idx + query.length);
            item.innerHTML = before + '<mark class="ai-search-highlight">' + match + '</mark>' + after;
            item.style.display = '';
        } else {
            item.textContent = model;
            item.style.display = 'none';
        }
    });
});
```

**`clearSettingModelSearch()`** 中也需要恢复所有 item 的 textContent（因为之前可能用 innerHTML 改变了内容）：

```js
function clearSettingModelSearch() {
    const search = document.getElementById('aiSettingModelSearch');
    if (search) {
        search.value = '';
        // 恢复所有 item 的 textContent
        document.querySelectorAll('.theme-select-item').forEach(item => {
            const model = item.dataset.modelValue || item.dataset.model;
            if (model) item.textContent = model;
            item.style.display = '';
        });
    }
}
```

**搜索框可见性控制** — 在以下 3 处添加：

a) **`loadAISettings()`** 末尾（加载完单模型或无模型后）：

```js
// 载入完成后更新搜索框可见性
const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
if (wrap) wrap.style.display = els.aiModelDropdown.querySelectorAll('.theme-select-item').length > 1 ? '' : 'none';
```

b) **"获取列表"按钮回调** — 循环 `addModelDropdownItem` 完成后：

```js
const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
if (wrap) wrap.style.display = models.length > 1 ? '' : 'none';
```

c) **切换服务商回调** — 清除模型列表项后：

```js
const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
if (wrap) wrap.style.display = 'none';
```

### 3. `frontend/src/css/components/ai-chat.css` （可选）

`.ai-model-search-wrap.hidden` 可用于动画过渡，但直接用行内 `display: none` 即可，不需要修改 CSS。

## 影响范围

- 仅影响两个模型下拉菜单内部的搜索和显示逻辑
- 复用已有的 `.ai-search-highlight` 样式（会话搜索已用）
- 不影响后端、模型获取、模型切换等核心功能

## 验证

1. 获取单个模型（或清除所有模型）→ 打开下拉，搜索框不出现
2. 获取多个模型 → 打开下拉，搜索框出现
3. 输入搜索关键词 → 匹配项高亮显示、非匹配项隐藏
4. 清空搜索 → 恢复 textContent、全部显示
5. 切换服务商 → 模型列表清空，搜索框隐藏
6. AI 聊天工具栏同步验证 1-4
