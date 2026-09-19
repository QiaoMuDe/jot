# 设置页新增设置项维护权威文档

本文档是本项目**设置项系统的唯一权威参考**。新增、修改或删除任何设置项（toggle/输入框/下拉/分段控件）时，优先参照本文件执行。

## 一、设置项系统的组成与数据流

设置项持久层为 SQLite `settings` 表（KV 结构，`models.Setting`），前端与后端之间只有一条边界：`GetAllSettings()` / `SaveAllSettings()`（Wails 绑定）。

端到端链路（保存：①→④；读取/回显：④→①）：

| 环节 | 文件 | 作用 |
| --- | --- | --- |
| ① UI 控件 | `frontend/index.html` | 设置页分区卡片内的 HTML 控件（唯一 UI 入口） |
| ② 前端逻辑 | `frontend/src/main.js` | `els` 注册元素 → `saveSettings()` 收集 DOM 值 → `loadSettings()` 把 cfg 回填 DOM |
| ③ 后端边界 | `internal/services/types.go` | `SettingsConfig` 统一结构体 + `GetAllSettings()` 读取 + `SaveAllSettings()` 写入（含 clamp 校验与特殊值处理） |
| ④ 持久层种子 | `internal/database/db.go` | `InitDefaultSettings()` 增量插入默认值（仅缺失 key） |

命名约定：设置 key 使用**小写英文下划线**（snake_case，如 `editor_word_wrap`），且必须与 `SettingsConfig` 对应字段的 `json` tag 完全一致（前端经 `cfg.xxx` 访问）。

## 二、维护涉及的文件

| 文件 | 角色 |
| --- | --- |
| `internal/database/db.go` | `InitDefaultSettings()` 种子默认值；`cleanupOrphanedData()` 孤儿键清理（删除设置项时用） |
| `internal/services/types.go` | `SettingsConfig` 结构体 + `GetAllSettings()`/`SaveAllSettings()` 读写映射 |
| `frontend/index.html` | 设置分区卡片 HTML 控件 |
| `frontend/src/main.js` | `els` 注册 + `loadSettings()` 回显 + `saveSettings()` 收集 + 可选 `change` 自动保存 |

> 样式：新控件复用现有 class（`.ai-setting-item`/`.toggle-switch`/`.settings-input`/`.theme-select`，见 [settings-panel.css](settings-panel.css)），**无需改 CSS**；只有全新控件形态才需要动样式文件。
> 例外：标签文案过长需放宽 `.ai-setting-label` 的固定列宽（详见第三节第 3 步的「标签文案宽度上限」）。

## 三、新增一个设置项

依次完成 4 个文件共 7-8 处改动。以新增 bool 型 toggle `my_feature_enabled` 为例：

### 1. `internal/database/db.go` — `InitDefaultSettings` 的 defaults 列表末尾追加

```go
{Key: "my_feature_enabled", Value: "false"},
```

- **增量插入**：仅对缺失 key 的库生效——新装用户取默认值，**存量用户不受种子值变化影响**（改默认值只影响新用户，存量库需另写迁移）。
- 复杂默认值可加一行中文注释说明用途（参照 `ai_context_token_budget` 的写法）。

### 2. `internal/services/types.go` — 三处，缺一不可

① `SettingsConfig` 结构体新增字段（json tag = key）：

```go
MyFeatureEnabled bool `json:"my_feature_enabled"`
```

② `GetAllSettings()` 初始化映射中新增读取（按类型选解析函数）：

```go
MyFeatureEnabled: parseBoolSetting(s.Get("my_feature_enabled")),
```

③ `SaveAllSettings()` 的 `sets` map 中新增写入：

```go
"my_feature_enabled": strconv.FormatBool(cfg.MyFeatureEnabled),
```

- 类型对照：bool → `parseBoolSetting`/`strconv.FormatBool`；int → `parseIntSetting`/`strconv.Itoa`；float → `parseFloatSetting`/`strconv.FormatFloat(..., 'f', -1, 64)`；string → `s.Get`/直接赋值。
- **int/float 需要范围限制时**，在 `SaveAllSettings()` 顶部 clamp 校验区追加（参照 `PageSize` 的 `< 1` 兜底 + `> 100` 上限模式）。
- **读取默认值必须与种子值一致**（两处独立定义，不一致会导致新装用户与存量用户行为分叉）。
- 敏感值特殊处理：API Key 类需 `EncodeB64`/`DecodeB64`，密码类在 `SaveAllSettings()` 有专用 SHA-256 分支（参照 `screen_lock_password`，一般勿仿照）。

### 3. `frontend/index.html` — 对应设置分区卡片内新增控件

toggle 模板（输入框/下拉参照同分区现有控件）：

