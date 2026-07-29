# 内置 API 服务商预设方案

## 一、需求概述

在设置页的 API 服务配置中预置常用服务商（DeepSeek、小米、智谱等），用户首次启动就能看到这些预设，只需填写 API Key 即可使用。预设定义在后端独立文件，方便新增和维护。

## 二、当前状态分析

- `APIProfile` 模型（[api_profile.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/api_profile.go)）已有字段：`ID`, `Name`, `Provider`, `BaseURL`, `APIKey`, `IsDefault`, `IsActive`, `CreatedAt`
- 数据库初始化（[db.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go)）已有种子数据模式：`InitBuiltinPrompts()` 和 `InitDefaultSettings()` 使用"检查存在性 → 仅插入缺失"的增量插入模式，可复用
- 服务层（[profile_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/profile_service.go)）提供 CRUD：`ListProfiles`, `CreateProfile`, `UpdateProfile`, `DeleteProfile`, `SwitchProfile`, `SetActive`
- 启动迁移（[app.go#L184-L225](file:///d:/峡谷/Dev/本地项目/jot/app.go#L184-L225)）处理"已有设置但无预设"的兜底创建逻辑
- 前端预设管理（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)）通过 `loadProfiles()` / `switchProfile()` 等函数渲染和操作预设

## 三、设计方案

### 总体思路

**不做模型改动，不新增字段。** 新建一个独立的 `builtin_profiles.go` 文件，专门存放内置服务商的定义。InitDB 时按 Name 去重插入即可。

为什么用 Name 去重就够了：DeepSeek、智谱 AI 这些名字不会和用户自建的普通预设撞车。用户删了某个内置预设也没关系，重启时 Name 不在库里就重新插入，相当于"自动恢复"。

### 3.1 新建内置服务商定义文件

**文件**: `internal/database/builtin_profiles.go`（新增）

集中存放所有内置服务商的 API 配置定义。用户在文件末尾按相同格式逐条追加即可。

```go
package database

import (
    "jot/internal/models"
    "gorm.io/gorm"
)

// InitBuiltinProfiles 增量插入内置 API 服务商预设（仅插入不存在的）
func InitBuiltinProfiles(db *gorm.DB) error {
    // 查询所有已有预设的名称
    var existingNames []string
    db.Model(&models.APIProfile{}).Pluck("name", &existingNames)
    existing := make(map[string]bool, len(existingNames))
    for _, n := range existingNames {
        existing[n] = true
    }

    builtinProfiles := []models.APIProfile{
        {
            Name: "DeepSeek", Provider: "openai",
            BaseURL: "https://api.deepseek.com",
        },
        {
            Name: "智谱 AI", Provider: "openai",
            BaseURL: "https://open.bigmodel.cn/api/paas/v4",
        },
        // ↓ 用户可在下面继续添加更多内置服务商 ↓
        // {Name: "XX", Provider: "openai", BaseURL: "https://api.xxx.com"},
    }

    var toInsert []models.APIProfile
    for _, p := range builtinProfiles {
        if !existing[p.Name] {
            toInsert = append(toInsert, p)
        }
    }
    if len(toInsert) == 0 {
        return nil
    }
    return db.Create(&toInsert).Error
}
```

要点说明：
- **APIKey 留空**：内置预设只预配 API 地址，不预填密钥
- **Provider 用 `"openai"`**：DeepSeek、智谱等都兼容 OpenAI API 协议，前端服务商控件默认也是这个值
- **不含 CreatedAt**：GORM 会自动填充零值时间戳

### 3.2 在 InitDB 中调用

**文件**: [internal/database/db.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go)

在 `InitDB` 函数的 `InitDefaultSettings` 之后追加调用：

```go
// 初始化内置 API 服务商预设
if err := InitBuiltinProfiles(db); err != nil {
    return nil, fmt.Errorf("初始化内置 API 预设失败: %w", err)
}
```

### 3.3 无需改动的部分

以下文件**无需任何修改**：

| 文件 | 原因 |
|------|------|
| `internal/models/api_profile.go` | 不新增字段，现有模型完全够用 |
| `internal/services/profile_service.go` | 内置预设不设特殊保护，删除后重启自动恢复 |
| `app.go` | 启动迁移逻辑在有预设时正常跳过，不受影响 |
| `frontend/src/main.js` | 内置预设与普通预设共享同一数据结构，前端自动兼容 |
| `frontend/index.html` | 无 UI 结构变更 |

## 四、完整的文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/database/builtin_profiles.go` | **新增** | 内置服务商定义文件，用户在此添加预设 |
| `internal/database/db.go` | 修改 | 在 `InitDB` 中调用 `InitBuiltinProfiles` |

仅涉及 2 处变更，极其轻量。

## 五、验证步骤

1. 删除 `api_profiles` 表或清空数据，重启应用，确认内置预设自动出现在预设下拉中
2. 切换到某个内置预设（如 DeepSeek），确认 API 地址自动填入，API Key 为空
3. 填写 API Key 并保存，重启后确认数据不丢失
4. 删除某个内置预设，重启应用，确认预设自动恢复
5. 在 `builtin_profiles.go` 中新增一条定义，重启应用，确认新预设自动出现
