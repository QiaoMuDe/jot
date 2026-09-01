# 拖拽导入文件：匹配与时间对比规则

> 功能入口：笔记首页拖拽文件 → `ImportFiles` → `processImportFile`
> 本文记录导入的完整规则与关键实现，供后续维护参考。最后更新：2026-09-01

## 一、功能概述

拖拽文件到笔记首页时，系统将文件内容导入为笔记，并具备"智能覆盖"能力：

- **无匹配笔记** → 创建新笔记
- **匹配到已有笔记**（标题 + 后缀 + 笔记本相同）→ 通过**内容哈希 + 时间对比**两级规则决定：跳过 / 直接覆盖 / 弹窗让用户选择

## 二、导入处理流程

入口：[app.go](../../app.go) `ImportFiles`（批量并发处理，批内按标题+后缀去重并追加编号）→ `processImportFile`（单文件处理）。

```
1. os.Stat 检查路径 / 拒绝目录 / 文件大小限制（设置中的 maxSize）
2. 提取标题（文件名去后缀，支持批内去重编号覆盖）与后缀
3. 内容读取：
   - 办公文件（.docx/.pdf/.xlsx 等）→ markitdown 转为 Markdown，后缀统一为 .md
   - 二进制文件 → 拒绝
   - 纯文本 → 直接读取
4. FindByTitleAndExt(title, fileExt, notebookID) 查找已有匹配笔记
5. 有匹配 → 两级对比规则（见下）；无匹配 → 创建新笔记
```

## 三、核心规则：两级对比（哈希兜底 + 时间对比）

```
匹配到已有笔记
│
├─ ① 内容哈希对比（最高优先级）
│     hash(规范化笔记内容) == hash(规范化文件内容)
│     → 一致：直接 skipped（两边本就同步，无论时间戳如何，不弹窗）
│       （哈希计算失败时记日志，降级为纯时间对比）
│
└─ ② 时间对比（仅内容不一致时）
      fileModTime >  note.UpdatedAt → "updated"  直接覆盖
      fileModTime <  note.UpdatedAt → "conflict" 弹窗让用户选择覆盖/保留
      相等                          → "skipped"
```

### 为什么这样设计（历史背景）

旧版实现中，首次导入把笔记 `CreatedAt`/`UpdatedAt` 设为导入时刻，导致重导入同一文件时 `UpdatedAt` 永远比文件 mtime 新，**每次都误报冲突**。

现行方案的核心：**导入写入（创建/覆盖）时，把笔记时间戳对齐为文件的修改时间 `ModTime()`**。时间戳本身成为同步基准：

- 重导入未变文件 → 时间相等 + 哈希一致，双保险静默跳过 ✓
- 文件被外部修改 → mtime 变新，时间对比正确触发覆盖 ✓
- 用户在应用内编辑笔记 → `UpdatedAt` 被刷新为编辑时刻，成为笔记侧真实时间 ✓
- 一个字段同时扮演"内容时间"和"同步基准"两个角色，**无需额外基准字段**

### 场景决策表

| 场景 | 内容哈希 | 时间 | 结果 |
|---|---|---|---|
| 重导入未变的文件 | 一致 | — | `skipped`（零打扰） |
| 文件被外部修改后重导入 | 不一致 | 文件更新 | `updated` 自动覆盖 |
| 编辑笔记后重导入未变文件 | 不一致 | 笔记更新 | `conflict` 弹窗（选保留即可，方向安全） |
| 编辑笔记、文件其后也被改 | 不一致 | 文件更新 | `updated` 覆盖（已知取舍：本地编辑丢失） |
| 手动创建的旧笔记 / 历史数据 | 不一致 | 任意 | 走纯时间对比，天然兼容 |

## 四、冲突解决（前端交互）

- `processImportFile` 返回 `status: "conflict"`，附带 `file_time` / `note_time` / `content` / `file_ext`
- 前端 [main.js](../../frontend/src/main.js) `showImportConflictDialog` 弹窗列出所有冲突项（支持逐项 / 全部覆盖 / 全部保留）
- 用户决策后调用后端 `ResolveImportConflict(noteID, overwrite, title, content, fileExt, fileTime)`：
  - 覆盖 → `noteService.UpdateWithTime`，**时间戳同样对齐为文件 mtime**（保证后续对比基准正确）
  - 保留 → 不做任何操作

