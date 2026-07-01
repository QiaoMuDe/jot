# AI 模型选择器添加搜索过滤

## 问题

当 AI 服务商提供大量模型（如 OpenAI 数百个、Ollama 数十个）时，两个模型选择器——AI 对话工具栏的 `#aiChatModelDropdown` 和设置页的 `#aiModelDropdown`——都是纯平铺列表，没有搜索/过滤功能，用户找模型需要滚动浏览全部列表，体验差。

## 方案

在两个模型选择器的下拉菜单内顶部各添加一个实时搜索输入框，输入文字后即时过滤下面列表中的模型项。

## 设计原则（frontend-design）

- **克制实用**：搜索框仅在下拉打开时出现，不影响现有视觉结构。搜索框获得焦点时自动聚焦，按 Esc 或下拉关闭时清空。
- **视觉融合**：搜索框使用 `--input-bg` 背景、`--border` 边框，圆角与菜单项一致，与下拉菜单浑然一体，不突兀。
- **交互流畅**：输入即过滤（无防抖），匹配项实时显示/隐藏。下拉关闭后自动清空搜索词，下次打开恢复完整列表。

## 修改文件与具体变更

### 1. `frontend/index.html`

**AI 对话工具栏** — 在 `#aiChatModelDropdown` 内添加搜索框：

```html
<div class="ai-chat-model-dropdown" id="aiChatModelDropdown">
  <!-- ↓ 新增 -->
  <div class="ai-model-search-wrap">
    <input class="ai-model-search" id="aiChatModelSearch" type="text" placeholder="搜索模型..." autocomplete="off" />
  </div>
  <!-- 原有模型列表将动态渲染到此容器内 -->
</div>
```

**设置页** — 在 `#aiModelDropdown` 内添加搜索框：

```html
<div class="theme-select-dropdown" id="aiModelDropdown">
  <!-- ↓ 新增 -->
  <div class="ai-model-search-wrap">
    <input class="ai-model-search" id="aiSettingModelSearch" type="text" placeholder="搜索模型..." autocomplete="off" />
  </div>
  <!-- 原有模型列表将动态添加到此处 -->
</div>
```

### 2. `frontend/src/css/components/ai-chat.css`

新增搜索框通用样式（在 `.ai-chat-model-dropdown` 块内）：

```css
.ai-model-search-wrap {
    padding: 4px 6px 2px;
    position: sticky;
    top: 0;
    background: var(--card-bg);
    z-index: 1;
}

.ai-model-search {
    width: 100%;
    padding: 5px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--input-bg);
    color: var(--text-primary);
    font-size: 0.78rem;
    outline: none;
    box-sizing: border-box;
    transition: border-color 0.15s;
}

.ai-model-search:focus {
    border-color: var(--accent);
}

.ai-model-search::placeholder {
    color: var(--text-muted);
}
```

### 3. `frontend/src/css/components/dropdowns.css`

在 `.theme-select-dropdown.open` 块中添加搜索框排版适配（确保搜索框在 dropdown 内显示正常）：

```css
.theme-select-dropdown.open {
    /* 原有样式不变 */
    display: flex;
    flex-direction: column;
}
```

### 4. `frontend/src/js/ai-chat.js`

修改 `renderModelDropdown()` 函数，在渲染模型列表后绑定搜索过滤逻辑：

```javascript
function renderModelDropdown() {
    if (!modelDropdown || !modelLabel) return;
    const current = modelLabel.textContent;
    // 清除现有列表项（保留搜索框）
    const items = modelDropdown.querySelectorAll('.theme-select-item');
    items.forEach(el => el.remove());
    // 渲染模型列表到搜索框后面
    const searchWrap = modelDropdown.querySelector('.ai-model-search-wrap');
    const fragment = document.createDocumentFragment();
    modelList.forEach(m => {
        const div = document.createElement('div');
        div.className = 'theme-select-item' + (m === current ? ' active' : '');
        div.dataset.model = m;
        div.textContent = m;
        fragment.appendChild(div);
    });
    // 插入到搜索框后面
    if (searchWrap) {
        searchWrap.after(fragment);
    } else {
        modelDropdown.appendChild(fragment);
    }
}
```

