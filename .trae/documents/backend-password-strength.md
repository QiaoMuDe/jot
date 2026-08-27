# 后端密码强度检测功能

## Summary

在后端集成 `github.com/trustelem/zxcvbn` 库，新增 `CheckPasswordStrength` Wails 绑定方法，暴露给前端调用，替换前端自编的 `pmCalcStrength` 逻辑。

## Current State

* 前端 [password-manager.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/password-manager.js#L213-L258) 中的 `pmCalcStrength` 是自定义的 0-4 评级算法

* 该函数仅在密码生成器结果展示时调用（[第287行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/password-manager.js#L287)），不作为保存时的强制校验

* 后端 [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L3797-L3876) 已有完整密码 CRUD 绑定方法

* 后端目前无任何密码强度检测逻辑

* `trustelem/zxcvbn` 返回 0-4 评分，与前端现有体系一致

## Proposed Changes

### Step 1: 添加依赖

```bash
go get github.com/trustelem/zxcvbn
```

### Step 2: 在 `app.go` 新增绑定方法

在 [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go) 的密码相关方法区域（第3875行附近）新增：

```go
// PasswordStrengthResult 密码强度检测结果
type PasswordStrengthResult struct {
    Score     int     `json:"score"`      // 0-4 强度评级
    Entropy   float64 `json:"entropy"`    // 信息熵（位）
    CrackTime float64 `json:"crackTime"`  // 预估破解时间（秒）
    CrackTimeDisplay string `json:"crackTimeDisplay"` // 人类可读的破解时间
}

// CheckPasswordStrength 检测密码强度（0-4 评级）
func (a *App) CheckPasswordStrength(password string) *PasswordStrengthResult {
    result := zxcvbn.PasswordStrength(password, nil)
    return &PasswordStrengthResult{
        Score:            result.Score,
        Entropy:          result.Entropy,
        CrackTime:        result.CrackTime,
        CrackTimeDisplay: result.CrackTimeDisplay,
    }
}
```

* 在文件顶部 `import` 中添加 `"github.com/trustelem/zxcvbn"`

* 新增 `PasswordStrengthResult` 结构体（定义在 `app.go` 中，Wails 绑定需要结构体在 `main` 包中）

### Step 3: 前端替换调用

在 [password-manager.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/password-manager.js) 中：

1. 将 `pmDoGenerate` 函数（第279行起）中的同步 `pmCalcStrength(pwd)` 调用改为异步调用后端：

```js
// 原：
const strength = pmCalcStrength(pwd);

// 改为：
const strengthResult = await window.go.main.App.CheckPasswordStrength(pwd);
const strength = strengthResult.score;
```

1. `pmDoGenerate` 函数需要改为 `async function`

2. **删除** `pmCalcStrength` 函数（不再需要前端自编逻辑）

## Files to Modify

| 文件                                    | 改动                                                                         |
| ------------------------------------- | -------------------------------------------------------------------------- |
| `go.mod` / `go.sum`                   | `go get` 自动更新                                                              |
| `app.go`                              | 新增 import + `PasswordStrengthResult` 结构体 + `CheckPasswordStrength` 方法      |
| `frontend/src/js/password-manager.js` | `pmDoGenerate` 改为 async，调用后端方法；删除 `pmCalcStrength` 函数和 `PM_KB_PATTERNS` 常量 |

## Assumptions

* `trustelem/zxcvbn` 的 `CrackTimeDisplay` 字段可直接用作 UI 展示（如 "3 小时"、"100 年" 等）

* 前端 `pmCalcStrength` 完全删除，不再保留

* 密码生成器一次生成多个密码时，逐个调用后端检测强度（Wails 调用足够快，无需批量接口）

## Verification

1. `go build .` 编译通过
2. 启动应用，在密码生成器中生成密码，确认强度指示点正常显示
3. 测试弱密码（如 `123456`）应返回 score=0，强密码（如 `Kj#9xL!mQ2@v`）应返回 score=3 或 4

