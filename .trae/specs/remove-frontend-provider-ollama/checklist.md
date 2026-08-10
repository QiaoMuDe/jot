# Checklist

- [x] index.html 中对话设置与量化连接均无「服务商」分段控件（`aiProviderSegmented` / `aiEmbedProviderSegmented`）
- [x] index.html 预设弹窗无「服务商」下拉（`presetModalProvider*`），其余字段（名称/地址/Key）保留完好
- [x] main.js 无 `ollama` 残留；`AI_DEFAULT_URLS` 无 ollama 条目
- [x] main.js 中 provider 相关辅助函数、状态、事件绑定、badge 渲染均已删除且无空引用报错
- [x] main.js 中 `TestAIBaseURL` / `TestAIConnection` / `FetchAIModels` / `CreateProfile` / `UpdateProfile` 调用参数个数保持不变，provider 位置固定传 `"openai"`
- [x] `saveSettings` 中 `ai_provider` / `ai_embed_provider` 固定提交 `"openai"`
- [x] ai-chat.js `hasRequired` 简化为仅检查 `api_key`，无 ollama 分支
- [x] data-management.js 注释已更新，无 provider 分支表述
- [x] settings-panel.css 中 `.preset-provider-badge` 样式已删除
- [x] `npm run build` 构建通过
- [x] 前端目录全局搜索无 `ollama`（大小写不敏感）
