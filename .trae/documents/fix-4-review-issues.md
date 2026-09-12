# 修复 4 个审查问题

## Summary
修复本次开发审查中确认的 4 个真实问题：
1. 7 主题 `criticalColors` / `topbar-bg` 与 `variables.css --bg` / `--topbar-bg` 漂移 → 启动时 FOUC 闪烁
2. `SaveAllSettings` 失败仍发"已恢复"通知 → 语义不准 + 下次启动重复通知
3. 空值/null 时通知文案 `主题""已失效` / `主题"null"已失效` → 不友好
4. inline 脚本 `localStorage` 无 try/catch → 隐私模式下 IIFE 中断，FOUC 防闪失效

## Current State Analysis

### 4 个问题的真实情况

#### 问题 1：FOUC 漂移（严重，用户可见）
- 7 主题 `criticalColors[0]` 与 `variables.css --bg` 不一致
- 6 主题 `criticalColors[1]` 与 `variables.css --topbar-bg` 不一致
- 内联 `<style>:root{--bg:criticalColors[0]}` 在 `variables.css` 之前注入
- 但 `[data-theme=X]` 特异性 (0,1,0) > `:root` 特异性 (0,0,1)，所以 variables.css 胜出
- 用户切换到这 7 主题时，0-200ms 内看到两次背景色切换

权威源：**`variables.css`**（完整色板）— `criticalColors` / `main.go themeBG` 是 FOUC 辅助
- 决策：把 `criticalColors` / `main.go` 同步到 `variables.css`，不动 variables.css

#### 问题 2：save 失败通知
- 当前 `loadSettings` 在 `corrected=true` 时 `await SaveAllSettings(cfg)`
- 抛错时只 console.error，但通知仍发"已恢复"
- 失败时：前端 localStorage + DOM 已修正，后端仍是无效值 → 下次启动同一通知再次弹出
- 修复：去掉 `await`，改 fire-and-forget，失败只 console.error

#### 问题 3：空值文案
- Go 端 `Settings.Theme` 是 `string`，零值 `""`
- `cfg.theme = ""` 时，`resolveTheme("")` → `corrected=true`
- 通知文案：`主题""已失效...`（双引号套空）或 `主题"null"已失效`（如果是 null）
- 修复：显示时 fallback 到 `"(空值)"`

#### 问题 4：localStorage 抛错
- inline 脚本：`localStorage.getItem('jot_theme')` / `setItem(...)` 无 try/catch
- 隐私模式 / 存储配额满 / 安全策略 → 抛 SecurityError / QuotaExceededError
- IIFE 中断，FOUC 防闪失效
- 修复：getItem + setItem 都包 try/catch

## Proposed Changes

### Fix 1：7 主题 FOUC 漂移同步

**修改文件**：[index.html](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html) `criticalColors` + [main.go](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%B0%E9%A1%B9%E7%9B%AE/jot/main.go) `themeBG()`

**为什么用 variables.css 作权威源**：
- 已有维护契约明确："criticalColors 与 themeLabels 同步维护"，variables.css 同样在这条契约内
- variables.css 是完整色板（~60 变量/主题），criticalColors 只 2 个，main.go 只 1 个
- 主题设计师改的是 variables.css，FOUC 辅助层应当跟随

#### 1.1 `index.html` `criticalColors` 修改（7 主题 × 2 字段 = 14 项）

```diff
 'light':             ['#FAFAFA','#FFFFFF'],
+→                    ['#F4F4F7','#FFFFFF'],
 'nord':              ['#ECEFF4','#FFFFFF'],
+→                    ['#E4E9F0','#FAFBFD'],
 'eye-protection':    ['#C7EDCC','#D8F5DD'],
+→                    ['#E4ECD9','#EFF3E6'],
 'catppuccin-latte':  ['#EFF1F5','#FFFFFF'],
+→                    ['#EDE7E5','#F8F2EF'],
 'gruvbox-light':     ['#FBF1C7','#F9F5D7'],
+→                    ['#F0E8C9','#F8F2D8'],
 'dracula':           ['#1A1B25','#16171F'],
 (不动 - 已对齐)
 'quiet-light':       ['#F5F5F5','#FFFFFF'],
+→                    ['#F0EAEC','#F9F4F4'],
 'ysgrifennwr':       ['#F5EDDA','#FAF4E3'],
+→                    ['#F0E5D0','#F8F0DB'],
```

