# 修复设置页三个问题

## 问题 1：知乎 Token 默认未隐藏

### 根因
设置页是 SPA 内嵌在 Wails 中的，DOM 元素在视图切换时持久化。用户如果在设置页点击了眼睛图标切换为明文（`type="text"`），切换到其他视图再切回设置页时，输入框的 `type` 属性**仍保持上一次的 `text`**，没有重置为 `password`。

### 修复
在 `loadSettings()` 中显式将 `aiZhihuAccessSecret` 和 `aiTavilyApiKey` 的 `type` 重置为 `password`，同时同步更新眼睛图标的显示状态。

**涉及文件：** `frontend/src/main.js`

---

## 问题 2：搜索开关未判断 Token/Key 是否已配置

### 现状
三个搜索开关（知乎搜索、全网搜索、Tavily搜索）在 `zhihu_access_secret` 或 `tavily_api_key` 为空时仍然可以开启，但实际搜索时会失败。

### 修复方案

#### 2.1 设置页开关禁用
在 `loadSettings()` 中：
- 如果 `cfg.zhihu_access_secret` 为空 → `aiSettingZhihuSearchLine` 和 `aiSettingZhihuGlobalSearchLine` 添加 `disabled` class，阻止点击
- 如果 `cfg.tavily_api_key` 为空 → `aiSettingTavilySearchLine` 添加 `disabled` class，阻止点击
- 自动将未配置对应 key 的开关强制关闭（移除 `active`），并保存
- 在开关的点击事件处理器中，检查父级是否有 `disabled` class，有则放弃操作

#### 2.2 AI 聊天栏复选框禁用
在 `ai-chat.js` 的工具栏同步函数 `syncToolbarState()` 和初始化代码中：
- 读取 `window.go.main.App.GetAllSettings()` 或 DOM 中的密码字段值判断 key 是否为空
- 如果对应 key 为空 → 复选框 `disabled`，显示灰色
- 在复选框 `change` 事件中检查是否 disabled

#### 2.3 CSS 禁用样式
新增 `.ai-setting-toggle-row.disabled` 和 `.ai-chat-search-source-item.disabled` 样式：
- 降低透明度（`opacity: 0.5`）
- `cursor: not-allowed`
- 阻止事件（`pointer-events: none`）

**涉及文件：**
- `frontend/index.html`（给搜索开关行和复选框容器添加 data 属性用于定位）
- `frontend/src/main.js`（loadSettings + 点击事件处理器）
- `frontend/src/js/ai-chat.js`（syncToolbarState + checkbox change 事件）
- `frontend/src/css/components/ai-chat.css`（禁用样式）
- `frontend/src/css/components/settings-panel.css`（禁用样式）

---

## 问题 3：知乎注册链接 URL 错误

### 现状
当前 URL: `https://open.zhihu.com`
正确 URL: `https://developer.zhihu.com/`

### 修复
在 `frontend/src/main.js` 中将 `.zhihu-link` 点击事件中的 URL 改为 `https://developer.zhihu.com/`。
同时在 `frontend/index.html` 中的提示文字也从 `open.zhihu.com` 改为 `developer.zhihu.com`。

**涉及文件：**
- `frontend/src/main.js`
- `frontend/index.html`

---

## 涉及文件清单

| 文件 | 修改内容 |
|------|----------|
| `frontend/index.html` | ① 提示文字 `open.zhihu.com` → `developer.zhihu.com`；② 搜索开关行添加 `data-depends-on` 属性（可选） |
| `frontend/src/main.js` | ① 加载设置时重置 password 输入框 type；② 加载设置时判断 key 并 disable 开关；③ 开关点击处理器判断 disabled；④ URL 修复 |
| `frontend/src/js/ai-chat.js` | ① `syncToolbarState()` 中判断 key 并 disable 复选框；② 复选框 change 事件判断 disabled |
| `frontend/src/css/components/settings-panel.css` | 新增 `.ai-setting-toggle-row.disabled` 样式 |
| `frontend/src/css/components/ai-chat.css` | 新增 `.ai-chat-search-source-item.disabled` 样式 |

## 验证步骤

1. **问题 1：** 进入设置页，点击眼睛图标显示 Token，切换到其他视图再返回设置页，确认 Token 自动隐藏
2. **问题 2：** 清空 Tavily API Key 并保存，确认 Tavily 搜索开关变为灰色不可点击；填入 Key 后重新加载设置页，开关恢复正常
3. **问题 2（聊天栏）：** Tavily Key 为空时，AI 聊天栏的 Tavily 搜索复选框为灰色不可勾选
4. **问题 3：** 点击提示中的知乎链接，确认浏览器打开 `https://developer.zhihu.com/`
