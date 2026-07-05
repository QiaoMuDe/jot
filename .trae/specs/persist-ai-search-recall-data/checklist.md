# Checklist

- [x] AIMessage 模型新增 `SearchSources TEXT` 和 `RecallCards TEXT` 字段
- [x] `services.Message` 结构体新增 `SearchSources` 和 `RecallCards` 字段
- [x] `SaveAIMessages` 方法在保存 assistant 消息时将结构化数据写入数据库
- [x] `CallAIStream` 中将搜索来源和召回卡片数据缓存并传入 `SaveAIMessages`
- [x] 前端 `addMessage()` 支持渲染搜索来源/召回卡片折叠面板
- [x] 前端 `switchSession()` 加载消息时恢复搜索来源/召回卡片 UI
- [x] 开启联网搜索发送消息，切换会话后搜索来源折叠面板仍然显示
- [x] 开启卡片召回发送消息，切换会话后召回卡片折叠面板仍然显示
- [x] 未开启搜索/召回时，历史消息不会出现空面板