## 五、代码位置索引

| 位置 | 职责 |
|---|---|
| [app.go](../../app.go) `ImportFiles` | 批量导入入口，并发处理 + 批内去重 + 进度事件 |
| [app.go](../../app.go) `processImportFile` | 单文件导入：读取、匹配、两级对比、创建/覆盖 |
| [app.go](../../app.go) `importContentHash` | 内容规范化哈希（SHA256），哈希兜底的核心 |
| [app.go](../../app.go) `ResolveImportConflict` | 冲突用户决策 API（Wails 绑定） |
| [note_service.go](../../internal/services/note_service.go) `FindByTitleAndExt` | 按标题+后缀+笔记本查找匹配笔记 |
| [note_service.go](../../internal/services/note_service.go) `CreateWithNotebookAt` | 创建笔记并对齐时间戳（导入专用） |
| [note_service.go](../../internal/services/note_service.go) `UpdateWithTime` | 覆盖笔记并对齐时间戳（导入专用） |
| [note.go](../../internal/models/note.go) | Note 模型（`UpdatedAt` 参与列表排序） |
| [main.js](../../frontend/src/main.js) `showImportConflictDialog` | 冲突弹窗 UI 与决策回传 |
| [note_service_test.go](../../internal/services/note_service_test.go) `TestCreateWithNotebookAt` / `TestUpdateWithTime` | 时间戳对齐的单元测试 |

## 六、关键实现细节（改代码前必读）

1. **GORM 自动时间戳**：`db.Save` / `db.Updates` 会自动把 `UpdatedAt` 刷成 now，会破坏时间基准。导入路径必须：
   - 创建 → `CreateWithNotebookAt`（预设非零值，GORM 保留；实测已验证，另有兜底修正逻辑）
   - 覆盖 → `UpdateWithTime`（`UpdateColumns` 绕过自动时间戳）
   - 不要在导入路径改用普通的 `Update` / `CreateWithNotebook`
2. **哈希规范化**：对比前统一 `\r\n → \n` + `TrimSpace`，避免换行符差异 / 尾部空行造成假性不一致。哈希运行时计算（go-kit `hash.HashString`，`gitee.com/MM-Q/go-kit/hash`），不持久化。
3. **修改 app.go 签名后**需执行 `wails generate module` 重新生成 `frontend/wailsjs` 绑定（沙箱受限时也可手动同步 `App.js` / `App.d.ts`）。
4. **UI 显示语义**：导入的笔记显示**文件的修改时间**而非导入时刻，列表按 `updated_at` 排序也随文件时间——这是产品上已确认接受的行为。

## 七、已知取舍（当前设计的有意为之）

1. 用户编辑笔记后重导入未变的文件 → 仍弹一次冲突窗（内容确实不同 + 笔记时间更新），选"保留"即可。频率低，方向安全。
2. 用户编辑笔记、文件在其后也被修改 → 自动覆盖，本地编辑丢失。这是"最后写入者胜"的固有取舍。
3. 若将来需要根治：给 Note 增加 `ImportedFileModTime` 基准字段（记录上次导入时文件的 mtime），可精确区分"文件变了"与"笔记变了"，届时本规则中的时间对比以该字段为基准即可，其余设计不变。

## 八、测试

### 单元测试

```powershell
go test ./internal/services/ -run 'TestCreateWithNotebookAt|TestUpdateWithTime' -v
```

### 手动验证流程（摘要）

1. `wails dev` 启动，准备测试文件并用 PowerShell 指定 mtime：

   ```powershell
   "# Hello v1" | Out-File D:\import-test\readme.md -Encoding utf8
   (Get-Item D:\import-test\readme.md).LastWriteTime = Get-Date "2026-08-30 10:00"
   ```

2. 首次拖入 → 笔记时间 = 文件修改时间
3. 原样再拖 → 静默跳过（日志出现 `processImportFile: 内容一致，跳过`）
4. 修改文件再拖 → 直接覆盖，无弹窗
5. 应用内编辑笔记后拖入未变文件 → 弹冲突窗，选跳过笔记不变
6. 冲突窗选覆盖 → 笔记时间重置为文件时间，再拖仍静默跳过

日志实时观察：

```powershell
Get-Content C:\Users\QIAOMU\.jot\logs\app.log -Wait -Tail 0 | Select-String "processImportFile|ResolveImportConflict"
```