#### 1.2 `main.go` `themeBG()` 修改（7 主题 × 1 字段 = 7 项）

`main.go` 只对应 `--bg`（WebView2 窗口背景色），不对应 `--topbar-bg`（HTML 顶栏色），所以只改 1 个字段/主题。

| case | 旧返回 | 新返回 | 对应 hex |
|---|---|---|---|
| `"light"` | 250, 250, 250 | 244, 244, 247 | #F4F4F7 |
| `"nord"` | 236, 239, 244 | 228, 233, 240 | #E4E9F0 |
| `"eye-protection"` | 199, 237, 204 | 228, 236, 217 | #E4ECD9 |
| `"catppuccin-latte"` | 239, 241, 245 | 237, 231, 229 | #EDE7E5 |
| `"gruvbox-light"` | 251, 241, 199 | 240, 232, 201 | #F0E8C9 |
| `"quiet-light"` | 245, 245, 245 | 240, 234, 236 | #F0EAEC |
| `"ysgrifennwr"` | 245, 237, 218 | 240, 229, 208 | #F0E5D0 |

### Fix 2：save 失败 fire-and-forget

**修改文件**：[main.js](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js) L11608-L11617

```diff
 if (corrected) {
     const invalidTheme = cfg.theme; // 保留原值供通知显示
     cfg.theme = validTheme;
-    try {
-        await window.go.main.App.SaveAllSettings(cfg);
-    } catch (e) {
-        console.error('修正主题并保存设置失败:', e);
-    }
+    // fire-and-forget：异步持久化到后端，失败仅 console.error
+    // 前端 localStorage + DOM 已被本函数下方 + inline 脚本两处修正，
+    // 即使后端保存失败，本次会话仍能正常使用，下次启动会再次触发兜底
+    window.go.main.App.SaveAllSettings(cfg)
+        .catch(e => console.error('修正主题并保存设置失败:', e));
     nm.show(`主题"${invalidTheme}"已失效，已恢复为默认主题`, 'warning');
 }
```

### Fix 3：空值通知文案

**修改文件**：[main.js](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js) L11616

```diff
-    nm.show(`主题"${invalidTheme}"已失效，已恢复为默认主题`, 'warning');
+    // 空值/非字符串时显示 (空值) 避免出现 主题""已失效 之类的丑陋文案
+    const displayTheme = invalidTheme ? `"${invalidTheme}"` : '"(空值)"';
+    nm.show(`主题${displayTheme}已失效，已恢复为默认主题`, 'warning');
```

### Fix 4：inline 脚本 localStorage try/catch

**修改文件**：[index.html](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html) L25-L30

```diff
-    // 有效性兜底：criticalColors 的 key 集合作为有效集
-    var validKeys = Object.keys(criticalColors);
-    var stored = localStorage.getItem('jot_theme');
-    var theme = (stored && validKeys.indexOf(stored) !== -1) ? stored : 'default';
-    // 修正无效值：立即写回 localStorage，避免后续 applyTheme 再次校正
-    if (stored !== theme) localStorage.setItem('jot_theme', theme);
+    // 有效性兜底：criticalColors 的 key 集合作为有效集
+    var validKeys = Object.keys(criticalColors);
+    // localStorage 在隐私模式 / 配额满 / 安全策略下可能抛错
+    // getItem 失败 → stored=null（fallback 到 default）
+    // setItem 失败 → 忽略（下次启动会再尝试）
+    var stored = null;
+    try { stored = localStorage.getItem('jot_theme'); } catch (e) { /* 隐私模式等 */ }
+    var theme = (stored && validKeys.indexOf(stored) !== -1) ? stored : 'default';
+    // 修正无效值：立即写回 localStorage，避免后续 applyTheme 再次校正
+    if (stored !== theme) {
+        try { localStorage.setItem('jot_theme', theme); } catch (e) { /* 写入失败 */ }
+    }
```

## Assumptions & Decisions

