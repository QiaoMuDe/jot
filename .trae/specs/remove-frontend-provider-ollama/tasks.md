# Tasks

- [x] Task 1: index.html 删除服务商 UI 组件
  - [x] 删除对话设置「服务商」行与分段控件（`aiProviderSegmented` / `aiProviderIndicator` 及 OpenAI/Ollama 两个按钮，L505-516）
  - [x] 删除量化连接「服务商」行与分段控件（`aiEmbedProviderSegmented` / `aiEmbedProviderIndicator` 及按钮，L584-595）
  - [x] 删除预设弹窗「服务商」下拉（`presetModalProvider` / `presetModalProviderTrigger` / `presetModalProviderLabel` / `presetModalProviderDropdown` 及其 OpenAI/Ollama 选项，L2250-2257），保留其它预设弹窗字段
- [x] Task 2: main.js 删除全部 provider 相关代码（最大改动点）
  - [x] `els` 中删除 `aiProviderSegmented` / `aiProviderIndicator` / `aiEmbedProviderSegmented` / `aiEmbedProviderIndicator` 引用（L553-554、L566-567）
  - [x] 删除 `AI_DEFAULT_URLS` 中 `ollama` 条目（L1913-1916）
  - [x] 删除分段控件辅助函数 `getSegmentedValue` / `setSegmentedActive` / `repositionSegmentedIndicator`（L1925-1964），并确认无其它调用方（日志级别/排序/分页用独立实现）
  - [x] 删除 `getActiveProvider` / `setActiveProvider` / `repositionProviderIndicator` / `repositionEmbedProviderIndicator`（L2418-2440）
  - [x] 对话模块配置对象删除 `seg` / `indicator` 字段、`getProvider` 改为返回 `'openai'` 或直接内联固定值（L2455-2480）
  - [x] 量化模块配置对象同步处理（L2490-2510）
  - [x] `initApiConnectionModule` 内：`saveModuleConfig` 去掉 provider 判断、测试/获取模型事件的 provider 变量改为固定 `'openai'`、删除「服务商分段控件切换」事件整段（L2222-2358）
  - [x] `renderPresetList` 去掉 provider badge 与 `p.provider === current.provider` 匹配条件（L2004-2016），`current` 入参不再含 provider
  - [x] `createPresetRowElement` 去掉 provider badge 渲染，`openEditProfileModal` 调用改为固定传 `'openai'`（L3497-3518）
  - [x] 预设弹窗事件：删除服务商下拉切换/选中/遮罩关闭事件（L3082-3100）
  - [x] `testPresetConnection`：provider 固定 `'openai'`、去掉 `provider === 'openai' && !apiKey` 分支（L3126-3153）
  - [x] 新增预设弹窗打开逻辑：去掉 provider 标记与 `setActiveProvider`（L3170-3280）
  - [x] `openEditProfileModal` / `savePresetModal`：去掉 provider 读取，保存/编辑调用 `CreateProfile` / `UpdateProfile` 固定传 `'openai'`（L3287-3360）
  - [x] `loadProfiles` / `loadProfilesEmbed` 传入 `current` 时去掉 provider 字段（L3166-3220）
  - [x] `loadSettings`：删除 `setActiveProvider(cfg.ai_provider)`、embed 的 `setSegmentedActive`、`canFetch` / `embedCanFetch` 判断（L9085、L9109、L9239-9252）
  - [x] `saveSettings`：`ai_provider` / `ai_embed_provider` 改为固定 `"openai"`（L9300-9313）
  - [x] 键盘快捷键 Enter 中 `presetModalProviderDropdown` 判断删除（L6404-6414）
  - [x] 全局清理：`repositionProviderIndicator` / `repositionEmbedProviderIndicator` 调用点（L9381-9382、L9401-9402）一并删除
- [x] Task 3: ai-chat.js 简化 provider 判断
  - [x] `onAIChatViewActivated` 中 `hasRequired` 简化为 `!!cfg.api_key`（L3286-3289）
  - [x] 模型选择器调用 `FetchAIModels` 固定传 `'openai'`（L166）
- [x] Task 4: 清理剩余前端文件
  - [x] `data-management.js` L599 注释去掉"openai 需 key"的分支表述
  - [x] `settings-panel.css` 删除 `.preset-provider-badge` 样式块（L1242-1251）
- [x] Task 5: 构建与残留验证
  - [x] `npm run build` 构建通过
  - [x] 前端目录全局搜索确认无 `ollama` 残留（大小写不敏感）
  - [x] 确认向后端调用（TestAIBaseURL / TestAIConnection / FetchAIModels / CreateProfile / UpdateProfile）参数个数未变，provider 位置固定 `"openai"`

# Task Dependencies
- [Task 5] depends on [Task 1]、[Task 2]、[Task 3]、[Task 4]
- [Task 1] ~ [Task 4] 改动文件互不重叠，可并行
