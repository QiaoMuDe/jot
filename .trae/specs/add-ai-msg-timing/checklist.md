# Checklist

## 后端
- [x] AIMessage 模型新增 `ThinkingElapsed` 和 `TotalElapsed` 字段
- [x] services.Message 结构体同步新增两个字段
- [x] CallAIStream onDone 回调签名新增两个耗时参数
- [x] CallAIStream 中正确记录 streamStart / thinkingStart / thinkingEnd 并计算耗时
- [x] 流式取消时不做耗时计算和保存
- [x] SaveAIMessages 写入 thinking_elapsed 和 total_elapsed
- [x] LoadAISessionMessages 返回 thinking_elapsed 和 total_elapsed
- [x] app.go CallAIStream 绑定适配新 onDone 签名，事件传递耗时参数

## 前端 — 思维链耗时
- [x] 流式正常结束且 thinkingElapsed > 0 时，summary 显示 `💭 已思考 X.X 秒`
- [x] 流式被取消时不做更新
- [x] 加载历史消息且 thinkingElapsed > 0 时显示 `💭 已思考 X.X 秒`
- [x] 加载历史消息且 thinkingElapsed == 0 时保持 `💭 已思考`

## 前端 — 回复耗时
- [x] 流式正常结束时在消息底部显示 `⏱ 总耗时 X.X 秒`
- [x] 加载历史消息且 totalElapsed > 0 时显示耗时
- [x] 流式被取消时不显示
- [x] 重新生成时重新计时并显示新耗时
- [x] `.ai-msg-time` 样式正确（小号、灰色、居右、上间距）