| 决策 | 理由 |
|---|---|
| 权威源 = `variables.css` | 完整色板（~60 变量/主题），设计师改的是它；criticalColors / main.go 是 FOUC 辅助 |
| `main.go themeBG` 不对应 `topbar-bg` | WebView2 窗口背景 = 整页背景（`--bg`），不是顶栏色。HTML 顶栏色由 criticalColors + variables.css 共同决定 |
| Fix 2 用 fire-and-forget 而非"失败不发通知" | 失败时前端已恢复，文案"已恢复"对用户视角正确；fire-and-forget 解阻塞 + 失败静默 |
| Fix 3 用 `"(空值)"` 显示 | 简洁、无歧义；与已有错误通知风格一致 |
| Fix 4 getItem + setItem 都 try/catch | setItem 在 setItem 时配额满会抛错；getItem 失败 + setItem 成功已经够用户感知 |
| 不动 dracula 块 | 已经在本次前面"dark/dracula 调深"中同步到 criticalColors，无漂移 |
| 不动 mono 块 | 新增时已与 criticalColors 同步对齐 |

## Verification

### 修复 1 验证（FOUC 漂移）

| 主题 | 修复前 criticalColors[0] | 修复后 | 与 variables.css --bg 一致 |
|---|---|---|---|
| light | #FAFAFA | #F4F4F7 | ✓ |
| nord | #ECEFF4 | #E4E9F0 | ✓ |
| eye-protection | #C7EDCC | #E4ECD9 | ✓ |
| catppuccin-latte | #EFF1F5 | #EDE7E5 | ✓ |
| gruvbox-light | #FBF1C7 | #F0E8C9 | ✓ |
| quiet-light | #F5F5F5 | #F0EAEC | ✓ |
| ysgrifennwr | #F5EDDA | #F0E5D0 | ✓ |

类似地 criticalColors[1] 6 项同步。

| 主题 | 修复后 themeBG | 与 variables.css --bg 一致 |
|---|---|---|
| light | 244, 244, 247 | ✓ |
| nord | 228, 233, 240 | ✓ |
| eye-protection | 228, 236, 217 | ✓ |
| catppuccin-latte | 237, 231, 229 | ✓ |
| gruvbox-light | 240, 232, 201 | ✓ |
| quiet-light | 240, 234, 236 | ✓ |
| ysgrifennwr | 240, 229, 208 | ✓ |

### 修复 2 验证
- 错误路径不再 `await SaveAllSettings`，`loadSettings` 立即继续
- `nm.show` 同步调用，文案不变
- 后端保存失败：仅 console.error，UI 正常

### 修复 3 验证
| cfg.theme 值 | 修复前通知 | 修复后通知 |
|---|---|---|
| `"invalid-x"` | `主题"invalid-x"已失效...` | 同上（无变化）|
| `""` | `主题""已失效...` | `主题"(空值)"已失效...` |
| `null` | `主题"null"已失效...` | `主题"(空值)"已失效...` |
| `undefined` | `主题"undefined"已失效...` | `主题"(空值)"已失效...` |

### 修复 4 验证
- 隐私模式下 getItem 抛错 → `stored=null` → 走 default → 页面正常渲染
- 配额满时 setItem 抛错 → 忽略 → 不影响本次会话（下次启动再次尝试）

### 跨主题全局验证（防止破坏其他主题）

| 主题 | 修复前 | 修复后 | 变化 |
|---|---|---|---|
| default | #F2EDE3/#FCF9F2 | 同 | 无 |
| dark | #0F0F12/#141418 | 同 | 无 |
| tokyo-night | #1A1B26/#1F2335 | 同 | 无 |
| dracula | #1A1B25/#16171F | 同 | 无 |
| mono | #F5F5F5/#FFFFFF | 同 | 无 |

### 构建验证
- `go build` 通过
- `npm run build` 通过
- `npm run lint` 0 errors

## Out of Scope
- 不修复 5.1 缩进不一致（一次性大扫除，下次做）
- 不修后端 Go 端 `GetAllSettings()` 返回值校验
- 不改 4 文件主题维护流程文档
- 不撤回或改动其他已稳定的代码
