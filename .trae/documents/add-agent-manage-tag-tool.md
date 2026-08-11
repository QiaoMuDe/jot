# manage\_tag 标签管理工具实现计划

## 摘要

为 Agent 新增 `manage_tag` 工具，支持两个动作：**列出全部标签**、**新建标签**（name 必填、color 可选）。复用现有 `services.TagService`（`GetAll` / `Create`），仅需新增一个按名称查重的方法。装配链路与 manage\_todo / manage\_notebook 完全一致，并顺带修复 `rebuildServices` 不重建 `AgentSvc` 的既有问题（否则数据库重置后所有管理类工具都会悬挂旧服务实例）。

## 现状分析

* [tag\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/tag_service.go) 已有 `GetAll()`（created\_at ASC）与 `Create(name, color)`（color 空默认 `#3b82f6`），足够支撑 list / create 两个动作；`Tag.Name` 有 `uniqueIndex`，重复插入会触发 SQLite 唯一约束错误（英文），需要友好查重。

* [tag.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/tag.go)：Tag{ID, Name(唯一), Color, CreatedAt}，无软删除。

* 前端通过 `GetAllTags` 展示标签（Tag 数组，不含笔记数），工具返回纯文本，前端无需改动。

* 现有工具装配模式（manage\_todo / manage\_notebook）：`Deps` 加字段 → `app.go` 构造 `NewAgentService` 处传入 → `registry.go` `buildTools` 注册（`WrapWithError` 包装）。

* **既有问题**：[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L3933-L3957) `rebuildServices` 重建各服务后**未重建** **`app.AgentSvc`**，`AgentSvc` 内 Deps 仍指向旧服务实例（旧 `gorm.DB` 连接）。数据库重置后调用 manage\_todo / manage\_notebook / 新 manage\_tag 会操作旧实例，属潜在悬挂问题，本次一并修复。

## 变更内容

### 1. `internal/services/tag_service.go`：新增 `GetByName` 查重方法

在 `Count` 之后新增：

```go
// GetByName 按名称精确查找标签（名称唯一），未找到返回 gorm.ErrRecordNotFound。
func (s *TagService) GetByName(name string) (*models.Tag, error)
```

实现：`s.db.Where("name = ?", name).First(&tag)`，出错时打 `Errorw` 日志后返回。不改动现有 `Create`（避免影响前端创建标签的调用方）。

### 2. `internal/agent/tools/manage_tag.go`：新建工具（照抄 manage\_todo.go 骨架）

* 结构体 `manageTagTool{ tag *services.TagService; ctx *Context }` + `var _ tool.InvokableTool = (*manageTagTool)(nil)`

* `NewManageTag(tag *services.TagService, ctx *Context) tool.InvokableTool` 构造器

* `Info()`：

  * Name: `manage_tag`

  * Desc: 管理标签。当用户要求创建标签或查看标签列表时调用。action=create 创建标签（name 必填，color 可选，格式 #RRGGBB，缺省 #3b82f6）；action=list 列出全部标签。

  * 参数：`action`（String，Enum `["create","list"]`，必填）、`name`（String，可选）、`color`（String，可选）

* `InvokableRun`：

  * 解析 args（Action/Name/Color），action 白名单校验

  * `ctx.Err()` 取消检查；`m.ctx.Logger.Debugw` 记录 action/name/color

  * **create**：name trim 非空校验 → color 格式校验（`^#[0-9a-fA-F]{6}$`，空则跳过用默认）→ `GetByName` 查重：

    * 已存在 → 返回文本 `标签「X」已存在（编号 [id]），无需重复创建`（非错误，模型直接停止）

    * 不存在 → `Create`，成功返回 `已创建标签「X」（编号 [id]）`

  * **list**：`GetAll` → 空返回 `当前没有任何标签`；有则 `当前标签列表（共 n 个）：` 逐条 `[id] 名称 · 颜色`（created\_at ASC，不分页——标签规模小且名称唯一）

* 文件头注释：职责 + 两动作说明 + 实现要点（照抄 manage\_todo.go 风格）

### 3. 装配链路

* [agent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L44-L53)：`Deps` 增加 `Tag *services.TagService // Tag 标签服务（manage_tag 工具使用）`

* [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L184-L192)：`NewAgentService` 的 `Deps` 增加 `Tag: tagService`（`tagService` 变量已在 L164 创建）

* [registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go#L26)：`buildTools` 追加 `tools.WrapWithError("manage_tag", tools.NewManageTag(p.deps.Tag, p.ctx), p.ctx),`

### 4. `app.go` `rebuildServices`：重建 `AgentSvc`（修复既有悬挂问题）

在 `rebuildServices` 末尾（`LogSvc` 重建成功后）追加重建 `AgentSvc`，将所有服务的最新实例与 `GetEmbedConfig` 重新注入，保证数据库重置后管理类工具（含新增 manage\_tag）操作的是新实例。注意：`a.ctx`、图片目录等初始化与重建顺序无关，保持现有逻辑不动。

### 5. 文档

* [TOOLS.md](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/TOOLS.md#L306) §6 表格追加一行：
  `| manage_tag | manage_tag.go | tag、ctx | 列出全部标签 / 创建标签（name 必填，color 可选） | 无 |`

* [tools/doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go#L4-L5)：工具清单与构造器名补 `manage_tag` / `NewManageTag`

* [ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2555-L2564) `showToolStatusStart` 增加 manage\_tag 分支：list→"列出标签"、create→"创建标签"（与 manage\_todo / manage\_notebook 一致的动作文案）

* `agent/doc.go`：与 manage\_todo / manage\_notebook 处理一致，清单表述维持现状（该处仍写"只读工具"，历史一致性问题不额外扩大）

## 假设与决策

1. **动作范围仅 list + create**：用户明确"列出标签，新建标签"。重命名 / 删除 / 改色等不在本次范围（TagService 已有 `Update` / `Delete`，后续按需扩展）。
2. **color 对模型可选**：模型一般不知道合法色值，不传走默认蓝色；传了则严格校验 `#RRGGBB`，非法直接报错回填。
3. **重名处理为提示而非错误**：`GetByName` 查到即返回"已存在（编号 \[id]）"文本，模型停止，避免触发 `WrapWithError` 的失败回填。
4. **list 不分页、不带笔记数**：标签规模小（几十个封顶，名称唯一），前端展示也不带笔记数；保持一致，不做过度设计。后续若需要可按 manage\_todo 模式扩展。
5. **`rebuildServices`** **修复纳入本次**：否则 manage\_tag（及既有 manage\_todo / manage\_notebook）在数据库重置后会操作旧实例，工具不稳定。属于本工具的必需配套修复。
6. **不新增 TagService 之外的服务方法**：`GetByName` 是唯一新增，`Create` / `GetAll` 直接复用。

## 验证步骤

1. `go build ./...`、`go vet ./internal/agent/...`、`gofmt -l internal/agent/tools internal/services` 全部通过
2. `rebuildServices` 编译通过，确认其中重建 `AgentSvc` 引用的字段（`tagService` 等）均在作用域内
3. 重启应用（Wails 需重新 `wails build` 后端生效），Agent 模式下触发：

   * "列出所有标签" → 返回默认标签列表（待办/工作/生活/个人/学习/重要）

   * "新建一个叫「读书」的标签" → 成功并返回编号

   * 重复创建同名标签 → 返回"已存在"提示
4. 数据库重置后再次调用 manage\_tag，确认操作新实例（修复项生效）

