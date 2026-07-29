# Checklist

- [x] `reasoningFramework` 常量已定义，包含 4 步内部推理框架文本
- [x] `baseNormsBoundaries` 末尾引用了 `reasoningFramework`
- [x] 无技能时 system prompt 正确拼接为 `baseIdentity + baseNormsBoundaries`（含推理框架）
- [x] 有技能时 system prompt 正确保留 `baseNormsBoundaries`（含推理框架），技能覆盖身份层
- [x] Tavily 搜索结果 `FormattedText` 包含「来源：Tavily 联网搜索」标签
- [x] 知乎站内搜索结果 `FormattedText` 包含「来源：知乎站内搜索」标签
- [x] 知乎全网搜索结果 `FormattedText` 包含「来源：知乎全网搜索」标签
- [x] 卡片召回 `FormattedText` 包含「来源：本地笔记，优先级最高」标签
- [x] 角色扮演笔记注入文本包含「来源：角色设定笔记」标签
- [x] 手动引用笔记注入文本包含「来源：手动引用笔记」标签
- [x] 上传文件注入文本包含「来源：上传文件」标签
- [x] 编译通过，无错误
