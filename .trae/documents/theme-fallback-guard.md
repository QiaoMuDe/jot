# 主题名有效性兜底机制

## Summary
在主题加载链路中加入"有效性校验 + 自动回退 + 通知告知"三件套。任何从 localStorage / 后端配置读出的主题名，只要不在 `themeLabels` 有效清单内，就替换为 `default`，用 default 应用，并通过 warning 通知告知用户。

## Current State Analysis

### 主题加载 3 个入口（全部无校验）
| 入口 | 位置 | 行为 |
|---|---|---|
| ① FOUC 防闪内联脚本 | [index.html L7-33](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html#L7-L33) | 读 `localStorage.jot_theme` 后**无校验**地 `setAttribute('data-theme', theme)`。CSS 加载后若无匹配的主题块，所有颜色变量丢失 |
| ② applyTheme | [main.js L1753-1775](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L1753-L1775) | 同样无校验直接 setAttribute。还有 `data-theme === themeName` 早返回逻辑（**校验必须放在它之前**） |
| ③ loadSettings | [main.js L11600-11606](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L11600-L11606) | 从后端 `GetAllSettings()` 读 `cfg.theme` → 写 localStorage → applyTheme。两端都无校验 |

### 关键资源
- **有效名单源头**：`themeLabels`（[theme-config.js L2-16](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/theme-config.js#L2-L16)）— 现有维护流程保证其权威性
- **criticalColors**：index.html 内联脚本的"备份清单"（[L15-28](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/index.html#L15-L28)），key 集合与 `themeLabels` 同步，可作为内联脚本的有效集来源
- **通知 API**：`nm.show(msg, type, duration)` — main.js L431 实例化；现有调用样例 L1844 `nm.show('主题设置已保存', 'success')`
- **存储双端**：localStorage `jot_theme`（前端缓存）+ 后端 `cfg.theme`（持久化）

### 早返回陷阱
applyTheme 当前有早返回逻辑：
```js
if (document.documentElement.getAttribute('data-theme') === themeName) return;
```
若 `themeName = "bogus"` 且当前 data-theme 也是 `"bogus"`，会直接返回，导致修正永远不发生。**校验必须放在早返回之前**。

## Proposed Changes

### 1. `frontend/src/js/theme-config.js` — 新增 `resolveTheme` 纯函数
在文件底部追加导出：

```js
/**
 * 校验主题名是否在 themeLabels 有效清单中
 * @param {string} themeName
 * @returns {{ name: string, corrected: boolean }} 校正后的主题名 + 是否发生了修正
 */
export function resolveTheme(themeName) {
    const valid = themeName && Object.prototype.hasOwnProperty.call(themeLabels, themeName);
    return { name: valid ? themeName : 'default', corrected: !valid };
}
```

**为什么用 `hasOwnProperty`**：防御 `themeName` 是 `__proto__` / `constructor` 等原型键时的污染。

### 2. `frontend/src/main.js` `applyTheme`（L1753-1775）— 校验 + 通知 + 写回
在函数最开头、`syncThemeUI` 调用之前插入 resolveTheme 逻辑：

```js
function applyTheme(themeName) {
    // 校验 + 兜底：无效主题名替换为 default，并同步落库 + 通知
    const { name, corrected } = resolveTheme(themeName);
    if (corrected) {
        localStorage.setItem('jot_theme', name);
        saveSettings(); // 静默持久化到后端
        nm.show(`主题"${themeName}"已失效，已恢复为默认主题`, 'warning');
    }
    themeName = name;

    // ↓↓↓ 以下保持原逻辑不变 ↓↓↓
    syncThemeUI(themeName);

    if (document.documentElement.getAttribute('data-theme') === themeName) return;

    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const apply = () => {
        document.documentElement.setAttribute('data-theme', themeName);
        updateCodeHighlightThemePairing(themeName);
    };
    if (!reduced && document.startViewTransition) {
        document.startViewTransition(apply);
    } else {
        apply();
    }
}
```

**关键点**：
- 校验在 `syncThemeUI` 和早返回**之前**，确保下拉菜单高亮也同步到 default
- `saveSettings()` 自身不发通知（已确认 L1844 模式：saveSettings 静默 + 单独调 nm.show），与我们的 warning 通知无冲突
- `nm` 在 L431 module 顶部实例化，loadSettings 调用时已就绪

### 3. `frontend/index.html` 内联脚本（L7-33）— FOUC 防闪 + 立即写回
将内联脚本改为：

```js
(function() {
    var criticalColors = {
        'default':           ['#F2EDE3','#FCF9F2'],
        'light':             ['#FAFAFA','#FFFFFF'],
        'dark':              ['#0F0F12','#141418'],
        'nord':              ['#ECEFF4','#FFFFFF'],
        'tokyo-night':       ['#1A1B26','#1F2335'],
        'catppuccin-latte':  ['#EFF1F5','#FFFFFF'],
        'gruvbox-light':     ['#FBF1C7','#F9F5D7'],
        'dracula':           ['#1A1B25','#16171F'],
        'eye-protection':    ['#C7EDCC','#D8F5DD'],
        'quiet-light':       ['#F5F5F5','#FFFFFF'],
        'ysgrifennwr':       ['#F5EDDA','#FAF4E3'],
        'mono':              ['#F5F5F5','#FFFFFF'],
    };
    // 有效性兜底：criticalColors 的 key 集合作为有效集（与 themeLabels 同步的维护契约）
    var validKeys = Object.keys(criticalColors);
    var stored = localStorage.getItem('jot_theme');
    var theme = (stored && validKeys.indexOf(stored) !== -1) ? stored : 'default';
    // 修正无效值：立即写回 localStorage，避免后续 applyTheme 再次校正
    if (stored !== theme) localStorage.setItem('jot_theme', theme);
    document.documentElement.setAttribute('data-theme', theme);
    var c = criticalColors[theme];
    var style = document.createElement('style');
    style.textContent = ':root{--bg:' + c[0] + ';--topbar-bg:' + c[1] + '}';
    document.head.appendChild(style);
})();
```

**为什么 inline 用 `Object.keys(criticalColors)` 作有效集**：
- 内联脚本在 ESM 加载前运行，**不能 import** `themeLabels`
- criticalColors 已在维护流程中与 themeLabels 同步（4 文件流程：variables.css / theme-config.js / index.html criticalColors / main.go themeBG）
- 不引入新漂移，零额外维护成本

### 4. 不需改动的位置
- **`main.js` `loadSettings`（L11600-11606）**：直接调 `applyTheme(cfg.theme)`，校验由 applyTheme 统一处理
- **`main.js` `cycleTheme`（L1811-1824）**：从 `Object.keys(themeLabels)` 选 next，天然不会出无效值
- **`main.js` `buildThemeDropdown`（L1830-1845）**：下拉项的 key 都来自 themeLabels，天然不会传无效值
- **后端 Go 端 `GetAllSettings()`**：返回什么值都合法，前端兜底覆盖所有路径

## Assumptions & Decisions

| 决策 | 理由 |
|---|---|
| `themeLabels` 是有效名单的唯一事实来源 | 已有 4 文件维护流程保证权威性 |
| 内联脚本用 `Object.keys(criticalColors)` 作有效集 | criticalColors 已在维护契约内与 themeLabels 同步，避免引入新漂移 |
| 通知类型用 `warning` | 用户已确认。已自动修正但需告知，比 `info` 醒目 |
| inline 立即写回 localStorage | 用户已确认。避免后续 applyTheme 再次校正，状态更干净 |
| `applyTheme` 校验在 `syncThemeUI` 和早返回之前 | 否则下拉菜单高亮不会同步到 default；早返回会跳过修正 |
| `saveSettings()` 同步调（不 await） | 持久化异步进行，通知立即发，不阻塞 UI |
| 不引入 zod / schema 校验库 | 轻量一行函数足够，无依赖膨胀 |
| 不修 Go 端 GetAllSettings() | 后端返回任何值都合法，前端兜底是更合适的边界 |

## Verification

### 功能验证
| 场景 | 期望行为 |
|---|---|
| 正常：localStorage 存 `'dark'` | 应用 dark 主题，无通知 |
| 兜底 A：localStorage 存 `'invalid-theme'` | 启动后立即写回 `'default'`，应用 default，看到 warning 通知"主题"invalid-theme"已失效" |
| 兜底 B：后端 cfg.theme 存 `'bogus'` | loadSettings 调用 applyTheme 校正，看到 warning 通知，localStorage + 后端都更新为 `'default'` |
| 空值：localStorage 无 `'jot_theme'` | 走 default，无通知（空值不算"失效"） |
| 早返回陷阱：data-theme 已是 `'bogus'`，applyTheme 收到 `'bogus'` | resolveTheme 把它转 `'default'`，**不**触发早返回，应用 default |

### 构建验证
- `go build` 成功
- `npm run build` 成功
- `npm run lint` 0 errors

### 边界保护
- `themeName` 为 `null` / `undefined` / `''` / `'__proto__'` → resolveTheme 都判为无效，回退 default
- `localStorage` 抛错（如隐私模式） → try/catch 包裹（可选；当前内联脚本未做，可后续补）

## Out of Scope
- 不修 Go 端 `GetAllSettings()` 返回值
- 不迁移主题存储结构（如统一到后端）
- 不引入 schema 校验库
- 不动 4 文件主题维护流程（criticalColors 与 themeLabels 同步契约保持）
