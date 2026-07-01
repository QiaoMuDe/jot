# API 配置预设 Spec

## Why

用户经常在多个 API 服务之间切换（如 DeepSeek、通义千问、本地 Ollama），每次需要手动修改 URL、Key、服务商三个字段，且模型列表需要重新获取。配置预设功能让用户一次性保存多组 API 配置，一键切换，无需重复录入。

## What Changes

### 后端

- 新增 `APIProfile` 模型，新建 `api_profiles` 数据库表
- 新增 `ProfileService` 的 CRUD 方法（List/Create/Update/Delete）
- 新增 `SwitchProfile` 方法：选中某个预设时，将其 URL+Key+Provider 写入现有 settings 表
- 新增 Wails 绑定方法暴露给前端

### 前端

- 设置页「API 连接」区域顶部新增「配置预设」管理区
- 下拉选择器切换预设 + 新建/编辑/删除操作
- 切换预设时自动填充 URL/Key/服务商 并保存到当前配置
- 现有输入框和保存逻辑保持不变

### 不改变

- AIConfig 结构体不变
- 现有 settings 表的 ai_base_url / ai_api_key / ai_provider / ai_model 读写逻辑不变
- AI 对话/流式调用/测试连接/获取模型列表 等流程不变
- Tavily API Key 保持全局独立，不纳入预设

## Impact

- Affected specs: 设置页 AI 配置区域
- Affected code:
  - `internal/models/api_profile.go` — 新文件
  - `internal/services/profile_service.go` — 新文件  
  - `app.go` — 新增 Wails 绑定
  - `frontend/index.html` — 设置页新增 UI
  - `frontend/src/main.js` — 新增前端的预设管理逻辑
  - `frontend/src/css/components/settings-panel.css` — 新增样式

## ADDED Requirements

### Requirement: API Profile 数据模型

APIProfile 结构体：
- ID（自增主键）
- Name（用户自定义名称，如"DeepSeek"、"本地 Ollama"，长度 ≤50）
- Provider（"openai" / "ollama"，长度 ≤20）
- BaseURL（API 地址，长度 ≤200）
- APIKey（API Key，长度 ≤200）
- CreatedAt（创建时间）
- 表名：`api_profiles`

#### Scenario: 创建预设
- **WHEN** 用户填写 Name + Provider + BaseURL + APIKey 并确认
- **THEN** 新增一条 `api_profiles` 记录，返回成功

#### Scenario: 查看预设列表
- **WHEN** 打开设置页或刷新列表
- **THEN** 前端获取所有 `api_profiles` 记录，按创建时间排序展示

#### Scenario: 编辑预设
- **WHEN** 用户修改已有预设的任意字段并保存
- **THEN** 数据库中对应记录更新

#### Scenario: 删除预设
- **WHEN** 用户删除某个预设
- **THEN** 从 `api_profiles` 表中删除该记录

#### Scenario: 切换预设（核心功能）
- **WHEN** 用户选中一个预设
- **THEN** 后端将该预设的 Provider → `ai_provider`、BaseURL → `ai_base_url`、APIKey → `ai_api_key` 写入 settings 表，前端自动刷新输入框的值，模型下拉重置

### Requirement: 前端 UI

设置页「API 连接」区域顶部，在 group-header 下方新增「配置预设」行：

```
┌──────────────────────────────────────────────┐
│  API 连接                                     │
│  ┌──────────────────────────────────────────┐ │
│  │  配置预设  [当前预设 ▼]  [+ 新增] [管理]  │ │
│  └──────────────────────────────────────────┘ │
│  │  API 地址    [input ──── 自动填充 ────]    │ │
│  │  服务商      [下拉 ──── 自动切换 ────]     │ │
│  │  API Key     [input ──── 自动填充 ────]    │ │
│  │  模型        [下拉]         [获取列表]      │ │
│  └──────────────────────────────────────────┘  │
└────────────────────────────────────────────────┘
```

#### Scenario: 预设选择器
- 下拉列表显示所有预设的名称
- 选中后立即触发切换
- 当前正在使用的预设显示选中态
- 无预设时显示"无预设配置"

#### Scenario: 新增预设
- 点击「+ 新增」打开内联表单（或轻量弹窗）
- 字段：名称、服务商（下拉）、API 地址、API Key
- 保存后自动加入下拉列表并切换到该预设

#### Scenario: 管理预设（可选轻量）
- 点击「管理」打开预设列表视图（或在当前区域扩展）
- 每条预设显示：名称、服务商、API 地址（Key 脱敏显示）
- 提供编辑和删除按钮
- 编辑时弹出和内联新增同样的表单

### Requirement: 迁移兼容

- **启动时检测**：如果 `api_profiles` 表为空且 settings 表中存在有效的 ai_base_url，则自动创建一条名为"默认配置"的预设
- 不影响已有用户的现有配置

## MODIFIED Requirements

### Requirement: 现有设置页 AI 配置输入框

URL/Key/服务商输入框的占位符/提示文字不变，新增行为：
- **WHEN** 切换预设后，输入框的值被自动填充并自动保存
- **WHEN** 用户手动修改输入框后，这些修改仅影响当前配置，不影响已保存的预设
- 输入框下方可加一行小字提示："手动修改仅影响当前配置，不会保存到预设"

## REMOVED Requirements

（无删除项）
