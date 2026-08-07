# Checklist

- [x] URL `change` 时空值不保存且提示「请先填写 API 地址」，后端配置保持原值
- [x] 斜杠错误态修正为非斜杠结尾时自动保存一次（无需再次失焦），错误样式移除
- [x] 原有斜杠结尾报错不保存行为保持（设置页 + 预设弹窗三个入口不受影响）
- [x] `switchProfile` 切换预设后模型下拉搜索框隐藏，与 `switchProfileEmbed` 一致
- [x] app.go 卡片召回段无 kwCards 死代码，`recallCardsJSON` 仍正确持久化、`ai:recall-cards` 事件照常发射
- [x] `startVectorIndex` 复用 `GetEmbedConfig`，provider/base_url/model 任一为空时拒绝启动并返回可读错误
- [x] `TestAIBaseURL`/`TestAIConnection` 签名与日志前缀保留，行为不变
- [x] openai/ollama 测试连通性与获取模型行为（端点/超时/状态码）不变，HTTP 模板已收敛
- [x] `go build ./...`、`golangci-lint run ./...` 通过
- [x] 前端 `npm run build` 通过
