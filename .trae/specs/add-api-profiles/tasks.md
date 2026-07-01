# Tasks

- [ ] Task 1: 后端 — APIProfile 数据模型 + 自动迁移
  - 新建 `internal/models/api_profile.go`，定义 `APIProfile` 结构体
  - 在 `main.go` 或 `database.go` 的自动迁移列表中加入 `&models.APIProfile{}`
  - 规格：ID（自增）、Name（varchar 50）、Provider（varchar 20）、BaseURL（varchar 200）、APIKey（varchar 200）、CreatedAt

- [ ] Task 2: 后端 — ProfileService CRUD + SwitchProfile
  - 新建 `internal/services/profile_service.go`
  - `ListProfiles()`：返回所有预设，按 `created_at` 降序
  - `CreateProfile(name, provider, baseURL, apiKey)`：创建预设，返回 `APIProfile`
  - `UpdateProfile(id, name, provider, baseURL, apiKey)`：更新预设
  - `DeleteProfile(id)`：删除预设
  - `SwitchProfile(id)`：读取预设 → 调用 `SettingService.Set()` 写入 `ai_provider`、`ai_base_url`、`ai_api_key`

- [ ] Task 3: 后端 — App 绑定 + 启动迁移
  - 在 `app.go` 中新增绑定：
    - `GetProfiles()` → `[]services.APIProfile`
    - `CreateProfile(name, provider, baseURL, apiKey)` → `services.APIProfile`
    - `UpdateProfile(id, name, provider, baseURL, apiKey)` → 无返回值
    - `DeleteProfile(id)` → 无返回值
    - `SwitchProfile(id)` → 无返回值（调用 ProfileService.SwitchProfile）
  - 启动迁移兼容：在 `App` 初始化或 `startup()` 中，若 `api_profiles` 表为空且 settings 表中有 `ai_base_url` 非空，自动创建一条"默认配置"预设

- [ ] Task 4: 前端 — 设置页 HTML 结构
  - 在 `index.html` 的「API 连接」区域内，group-header 下方新增「配置预设」行：
    - `.preset-select` 下拉选择器 + 标签 `配置预设`
    - `[+ 新增]` 按钮
    - `[管理]` 按钮
    - 备注小字提示区
  - 新增/编辑弹窗 HTML（模态表单）：名称、服务商（下拉）、API 地址、API Key，保存/取消按钮

- [ ] Task 5: 前端 — CSS 样式
  - 在 `settings-panel.css` 中新增样式：
    - `.preset-select-row`：预设选择器整行布局
    - `.preset-select`：预设下拉样式（复用 `.theme-select` 风格）
    - `.preset-modal-overlay` / `.preset-modal`：新增编辑弹窗样式
    - `.preset-hint`：小字提示样式

- [ ] Task 6: 前端 — 预设管理 JS 逻辑
  - 在 `main.js` 中增加：
    - `loadProfiles()`：调用 `GetProfiles()`，渲染预设下拉列表 + 标记当前使用项
    - `switchProfile(id)`：调用 `SwitchProfile(id)` → 刷新当前配置输入框 → 重置模型下拉
    - `openAddProfileModal()` / `openEditProfileModal(id)`：弹窗表单，保存时调用 CreateProfile / UpdateProfile
    - `deleteProfile(id)`：确认后调用 DeleteProfile，刷新列表
    - 整合到 `loadAISettings()` 流程中：初始化时加载预设列表
  - 弹窗的表单校验：名称必填，服务商必选，API 地址必填

- [ ] Task 7: 联调验证
  - 启动应用，验证自动迁移：`api_profiles` 表已创建
  - 验证默认配置自动创建：已有配置的用户首次打开，应出现"默认配置"预设
  - 验证新增/编辑/删除预设 CRUD 正常
  - 验证切换预设：URL/Key/服务商自动填充并保存到 settings 表
  - 验证手动修改输入框不影响预设

# Task Dependencies

- Task 4（HTML 结构）需在 Task 3 之后（前端需要后端绑定可用）
- Task 5（CSS）可和 Task 4 并行
- Task 6（JS 逻辑）需在 Task 4 + Task 5 之后
- Task 7（联调）在所有 Task 完成后