新增搜索过滤函数并绑定事件（在 `modelTrigger` 事件附近）：

```javascript
// 模型搜索过滤
const modelSearch = document.getElementById('aiChatModelSearch');
if (modelSearch) {
    modelSearch.addEventListener('input', () => {
        const query = modelSearch.value.trim().toLowerCase();
        modelDropdown.querySelectorAll('.theme-select-item').forEach(item => {
            const match = !query || item.textContent.toLowerCase().includes(query);
            item.style.display = match ? '' : 'none';
        });
    });
}
```

在 `openModelDropdown()` 中，下拉打开后自动聚焦搜索框：

```javascript
async function openModelDropdown() {
    // ... 原有逻辑 ...
    renderModelDropdown();
    modelDropdown.classList.add('open');
    // ↓ 新增：聚焦搜索框
    const search = modelDropdown.querySelector('.ai-model-search');
    if (search) setTimeout(() => search.focus(), 50);
}
```

在点击关闭下拉时清空搜索：

```javascript
// 在 document click 关闭逻辑处
document.addEventListener('click', () => {
    modelDropdown.classList.remove('open');
    // ↓ 新增：清空搜索
    const search = modelDropdown.querySelector('.ai-model-search');
    if (search) { search.value = ''; search.dispatchEvent(new Event('input')); }
});
```

Esc 键关闭下拉时也清空搜索：

```javascript
modelSearch.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        modelDropdown.classList.remove('open');
        const search = modelDropdown.querySelector('.ai-model-search');
        if (search) { search.value = ''; search.dispatchEvent(new Event('input')); }
    }
    // 阻止 Enter 提交
    if (e.key === 'Enter') e.preventDefault();
});
```

### 5. `frontend/src/main.js`

在设置页下拉菜单中添加搜索逻辑。在 `initAISettings()` 中打开下拉的 click 事件附近：

```javascript
// 在 dropdown 打开后聚焦搜索框
trigger.addEventListener('click', (e) => {
    // 原有逻辑...
    // ↓ 新增聚焦
    setTimeout(() => {
        const search = dropdown.querySelector('.ai-model-search');
        if (search) search.focus();
    }, 50);
});

// 下拉关闭时清空搜索
document.addEventListener('click', (e) => {
    if (!trigger.contains(e.target) && !dropdown.contains(e.target)) {
        dropdown.classList.remove('open');
        const search = dropdown.querySelector('.ai-model-search');
        if (search) { search.value = ''; filterModels(); }
    }
});

// 搜索过滤
function filterModels() {
    const search = document.getElementById('aiSettingModelSearch');
    if (!search) return;
    const query = search.value.trim().toLowerCase();
    dropdown.querySelectorAll('.theme-select-item').forEach(item => {
        item.style.display = !query || item.textContent.toLowerCase().includes(query) ? '' : 'none';
    });
}

// 绑定搜索事件
const settingSearch = document.getElementById('aiSettingModelSearch');
if (settingSearch) {
    settingSearch.addEventListener('input', filterModels);
    settingSearch.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            dropdown.classList.remove('open');
            settingSearch.value = '';
            filterModels();
        }
        if (e.key === 'Enter') e.preventDefault();
    });
}
```

同时修改 `addModelDropdownItem()` 或确保新添加的模型项同样受搜索过滤：

无需修改 `addModelDropdownItem`，因为 `filterModels()` 是在已存在的 DOM 上过滤，新加的 item 只需设置初始可见即可。

## 影响范围

- 仅两个模型下拉菜单的 UI/UX 改进
- 不影响后端、模型获取逻辑、模型切换逻辑
- 不影响其他下拉菜单（主题、语言、服务商等）
- 不影响核心编辑器功能

## 验证

1. 在 AI 对话页打开模型下拉，输入关键词过滤，确认列表实时收缩
2. 选择过滤后的模型，确认选择生效、下拉关闭
3. 再次打开下拉，确认搜索框已清空、列表恢复完整
4. 在设置页重复上述验证
5. 按 Esc 关闭下拉，确认搜索框已清空
6. 确认搜索框视觉风格与下拉菜单一致