```html
<div class="ai-setting-item">
    <label class="ai-setting-label">我的功能</label>
    <span class="font-setting-desc">功能一句话说明</span>
    <div class="toggle-switch">
        <input type="checkbox" id="myFeatureEnabledToggle">
        <label class="toggle-track" for="myFeatureEnabledToggle">
            <span class="toggle-thumb"></span>
        </label>
    </div>
</div>
```

- **标签文案宽度上限（易踩坑）**：`.ai-setting-label` 是**固定列宽**（`width: 136px`，见 [settings-panel.css](settings-panel.css)），且为 `nowrap + flex-shrink: 0`。文案超出时**不会撑开盒子，而是直接溢出盒子**，与右侧描述粘连——因为描述是从标签盒子右边界 + `gap: 12px` 处起排，而不是紧跟文字末尾。
- 估算规则（`font-size: 0.813rem`）：中文 ≈ 13px/字、英文 ≈ 7px/字符。即**纯中文标签 ≤ 10 字**；中英混合需满足总宽 ≤ 136px（例：`SubAgent 运行上限` ≈ 110px，通过；`SubAgent 最大运行次数上限` 会溢出）。
- 超长时二选一：① 精简文案（首选，零风险）；② 同步放宽 `.ai-setting-label` 的 `width`（**影响全页所有设置项**的标签列与描述可用宽度，描述是 `flex: 1` + `nowrap + ellipsis`，过窄会触发省略号）。

### 4. `frontend/src/main.js` — 三至四处

④ `els` 对象注册元素引用：`myFeatureEnabledToggle: $('myFeatureEnabledToggle'),`
⑤ `loadSettings()` 中回显：`if (els.myFeatureEnabledToggle) els.myFeatureEnabledToggle.checked = cfg.my_feature_enabled;`
⑥ `saveSettings()` 的 `cfg` 对象中收集：`my_feature_enabled: els.myFeatureEnabledToggle?.checked || false,`
⑦ 若需修改即保存，在事件绑定区域添加：

```js
els.myFeatureEnabledToggle.addEventListener('change', async () => {
    await saveSettings();
    nm.show('设置已保存', 'success');
});
```

### 5. CM6 编辑器相关设置（特殊）

编辑器行为类设置（如 `initCodeMirror` 参数）需在**所有调用点透传**：`openEditor`/`applyFileExt`/`toggleFileExt`/`applyCodeHighlightTheme` 共 4 处，漏一处则该入口不生效。

### 6. 验证

- `go build ./...` 通过；`wails dev` 后设置项出现在对应分区；
- 修改后刷新/重启设置保留（落库验证）；
- 全新库与存量库两种场景下默认值行为一致；
- CM6 类设置在编辑器各入口实时生效。

## 四、删除一个设置项

删除时清理**同样四处**，并额外清理存量库残留：

1. `index.html`：删除控件节点；
2. `main.js`：删除 `els` 注册、`loadSettings` 回显、`saveSettings` 收集、`change` 监听；
3. `types.go`：删除结构体字段 + `GetAllSettings` 读取 + `SaveAllSettings` 写入（及专属 clamp）；
4. `db.go`：删除 `InitDefaultSettings` 种子行，**并在 `cleanupOrphanedData()` 的 `orphanSettingKeys` 清单追加该 key**（`InitDefaultSettings` 只插缺失键，存量库的旧键会永久残留，必须走孤儿键 DELETE 清理）。

历史案例：`ai_context_window_size`/`ai_card_recall_enabled` 等移除时均走 `orphanSettingKeys` 清理。

## 五、无 UI 的纯后端设置项（简化流程）

仅后端使用、不暴露到设置页的 key（如 `ai_read_url_max_chars`/`ai_http_max_chars`）：

- 只做第三节第 1 步（种子）+ 使用处直接 `s.Get("key")` 读取；
- **不加** `SettingsConfig` 字段与读写映射（避免前后端无意义的全量传输），注释标明「无前端 UI，仅初始化默认值」。

## 六、易漏项清单

- [ ] `types.go` 三处缺一不可——漏读取映射 → 前端永远拿到零值；漏写入映射 → 保存静默丢失（`SaveAllSettings` 不报错）；
- [ ] `GetAllSettings` 默认值 = 种子值（两处独立定义）；
- [ ] json tag 与 key 完全一致；
- [ ] 标签文案宽度 ≤ `.ai-setting-label` 固定列宽（136px，纯中文 ≤ 10 字）——超出会溢出并与右侧描述粘连（见第三节第 3 步）；
- [ ] 存量用户默认值不随种子变化，需迁移时在 `InitDefaultSettings`/迁移区单独处理；
- [ ] CM6 类设置的 4 处调用点透传；
- [ ] 删除时 `orphanSettingKeys` 追加清理存量键。
