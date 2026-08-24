# 名称长度限制统一方案

## 摘要
统一笔记本名称和AI会话标题的长度限制为50字符，前后端校验对齐，自动标题截断到50字。

## 当前状态
- 笔记本前端 maxlength=50，后端无校验，DB VARCHAR(100)
- AI会话前端 maxlength=100，后端无校验，DB VARCHAR(100)
- 自动标题已移除截断，直接存全文

## 变更方案

### 1. 前端 — `ai-chat.js`：AI会话重命名 maxlength 50→50
- `showAISessionRenameDialog` 中 input 的 `maxlength="100"` → `maxlength="50"`

### 2. 后端 — `notebook_service.go`：Create/Update 加长度校验
- `Create(name)` 和 `Update(id, name)` 中新增 `utf8.RuneCountInString(name) > 50` 检查，返回错误提示

### 3. 后端 — `ai_service.go`：RenameAISession 加长度校验
- `RenameAISession(id, title)` 中新增 `utf8.RuneCountInString(title) > 50` 检查

### 4. 后端 — `ai_service.go`：自动标题截断到50字
- 3处自动标题生成（SaveAIMessage/SaveAIMessages/ReplaceAISessionMessages）恢复截断逻辑，改为50字

### 5. 后端 — `app.go`：App 层也可加前置校验（可选）
- CreateNotebook/RenameNotebook/RenameAISession 在调用 service 前校验

## 涉及文件
- `frontend/src/js/ai-chat.js` — maxlength
- `internal/services/notebook_service.go` — Create/Update 校验
- `internal/services/ai_service.go` — RenameAISession 校验 + 自动标题截断
- `app.go` — App 层前置校验

## 验证
1. 新建笔记本输入51字 → 后端拒绝 + 前端 toast 提示
2. 重命名笔记本输入51字 → 同上
3. 重命名AI会话输入51字 → 同上
4. 新建AI会话发50+字消息 → 标题自动截断为50字
