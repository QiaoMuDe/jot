# Tasks

- [ ] Task 1: `recall_service.go` — 新增上下文截取算法 + 修改 `CardRecallSearch`
  - [ ] 新增 `extractContext(content string, keywords []string, contextChars int) string` 辅助函数
  - [ ] `CardRecallSearch` 签名新增 `maxChars int` 参数
  - [ ] 对每条召回笔记，判断长度是否超过 maxChars，超过则调用 extractContext 截取
  - [ ] `RecallCard` 新增 `Truncated bool` 字段标记截断状态
  - [ ] 格式化文本中体现截断标记（`...(内容已截断)`）

- [ ] Task 2: `app.go` — 调用方读取设置并传入
  - [ ] 调用 `CardRecallSearch` 前，从 `settingService.Get("ai_ref_max_chars")` 读取阈值
  - [ ] 空值/非法值时回退到默认值 5000
  - [ ] 将 maxChars 传入 `CardRecallSearch`

## Task Dependencies

- [Task 1] → [Task 2]
