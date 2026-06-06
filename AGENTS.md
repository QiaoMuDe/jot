# Jot 项目分析报告

> 生成日期: 2026-06-07
> 项目类型: 桌面端卡片式笔记应用（类小米笔记）
> 技术栈: Wails v2 + Go + GORM + SQLite + 原生 HTML/CSS/JS

---

## 一、目录结构梳理

```
jot/                                    # 项目根目录
├── main.go                             # 【入口文件】Wails 应用启动入口，配置窗口/资源/绑定
├── app.go                              # 【核心文件】Wails 绑定层，暴露 38 个 Go API 给前端
├── go.mod                              # Go 模块定义，声明依赖版本
├── go.sum                              # Go 依赖锁文件
├── wails.json                          # Wails 项目配置（名称/构建脚本/作者）
├── AGENTS.md                           # 本报告文件
│
├── internal/                           # 【内部包】Go 子包统一目录
│   ├── database/
│   │   └── db.go                       # SQLite 初始化（glebarez/sqlite 纯 Go 驱动）+ DefaultDBPath() 路径函数
│   ├── fontutil/
│   │   └── fonts_windows.go           # EnumFontFamiliesW API 封装
│   ├── models/
│   │   ├── note.go                     # Note 实体（笔记）
│   │   ├── tag.go                      # Tag 实体（标签）
│   │   ├── setting.go                  # Setting 实体（KV 配置）
│   │   └── draft.go                    # Draft 实体（草稿，仅 1 行记录 ID=1）
│   └── services/
│       ├── note_service.go             # 笔记 CRUD + 搜索 + 置顶 + 回收站 + 统计 + 导入导出 + GetAllIDs
│       ├── tag_service.go              # 标签管理 + 笔记标签关联 + 标签计数
│       ├── setting_service.go          # 配置读写
│       ├── draft_service.go            # 草稿 SaveDraft/GetDraft/ClearDraft
│       └── types.go                    # 通用类型（PaginatedResult, DataStats, ImportResult 等）
│
├── frontend/                           # 【前端目录】Wails 前端（Vanilla + Vite）
│   ├── index.html                      # 入口 HTML，单栏布局 + 6 个视图
│   ├── package.json                    # 前端依赖（仅 Vite 3.x）
│   ├── src/
│   │   ├── main.js                     # 【核心文件】前端逻辑 ~3800 行
│   │   ├── style.css                   # 组件样式 ~2974 行
│   │   └── app.css                     # 全局样式（reset/布局/滚动条 ~458 行）
│   ├── wailsjs/                        # Wails 自动生成的 JS 绑定
│   │   └── go/main/
│   │       ├── App.js                  # 后端 API 的 JS 封装
│   │       ├── App.d.ts               # TypeScript 类型定义
│   │       └── models.ts              # Go 模型的 TS 类型
│   └── dist/                           # Vite 构建产物（前端编译输出）
│
└── .trae/specs/                        # 项目 Spec 文档目录
    ├── add-card-note-app/              # 初始需求规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-data-management/            # 数据管理功能规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-font-settings/              # 字体设置功能规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-quick-note-mode/          # 快速笔记模式规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-md-rendering/             # Markdown 渲染查看规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-about-page/               # 关于页面规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-misc-improvements/         # 杂项优化规格（滚动条美化/默认标签/快捷键说明/分段控件重构）
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── add-one-click-backup-restore/   # 一键备份还原规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── redesign-data-management/       # 数据管理页面 UI 重构与动画增强规格
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    ├── unify-notification-system/       # 统一通知系统规格（右上角浮动通知组件）
    │   ├── spec.md
    │   ├── tasks.md
    │   └── checklist.md
    └── enhance-interaction-animation/  # 交互体验与动画增强规格（16 项动画/过渡/交互动画）
        ├── spec.md
        ├── tasks.md
        └── checklist.md
```

### 目录规范评价

| 维度 | 评价 |
|------|------|
| **分层清晰度** | 优秀。严格按 `models → services → database → app` 分层，前端后端隔离清晰 |
| **命名规范** | 良好。目录名使用复数形式（models/services），符合 Go 社区惯例 |
| **冗余目录** | 无。每个目录职责单一，无多余层级 |
| **待改进** | frontend/dist 为构建产物，应加入 .gitignore |

---

## 二、核心功能模块识别

### 2.1 基础支撑模块

| 模块名称 | 核心功能 | 对应文件 | 核心依赖 |
|----------|----------|----------|----------|
| **数据库初始化模块** | SQLite 连接建立、连接池配置、AutoMigrate | `database/db.go` | glebarez/sqlite, GORM |
| **数据模型层** | Note/Tag/Setting 实体定义、GORM tag 映射 | `models/note.go`, `models/tag.go`, `models/setting.go` | GORM |
| **通用类型** | 分页返回格式、统计数据、导入导出结构 | `services/types.go` | 无外部依赖 |
| **Wails 绑定层** | Go API → JS Bridge，含 runtime.SaveFileDialog | `app.go` | Wails v2 binding + runtime |
| **前端构建** | Vite 打包、Wails dev 热重载 | `frontend/package.json`, `wails.json` | Vite 3.x（保留，未移除）|
| **前端构建流程** | `wails build` 自动执行 `npm run build`（Vite）→ `frontend/dist/`，再嵌入 Go 二进制 | `go:embed all:frontend/dist` | 前端构建和后端编译都由 `wails build` 一条命令完成 |
| **字体枚举** | Windows GDI EnumFontFamiliesW 系统字体枚举 | `fontutil/fonts_windows.go` | gdi32.dll / user32.dll (syscall) |
| **配置存储** | KV 结构配置读写（字体偏好等） | `services/setting_service.go` | GORM |
| **路径工具** | 数据库默认路径 `~/.jot/data/jot.db` | `database/db.go:DefaultDBPath()` | `os.UserHomeDir()` |

### 2.2 业务核心模块

| 模块名称 | 核心功能 | 对应代码 | 核心输入 | 核心输出 |
|----------|----------|----------|----------|----------|
| **笔记 CRUD** | 创建/更新/查询/删除笔记 | `services/note_service.go` | 标题/内容/颜色/ID | Note 对象/错误 |
| **笔记搜索** | 标题+内容 LIKE 模糊搜索 | `note_service.go:Search()` | 关键词/分页参数 | 笔记列表+总数 |
| **笔记置顶** | 切换置顶状态 | `note_service.go:TogglePin()` | 笔记 ID | 更新后的笔记 |
| **回收站** | 软删除/查看/恢复/永久删除 | `note_service.go:Delete/GetTrash/Restore/PermanentDelete` | 笔记 ID | 操作结果 |
| **批量回收站操作** | 全部恢复/全部清空 | `note_service.go:RestoreAll/EmptyTrash` | — | 操作结果 |
| **标签管理** | 标签 CRUD | `services/tag_service.go` | 名称/颜色/ID | Tag 对象 |
| **笔记标签关联** | 为笔记添加/移除标签 | `tag_service.go:AddTagToNote/RemoveTagFromNote` | 笔记ID+标签ID | 操作结果 |
| **按标签筛选** | 通过标签 ID 查询笔记 | `note_service.go:GetByTag()` | 标签ID/分页参数 | 笔记列表+总数 |
| **数据统计** | 统计笔记总数/回收站数/标签数 | `note_service.go:GetStats()` + `tag_service.go:Count()` | — | DataStats 对象 |
| **数据导出为 .db** | 导出为 SQLite 数据库文件（VACUUM INTO + fs.CopyEx）| `app.go:ExportDataWithDialog()` | — | "导出成功" 提示 |
| **数据导入** | 从 JSON 文件导入笔记（跳过同名） | `note_service.go:ImportFromJSON()` | JSON 字节数组 | ImportResult 对象 |
| **前端卡片渲染** | 卡片网格展示 | `frontend/src/main.js` | 笔记数据数组 | DOM 渲染 |
| **前端编辑器** | 笔记编辑模态框（含标签选择/颜色选择） | `frontend/src/main.js` | 笔记数据/用户输入 | 保存/取消 |
| **前端搜索交互** | 输入框 250ms 防抖自动搜索，支持标题/内容/标签 | `frontend/src/main.js` | 关键词 | 搜索结果列表 |
| **前端导航切换** | 网格/搜索/设置/数据管理/回收站视图切换 | `frontend/src/main.js:switchView()` | 视图名称 | 视图 DOM 切换 |
| **前端右键菜单** | 右键弹出菜单（查看/编辑/置顶/删除） | `frontend/src/main.js` | 鼠标事件+笔记ID | 菜单显示/操作 |
| **前端只读查看** | 左击笔记打开只读查看器 | `frontend/src/main.js:openEditor()` | 笔记 ID | 只读查看模态框 |
| **标签搜索** | 点击标签 chip 触发按标签名搜索 | `frontend/src/main.js:searchByTag()` | 标签名 | 搜索结果列表 |
| **键盘快捷键** | Ctrl+F 搜索 / Ctrl+N 新建 / PgUp/PgDn 滚动 / Ctrl+Home/End | `frontend/src/main.js:handleKeyboardNavigation()` | 键盘事件 | 对应操作 |
| **版本号信息** | 返回 verman.V.GitVersion 纯版本号 | `app.go:GetVersion()` | — | 版本字符串 |
| **打开外链** | 调用 runtime.BrowserOpenURL 在默认浏览器打开链接 | `app.go:OpenProjectURL()` | URL 字符串 | — |
| **打开数据目录** | 在文件管理器中打开 `~/.jot/data/` | `app.go:OpenDataDir()` | — | explorer 文件管理器 |
| **一键备份** | 备份当前库到 `~/.jot/backup/jot-backup.db`（覆盖）| `app.go:BackupToDir()` | — | 备份成功提示 |
| **一键还原** | 从 `jot-backup.db` 还原并刷新笔记/标签/统计 | `app.go:RestoreFromDir()` | — | Toast 提示结果 |
| **字体设置** | 字体族下拉选择（搜索+键盘导航）+ 字体大小预设/自定义 | `frontend/src/main.js:loadFontSettings/applyFontFamily/applyFontSize` | 字体名称/大小 | 更新 CSS 变量 |
| **统一通知系统** | NotificationManager 单例类，右上角浮动通知，4 种类型 + undo 撤销 | `frontend/src/main.js:NotificationManager` | 消息/类型/回调 | 通知 DOM 创建与自动销毁 |

### 2.3 模块分层图

```
┌─────────────────────────────────────────────────────┐
│                    Frontend                          │
│  (main.js / style.css / index.html)                  │
│   ├─ 视图渲染 (卡片/搜索/设置/数据管理/回收站)          │
│   ├─ 交互逻辑 (事件绑定/状态管理)                      │
│   └─ Wails Bridge (window.go.main.App.*)              │
└────────────────────────┬────────────────────────────┘
                         │ Wails Binding (JSON 序列化)
┌────────────────────────▼────────────────────────────┐
│              App 层 (app.go)                         │
│  32 个绑定方法（CRUD/搜索/置顶/回收站/统计/导入导出/路径）│
│  (含 runtime.SaveFileDialog 原生对话框调用)            │
└────────────────────────┬────────────────────────────┘
                         │
              ┌──────────┼──────────┐
              ▼          ▼          ▼
    ┌─────────────┐ ┌──────────┐ ┌──────────────┐
    │ NoteService │ │TagService│ │    Types     │
    │ (CRUD/搜索/ │ │(CRUD/关联)│ │ Paginated    │
    │  置顶/回收站 │ │          │ │ DataStats    │
    │  统计/导入   │ │          │ │ Import       │
    │  导出)      │ │          │ │ Result 等    │
    └──────┬──────┘ └─────┬────┘ └──────────────┘
           │              │
           └──────┬───────┘
                  ▼
        ┌─────────────────┐
        │    GORM ORM     │
        │ (数据访问层)      │
        └────────┬────────┘
                 ▼
        ┌─────────────────┐
        │    SQLite       │
        │ (glebarez/sqlite│
        │  纯 Go 驱动)    │
        └─────────────────┘
```

---

## 三、模块间依赖关系分析

### 3.1 依赖关系详表

| 依赖方 | 被依赖方 | 依赖类型 | 依赖详情 |
|--------|----------|----------|----------|
| `app.go` | `database` | 编译依赖 | 调用 `database.InitDB()` 获取 `*gorm.DB` 实例 |
| `app.go` | `services` | 编译依赖 | 创建 `NoteService` / `TagService` / `SettingService` 实例 |
| `app.go` | `models` | 编译依赖 | 返回 `*models.Note` / `*models.Tag` / `*models.Setting` 类型 |
| `app.go` | `runtime` | 编译依赖 | `runtime.SaveFileDialog` 原生保存对话框 |
| `app.go` | `fontutil` | 编译依赖 | `fontutil.GetFonts()` 枚举系统字体 |
| `services` | `models` | 编译依赖 | 操作 Note/Tag/Setting 结构体 |
| `services` | GORM | 编译依赖 | `*gorm.DB` 数据库操作 |
| `database` | `models` | 编译依赖 | `AutoMigrate(&models.Note{}, &models.Tag{}, &models.Setting{})` |
| `database` | glebarez/sqlite | 编译依赖 | 纯 Go SQLite 驱动 |
| `fontutil` | gdi32/user32 | 运行时依赖 | syscall 调用 Windows GDI API |
| `frontend/main.js` | `wailsjs/go/main/App.js` | 运行时调用 | `window.go.main.App.*` 调用后端 API |
| `frontend/wailsjs` | `app.go` | 构建时生成 | `wails generate module` 自动生成 |

### 3.2 依赖关系图（Mermaid）

```mermaid
graph TD
    subgraph Backend
        A[main.go] --> B[app.go]
        B --> C[database/db.go]
        B --> D[services/note_service.go]
        B --> E[services/tag_service.go]
        B --> F[services/types.go]
        C --> G[models/note.go]
        C --> H[models/tag.go]
        D --> G
        D --> H
        E --> G
        E --> H
        C --> I[glebarez/sqlite]
        D --> J[GORM]
        E --> J
        B -.-> K[runtime.SaveFileDialog]
    end

    subgraph Frontend
        L[index.html] --> M[main.js]
        M --> N[style.css]
        M --> O[app.css]
        M --> P[wailsjs/go/main/App.js]
    end

    B -.->|Wails Binding| P
```

### 3.3 依赖问题分析

| 问题类型 | 描述 | 严重程度 |
|----------|------|----------|
| **循环依赖** | 无。所有依赖为单向 `main → app → services → models`，无循环 | ✅ 无问题 |
| **过度依赖** | 无。每个 Service 仅依赖 `*gorm.DB` 和自身模型 | ✅ 无问题 |
| **依赖缺失** | 无。`go.sum` 中所有传递依赖完整 | ✅ 无问题 |
| **隐式依赖** | 前端 `window.go` 对象依赖 Wails 运行时注入，本地开发/独立预览时不可用 | ⚠️ 有降级处理（Mock 数据） |
| **编译期依赖 vs 运行时依赖** | `wailsjs/` 目录需在修改 `app.go` 后重新生成 | ⚠️ 需手动执行 `wails generate module` |

---

## 四、设计模式与实现逻辑

### 4.1 设计模式识别

| 模式名称 | 应用位置 | 说明 | 代码示例 |
|----------|----------|------|----------|
| **Service Layer 模式** | `services/` 包 | 将业务逻辑从 controller（app.go）中抽离，封装为独立 Service 结构体 | `NoteService` / `TagService` |
| **依赖注入 (DI)** | `app.go` | Service 依赖的 `*gorm.DB` 通过构造函数注入 | `NewNoteService(db)` / `NewTagService(db)` |
| **Repository 模式** | `services/` 包内嵌 GORM | Service 内部直接使用 GORM 作为数据访问层 | `s.db.Create()` / `s.db.Where()` |
| **单例模式 (应用级)** | App 结构体 | Wails 运行时保证 App 实例唯一 | `NewApp()` 在 `main()` 中仅调用一次 |
| **MVC 变体** | 整体架构 | Model(models) - View(frontend) - Controller(app.go + services) 分层 | 见分层图 |
| **降级策略 (Fallback)** | `frontend/main.js` | 后端未绑定时自动使用 Mock 数据 | `if (!window.go.main.App.GetNotes) { state.notes = getMockNotes(); }` |
| **Wails Runtime 集成** | `app.go` | 通过 runtime 包调用原生桌面功能 | `runtime.SaveFileDialog()` 弹出系统保存对话框 |

### 4.2 核心业务逻辑流程

#### 4.2.1 笔记创建流程

```
用户点击 "+" 按钮 / Ctrl+N
  → openEditor(null)          // 打开空编辑器模态框
    → 用户填写标题/内容/选择颜色/选择标签
    → 点击"保存"按钮
      → createNote()
        → 前端校验（标题不为空）
        → window.go.main.App.CreateNote(title, content, color)
          → app.go:CreateNote()
            → noteService.Create()          // GORM db.Create(&note)
            → 返回 *models.Note（含 id）
          → 遍历 selectedTags
            → window.go.main.App.AddTagToNote(note.id, tagId)
              → tagService.AddTagToNote()   // GORM Association("Tags").Append
        → closeEditor()                     // 关闭模态框
        → loadNotes()                       // 重新加载笔记列表
          → GetNotes(1, 100)                // 分页查询
          → renderCardGrid()                // 渲染右侧卡片网格
```

#### 4.2.2 笔记搜索流程

```
Ctrl+F / 用户点击搜索框 → 输入框聚焦
用户在搜索框输入文字 → 250ms 防抖
  → searchNotes(keyword, 'input')
    → 前端调用 SearchNotes(kw, 1, 100)
      → app.go:SearchNotes()
        → noteService.Search(keyword, page, pageSize)
          → GORM: WHERE title LIKE '%kw%' OR content LIKE '%kw%'
                 AND deleted_at IS NULL
          → ORDER BY pinned DESC, updated_at DESC
          → Preload("Tags")
          → 返回 []Note + total
    → 接收 PaginatedResult.Items
    → renderSearchResults(results, keyword)
      → 关键词高亮 (highlightText)
      → 渲染搜索结果列表（含可点击标签）
    → switchView('search')                  // 切换到搜索列表视图

清空搜索框 → 返回网格视图

点击标签 chip → searchByTag(tagName)
  → 设置搜索框值 → searchNotes(tagName, 'tag')
```

#### 4.2.3 笔记查看与右键菜单流程

```
左击笔记卡片/搜索结果
  → window.viewNote(id)
    → openEditor(id, true)
      → 标题输入框 readonly
      → 内容文本域 readonly
      → 标签仅展示（不可切换）
      → 隐藏"保存"/"取消"按钮
      → 标题显示"查看笔记"

右键笔记卡片/搜索结果
  → window.showContextMenu(event, noteId)
    → 阻止默认右键菜单
    → 定位菜单到鼠标位置
    → 菜单项：查看 / 编辑 / 置顶(取消置顶) / 删除
  → 点击菜单项 → window.handleContextAction(action)
    → 'view': 打开只读查看
    → 'edit': 打开编辑器(编辑模式)
    → 'pin': 切换置顶
    → 'delete': 确认后删除
  → 点击页面其他区域 → 关闭菜单
```

#### 4.2.4 笔记软删除与回收站流程

```
用户点击卡片上的 "✕" 删除按钮
  → confirm("确定要删除这条笔记吗？")
    → [确认]
      → deleteNote(id)
        → window.go.main.App.DeleteNote(id)
          → app.go:DeleteNote()
            → noteService.Delete(id)
              → GORM: db.Delete(&Note{}, id)    // 设置 deleted_at 时间戳
        → loadNotes()                           // 刷新列表
    → [取消] 无操作

用户进入回收站（通过顶部 ☰ → 回收站）
  → switchView('trash')
    → loadTrashNotes()
      → GetTrashNotes(1, 100)
        → GORM: Unscoped().Where("deleted_at IS NOT NULL")
  → 渲染回收站列表（含"全部恢复"和"全部清空"按钮）
    → 恢复单个: RestoreNote(id) → GORM: Update("deleted_at", nil)
    → 永久删除单个: PermanentDeleteNote(id) → GORM: Unscoped().Delete()
    → 全部恢复: RestoreAll() → GORM: Update 所有回收站笔记
    → 全部清空: EmptyTrash() → GORM: Unscoped().Delete 所有回收站笔记
```

#### 4.2.5 数据库导出流程

```
用户进入数据管理页面（通过顶部 ☰ → 数据管理）
  → loadDataStats()
    → GetDataStats()
      → noteService.GetStats() + tagService.Count() + os.Stat(dbPath) 获取文件大小
    → 渲染 4 张统计卡片（笔记总数/标签总数/回收站数/数据库大小）
      → 数字使用 countUp 动画递增显示

用户点击「导出数据」
  → window.go.main.App.ExportDataWithDialog()
    → app.go:ExportDataWithDialog()
      → 创建临时路径 → noteService.ExportBackup(tempPath)
        → GORM: VACUUM INTO tempPath       // SQLite 原生在线备份
      → runtime.SaveFileDialog()            // 弹出原生保存对话框
      → fs.CopyEx(tempPath, filePath, true) // go-kit/fs 原子性复制
      → 清理临时文件
    → 返回 "导出成功：/path/to/file.db"
    → 前端 Toast 弹出提示
```

#### 4.2.6 数据库导入流程

```
用户点击「导入数据」
  → runtime.OpenFileDialog()                // 弹出原生文件选择器，过滤 *.db
  → 用户选择 .db 文件
  → window.go.main.App.ImportDatabaseWithDialog()
    → app.go:ImportDatabaseWithDialog()
      → Step 1: 备份当前数据库 → fs.CopyEx(dbPath, backupPath)   // go-kit/fs
      → Step 2: sqlDB.Close() 关闭旧 SQLite 连接
      → Step 3: fs.CopyEx(filePath, dbPath) 覆盖数据库文件
      → Step 4: database.InitDB(dbPath) 重新打开数据库
      → Step 5: 重建 NoteService/TagService/SettingService
      → Step 6: 清理 .bak 备份
      → [任何步骤失败] 自动从 backupPath 恢复原始文件 + 重连
    → 返回 ImportResult{Message, SuccessCount}
    → 前端 Toast 提示结果 + 自动刷新笔记/标签/统计

用户点击「恢复出厂设置」
  → 二次确认对话框
  → window.go.main.App.ResetDatabase()
    → app.go:ResetDatabase()
      → 清空 notes、tags、note_tags、settings 所有表
      → 重新注入 6 个默认标签
    → 前端切回首页 + loadNotes() 刷新笔记列表
    → 统计卡片刷新显示零值
```

### 4.3 代码质量分析

| 维度 | 评价 | 具体表现 |
|------|------|----------|
| **命名规范** | 良好 | Go 使用 CamelCase（`NoteService`、`GetAll`），JS 使用 camelCase + snake_case（`loadNotes`, `created_at`），CSS 使用 kebab-case（`note-card`, `search-box`） |
| **注释覆盖** | 良好 | 所有公开方法和关键函数均有 `//` 行注释说明，JS 函数有 JSDoc 风格注释 |
| **硬编码** | 轻量 | 数据库路径通过 `database.DefaultDBPath()` 统一获取（`~/.jot/data/jot.db`）；Mock 数据硬编码在 JS 中 |
| **冗余代码** | 无 | 无明显重复代码，Service 间职责分离清晰 |
| **错误处理** | 中等 | Go 层完整处理 GORM 错误（ErrRecordNotFound），JS 层有 try/catch 包裹后端调用；前端有导入结果 Toast 提示 |
| **安全性** | 基础 | Go 层使用了参数化查询（GORM 防 SQL 注入），JS 层有 `escapeHtml()` 防 XSS；但无输入长度/格式校验 |

---

## 五、技术栈评估

### 5.1 技术栈清单

| 层级 | 技术组件 | 版本 | 用途 | 社区活跃度 |
|------|----------|------|------|------------|
| **桌面框架** | Wails v2 | v2.12.0 | 跨平台桌面应用框架（Go 后端 + 原生 WebView） | 🟢 活跃，Wails 社区增长迅速 |
| **后端语言** | Go | 1.26.1 | 编译型，高性能，跨平台 | 🟢 主流 |
| **ORM** | GORM | v1.31.1 | Go ORM 库，数据库操作抽象 | 🟢 活跃，Star 37k+ |
| **SQLite 驱动** | glebarez/sqlite | v1.11.0 | 纯 Go SQLite 驱动（免 cgo） | 🟢 活跃，纯 Go 实现 |
| **底层 SQLite** | modernc.org/sqlite | v1.23.1 | CGo-free SQLite 实现 | 🟢 活跃 |
| **前端构建** | Vite | v3.2.11 | 前端构建工具 | 🟡 较老版本（v3 已 EOL，建议 v5/v6） |
| **前端技术** | 原生 HTML/CSS/JS | — | 无框架依赖 | 🟢 稳定 |

### 5.2 技术栈选型评价

| 选型 | 评价 |
|------|------|
| **Wails v2** | ✅ 适合小型桌面应用。相比之下 Electron 更重（>100MB），Wails 打包体积小（<10MB），内存占用低 |
| **Go + GORM** | ✅ 适合本项目的 CRUD 密集型场景，GORM 降低 SQL 编写量 |
| **glebarez/sqlite** | ✅ 纯 Go 实现，免 CGO 编译。解决了 Windows 下 CGO 交叉编译的痛点 |
| **原生 HTML/CSS/JS** | ✅ 适合小项目，无需引入 React/Vue 等框架的打包体积和学习成本 |
| **Vite 3.x** | ⚠️ 版本较旧（v3）。Wails 默认模板锁定 v3，升级需手动修改。v3 已停止维护 |

### 5.3 版本兼容性问题

| 问题 | 描述 |
|------|------|
| GORM v1.31 vs Go 1.26 | ✅ 兼容。GORM v1.31.1 支持 Go 1.23+ |
| glebarez/sqlite v1.11 vs modernc.org/sqlite v1.23 | ✅ 兼容，glebarez 为 modernc 的上层封装 |
| Wails v2.12 vs Go 1.26 | ✅ 兼容，Wails v2.12 支持 Go 1.21+ |
| Vite 3.x vs Node 版本 | ⚠️ 需确认开发环境的 Node.js 版本是否在 Vite 3 支持范围内 |

---

## 六、补充分析

### 6.1 扩展性评估

| 维度 | 评价 |
|------|------|
| **新增数据模型** | 容易。在 `models/` 新建文件，`database/db.go` 中追加 `AutoMigrate`，新增对应 Service |
| **新增 API 接口** | 容易。在对应 Service 新增方法，`app.go` 新增绑定方法，前端 `main.js` 新增调用函数 |
| **前端新增视图** | 容易。在 `index.html` 新增 `div.view`，`main.js` 中 `switchView()` 添加分支 |
| **替换数据库** | 中等。改驱动 + 适配 SQL 方言，Service 层使用 GORM 可降低迁移成本 |
| **UI 框架集成** | 容易。原生 HTML/JS，可逐步引入 Vue/React |

### 6.2 性能关键点

| 关注点 | 描述 | 影响 |
|--------|------|------|
| **SQLite 单连接** | `SetMaxOpenConns(1)` | ✅ SQLite 本身不支持并发写入，限制为 1 是正确做法 |
| **N+1 查询** | `Preload("Tags")` 确保了预加载 | ✅ 避免了遍历笔记时逐个查询标签 |
| **LIKE 搜索** | `LIKE '%keyword%'` 无法利用索引 | ⚠️ 大数据量（>10万条）时性能下降，建议引入全文检索（FTS5） |
| **全量加载** | `GetNotes(1, pageSize)` 首屏只加载当前分页大小；底部滚动/键盘快捷键可按需追加剩余页 | ⚠️ 分页可配（20-100），大数量下分页加载已优化 |
| **前端 DOM 操作** | 每次 `loadNotes()` 全量重新渲染所有卡片 | ⚠️ 笔记数量>500时可能有卡顿，建议虚拟滚动或增量更新 |

### 6.3 异常处理分析

| 层级 | 处理方式 | 不足 |
|------|----------|------|
| **Go Service 层** | GORM 错误处理完整（ErrRecordNotFound），返回 `error` | 缺少自定义错误类型，错误信息不够结构化 |
| **Go App 层** | 透传 Service 层 error 到 Wails；导出使用 runtime.SaveFileDialog | 无额外的错误包装/日志记录 |
| **JS 前端** | `try/catch` 包裹后端调用，console.error 输出；导入结果有 Toast 显示 | 其他场景仍缺少用户可见的错误提示 |
| **数据库初始化** | Panic 策略（InitDB 失败直接 panic） | ✅ 数据库不可用是致命错误，panic 合理 |

### 6.4 安全分析

| 维度 | 状态 | 说明 |
|------|------|------|
| **SQL 注入** | ✅ 安全 | 全部使用 GORM 参数化查询或 `?` 占位符 |
| **XSS** | ✅ 防护 | `escapeHtml()` 对用户输入做 HTML 转义后渲染 |
| **文件路径** | ✅ 安全 | 数据库路径通过 `DefaultDBPath()` 统一定位到 `~/.jot/data/jot.db`；导出路径由 `runtime.SaveFileDialog` 提供 |
| **输入校验** | ⚠️ 基础 | 仅检查空标题，无长度/格式限制 |

---

## 七、项目核心特点

1. **轻量架构**：原生 WebView（Wails）+ 纯 Go 后端，构建产物小、内存占用低
2. **免 CGO 编译**：glebarez/sqlite 纯 Go 实现，Windows 交叉编译无障碍
3. **Service Layer 分离**：业务逻辑与界面绑定完全解耦，便于测试和扩展
4. **前后端松耦合**：前端通过 Wails Binding 调用后端，无直接依赖
5. **降级友好**：前端内置 Mock 数据，后端未绑定时仍可独立预览 UI
6. **组件化渲染**：前端渲染函数独立，数据驱动 DOM 更新
7. **原生桌面体验**：通过 Wails runtime 集成系统保存对话框（SaveFileDialog）
8. **键盘驱动**：支持 Ctrl+F/Ctrl+N/PgUp/PgDn（触底加载下一页）/Ctrl+Home/Ctrl+End（加载全部到底）/E（回首页）及数字键 1-5 快捷导航
9. **窗口焦点自动刷新**：`visibilitychange` 事件监听，切回应用自动刷新数据
10. **批量管理**：批量选择（全选从后端拉取全部 ID）、批量删除（支持撤销）、选中计数联动
11. **动画系统**：全局 CSS 变量驱动动画体系（13 个 keyframes + 20 项交错延迟工具类），16 项动画覆盖所有交互场景，`prefers-reduced-motion` 降级支持
12. **一键备份还原**：`~/.jot/backup/jot-backup.db` 固定路径，覆盖式备份，还原带确认弹窗
13. **按钮交互反馈**：按压缩放 + 涟漪效果 + 危险按钮全红按压态 + 禁用态灰化，覆盖所有交互场景
14. **统一通知系统**：右上角浮动通知组件，4 种类型色彩区分，自动消失/手动关闭/撤销回调
15. **MD 实时预览编辑器**：纯文本/预览双模式切换（`Ctrl+L` 快捷键），marked + highlight.js 实时渲染，底部状态栏切换按钮
16. **置顶不更新时间**：后端 `TogglePin` 使用 `UpdateColumn("pinned")` 跳过 GORM 的 `UpdatedAt` 自动更新
17. **新建默认时间标题**：新建笔记自动填入 `YYYY-MM-DD HH:mm ☺️` 格式标题
18. **批量标签操作**：批量模式下 +标签/-标签 按钮，标签选择弹窗（选中态切换 + 确认按钮 + 已选计数），操作后不退出批量模式保持选中状态
19. **右键导出为 Markdown**：右键菜单「导出」→ 标题特殊符号→下划线 → 系统保存对话框 → `.md` 文件写入
20. **笔记本侧边栏系统**：三段式侧栏设计（header/list/footer），使用 `--card-bg`/`--bg-secondary` + `color-mix()` 配色过渡。书签隐喻（左侧 3px 指示条 + hover 微弹）。新建按钮移入 header 标题右侧（简洁 `+`）。footer 与按钮铺平融为一体，折叠动画 `white-space: nowrap` 防文字换行。6 主题自适应
21. **笔记本 CRUD**：后端 NotebookService（Create/Update/Delete/DeleteWithNotes/GetAll/GetAllNotesCount/EnsureDefaultNotebook），默认笔记本（ID=1）不可删不可改名。前端 rename 防重名校验（`isNameTaken`），删除对话框带 checkbox（迁移笔记到默认 / 连带永久删除笔记），删除活跃笔记本后自动切换到默认笔记首页
22. **笔记本笔记隔离**：所有笔记查询按 `activeNotebookId` 过滤（`GetNotes`/`SearchNotes` 均接受 `notebookID` 参数）。`selectAllIds` 使用 `GetNoteIDsByNotebook` 替代 `GetAllNoteIDs`，确保全选仅限当前笔记本笔记。各笔记本笔记数 badge 自动同步更新

---


## 八、待优化点

### 优先优化
1. **Vite 升级**：Vite 3 → Vite 5/6，获取更好的构建性能和生态支持

### 中期优化
2. **全文搜索**：引入 SQLite FTS5，替代 `LIKE %keyword%` 模糊搜索

### 架构层面
4. **前端框架**（可选）：若功能持续增长，可引入 Vue/React 管理复杂状态
5. **单元测试**：补充 Service 层的 `_test.go` 测试文件

### 已实现
- ✅ **滚动/分页加载**：首屏按分页大小加载，触底按需追加下一页，Ctrl+End 一次加载全部
- ✅ **Pagination 设置**：操作 CRUD 后重置到第 1 页
- ✅ **PgDn/ESC 快捷键**：PgDn 触底主动加载，ESC 退出子视图回首页
- ✅ **加载动画**：触底加载时显示 CSS 旋转动画 + 最短 1s 显示时间
- ✅ **排序设置**：设置页支持按更新时间/创建时间/名称排序，iOS 风格分段控件（3 等分滑动指示器），持久化到 Setting `sort_order`
- ✅ **分页大小设置**：设置页支持 20/40/60/80/100 分页档位，iOS 风格分段控件
- ✅ **Go 内部包重构**：database/fontutil/models/services 移至 internal/ 目录
- ✅ **主题系统**：6 个主题 — default（暖灰）、nord（北极蓝调）、monokai-pro（荧光粉墨）、light（亮白蓝强调）、tokyo-night（靛紫夜幕）、dark（纯黑琥珀强调）。CSS 变量体系统一，`data-theme` 属性切换，iOS 风格分段控件选择（6 等分滑动指示器），持久化到 Setting `theme`
- ✅ **标签选中态视觉优化**：未选中标签压暗去饱和，选中亮色 + ✓ 前缀 + 脉冲动画 + 外发光
- ✅ **悬浮操作按钮**：右下角 `+`（新建）和 `↑`（回到顶部），`↑` 滚动超过 300px 时淡入
- ✅ **滚动条美化**：主内容区滚动显示/停止 1s 淡出，6px 半透明灰条，6 主题独立配色；textarea `flex:1` 独占滚动，编者面板高度固定 85vh
- ✅ **快速笔记模式**：设置页 iOS 风格开关，启用后启动自动全屏编辑，保存/取消后回到网格首页
- ✅ **Markdown 渲染查看**：查看模式使用 marked + highlight.js 渲染 Markdown 为 HTML，代码块语法高亮，编辑模式不变
- ✅ **关于页面**：品牌名点击弹出覆盖层，展示项目名/简介/版本号/项目地址链接；版本号由 verman 库通过 ldflags 注入；项目链接点击在默认浏览器打开
- ✅ **快捷键说明**：更多菜单「帮助」或按数字键 `5` 弹出模态覆盖层，展示全部快捷键列表；与关于页面相同遮罩/卡片动画
- ✅ **窗口焦点自动刷新**：`visibilitychange` 事件监听，切回应用自动刷新 grid/trash/data 视图
- ✅ **数据库路径迁移**：从 `./data/jot.db` 迁移到 `~/.jot/data/jot.db`（`DefaultDBPath()`），应用/种子脚本统一
- ✅ **批量全选加载全部 ID**：全选时调用 `App.GetAllNoteIDs()` 获取所有笔记 ID，一次选中全部
- ✅ **搜索框逻辑修复**：回车立即搜索，清空自动刷新笔记列表（footer 清理 bug 修复）
- ✅ **交互体验与动画增强**：16 项动画/过渡增强 — 视图切换过渡、卡片网格交错入场/hover/active 动画、统计卡片 count-up 数字动画、回收站 shrinkOut 删除/恢复动画、设置 toggle-switch/字体下拉展开收起、帮助/关于模态弹簧动画、编辑器模态缩放淡入 + 品牌色条展开 + 标签脉冲、Markdown 淡入/代码块交错、快速笔记全屏过渡、批量模式滑入滑出 + 复选框交错、Toast 滑入滑出/堆叠上移、右键菜单缩放展开/收起、更多菜单 transform-origin 缩放、主题切换 300ms 平滑过渡；全局 prefers-reduced-motion 降级支持
- ✅ **搜索结果滚动条贴边**：搜索列表容器负边距贴靠窗口右边缘，列表内容保持 32px 内边距
- ✅ **数据管理三层布局**：第一层统计卡片（4 列网格）、第二层操作按钮（纵向 card 列表 + 恢复出厂设置危险按钮）、第三层数据目录
- ✅ **数据库大小统计**：4 张统计卡片包含数据库文件大小，通过 os.Stat 读取并格式化显示 B/KB/MB
- ✅ **导出改为 DB 文件备份**：VACUUM INTO → fs.CopyEx → .db 文件，完整备份所有数据（含回收站/设置/关联关系）
- ✅ **导入改为文件覆盖**：6 步流程（备份→关连接→覆盖→重连→重建→清理），出错自动回滚
- ✅ **恢复出厂设置刷新首页**：重置后自动切回首页 + loadNotes()，空库显示「暂无笔记」
- ✅ **打开数据目录**：数据管理第三层按钮，exec.Command("explorer") 在文件管理器中打开数据库目录
- ✅ **Ctrl+A/Ctrl+D 批量快捷键**：全局 Ctrl+A 阻止全选，批量模式 Ctrl+A 全选 Ctrl+D 取消全选
- ✅ **lint 0 issues**：golangci-lint errcheck 等 7 个问题全部修复，0 issues 通过
- ✅ **一键备份还原**：`~/.jot/backup/jot-backup.db` 固定路径，每次覆盖备份；还原带自定义确认弹窗 + 按钮加载状态；备份信息标签（时间/大小/绿色标识）
- ✅ **数据管理页面重构**：除统计卡外所有区域改卡片风格（圆角/阴影/边框），标题与设置页统一（0.938rem 无装饰条），view-header 整体左移 16px
- ✅ **按钮点击反馈增强**：所有 `data-action-btn` 按压缩放（0.975）+ 涟漪 `::after` 闪现；危险按钮按压全红底白字；禁用态灰化+禁止点击；统计卡卡 hover 上浮 + active 按压 + 交错入场动画
- ✅ **全局 overscroll-behavior 禁用**：`body` + `#mainContent` 设置 `overscroll-behavior: none`，双指触控板滑动不回弹
- ✅ **统一通知系统**：删除旧底部堆叠 toast（`#undoToast`/`showToast`/`showUndoToast`），替换为 `NotificationManager` 单例类，右上角浮动通知组件。支持 4 种通知类型（success/error/warning/info）+ undo 类型，左侧色标条 + 图标区分，入场 `notifSlideIn` 弹性滑入，出场 `notifSlideOut` 滑出淡出。`nm.show(msg, type, duration?)` 自动 3s 消失，`nm.showUndo(msg, onUndo, duration?)` 带撤销按钮 5s 消失。替换全部 34 个旧调用点，删除旧函数 5 个 + 状态变量 4 个。设置页保存操作后发通知提示。创建/删除标签操作发通知提示
- ✅ **MD 实时预览编辑器**：编辑器新增纯文本/预览双模式切换，底部状态栏中间胶囊按钮组。查看模式自动切预览，编辑模式默认纯文本。使用已有 marked + highlight.js 渲染，300ms 防抖自动更新。预览区隐藏滚动条，各滚各的不同步
- ✅ **移除 Google Fonts CDN**：删除 DM Sans 字体依赖，默认字体改为 `system-ui, -apple-system, sans-serif`，完全无外部网络请求
- ✅ **只读模式标签过滤**：查看页面标签只显示该笔记已添加的标签，不再展示全部标签
- ✅ **README 精简**：删除内部 API 文档/使用示例/配置选项/项目结构/测试说明，聚焦用户视角的使用和安装
- ✅ **CSS 清理**：删除 Section L 旧确认弹窗死代码（38 行）、删除 style.css 中重复的 `cardEnter` 关键帧（app.css 版本生效）、补齐 6 主题 `--bg-secondary`/`--text-tertiary` 变量（此前引用未定义）
- ✅ **确认弹窗修复**：HTML 类名 `confirm-dialog-overlay` 指向已被删除的旧 CSS，修复为 `confirm-overlay` + 适配 CSS/JS，弹窗居中 + 模糊背景 + 淡入动画
- ✅ **标签删除刷新笔记**：`deleteTag()` 末尾追加 `await loadNotes()`，删除标签后卡片网格立即更新，不再显示已删除的标签
- ✅ **全屏动画优化**：CSS 去掉 `!important`，尺寸/圆角加入默认 transition；JS 移除 inline transition 逻辑 + transitionend 监听，纯 classList 切换；overlay 全屏态加深背景 + 升模糊至 10px；editor-body 过渡期间淡出/淡入防内容跳跃
- ✅ **编辑器快捷键放行**：编辑器打开时 Ctrl+Home/End 和 PgUp/PgDn 不拦截，交由 textarea 原生处理（光标到行首/行末，上下翻页）
- ✅ **ESC 关闭编辑器**：编辑器打开时按 ESC 触发 `closeEditor()`，关闭查看/新建/编辑弹窗
- ✅ **Ctrl+L 切换编辑器模式**：编辑器打开时按 `Ctrl+L` 切换纯文本/预览模式，已在快捷键说明页注册
- ✅ **新建默认时间标题**：新建笔记自动填入当前日期时间 `2026-06-06 14:30 ☺️`
- ✅ **置顶不更新时间**：`TogglePin` 改为 `UpdateColumn("pinned")`，跳过 `UpdatedAt` 自动更新
- ✅ **右键导出为 Markdown**：右键菜单「导出」→ 标题特殊符号→下划线 → 系统保存对话框 → `.md` 文件（`# 标题\n\n内容` 格式写入）
- ✅ **批量标签操作**：批量工具栏新增 +标签/-标签 按钮；点击弹出标签选择弹窗（毛玻璃背景 + 弹入动画），所有标签以彩色圆点展示；添加/移除模式统一为「点击标签切换选中态 → 确认按钮执行」；底部确认按钮显示已选数量；移除模式不可移除标签灰色禁用；操作后不退出批量模式保持选中状态；空态提示「当前选中的笔记中没有可移除的标签」
- ✅ **草稿自动保存与恢复**：新建 `drafts` 表（仅 1 行 ID=1，存 title/content），后端 `DraftService`（SaveDraft upsert / GetDraft / ClearDraft）+ app.go 绑定 3 个方法。前端 `startAutoSave()` 扩展：有 ID → `UpdateNote`（编辑已有笔记），无 ID → `SaveDraft`（新建笔记草稿），3s 防抖，标题内容均空不保存。`loadNotes()` 完成后延迟 1s 检测草稿（编辑器打开时跳过），有则弹窗「发现未保存的草稿，是否恢复？」恢复→填入编辑器、放弃→清除。清除时机：保存成功（`createNote`）、取消按钮（明确放弃）、恢复弹窗恢复/放弃。叉号/点击蒙层/ESC 不清除（保留供下次恢复）。快速笔记启用启动时 `loadQuickNoteSetting()` 自动查草稿并填入编辑器。底部栏 autoSaveIndicator 提示文字 2s 后清空（`textContent=''`），布局自动恢复
- ✅ **设置页首次打开白屏修复**：根因为 `updateFontSettingsUI()` 调用 `renderFontFamilyOptions()` 在首次显示时创建 200+ 带 `style="font-family:..."` 的字体选项 DOM 节点，浏览器首次布局计算耗时 1-2 秒。修复：`updateFontSettingsUI()` 移除 `renderFontFamilyOptions()` 调用，改为仅在用户点击下拉触发器时渲染（已有点击处理逻辑），设置页首次打开不再含大量字体节点参与布局，白屏消除
- ✅ **笔记类型功能**：Note 模型新增 `NoteType string` 字段（`gorm:"size:20;default:text"`），支持 `"text"`（纯文本）和 `"markdown"`（Markdown）两种类型。新建笔记默认选中「纯文本」（text）。纯文本模式底部隐藏「编辑/预览」切换按钮（`#editorModes`），仅显示纯文本编辑区。查看模式：`note_type="text"` 或 `""` 时内容以 `<pre>` 原始格式显示跳过 marked 解析；`note_type="markdown"` 走现有渲染流程。创建/更新笔记时 `note_type` 字段随请求写入，旧笔记 `note_type=""` 默认按 text 处理
- ✅ **笔记类型切换器位置重构**：类型切换器从 footer 底部（与模式切换器冲突）移到 header 标题行右侧紧凑胶囊（`.editor-title-row`），再重构为 header 工具栏中的 32×32 T/M 图标按钮（`#editorTypeToggle`），与全屏/关闭按钮同尺寸同风格。查看只读模式隐藏
- ✅ **置顶按钮重新设计**：从旧 `.card-pin-badge` 左侧指示器 + hover 显示按钮改为始终可见的圆形图标按钮（`border-radius: 50%`）。分 4 级透明度：默认 20% → 卡片 hover 50% → 按钮自身 hover 100% + 放大 1.15x → 已置顶 100% + `accent-lighter` 背景 + 阴影。star 使用字体加粗区分状态
- ✅ **置顶操作局部更新**：`togglePin()` 切换后不再 `await loadNotes()` 全量重载，改为本地更新 `state.notes` + `renderCardGrid('none')`，省掉后端请求和整页入场动画
- ✅ **批量模式 pin 按钮可见**：批量模式下 pin 按钮不再 `display: none`，与 checkbox 并存显示；添加 `.disabled` class（`pointer-events: none`），退出批量模式恢复交互
- ✅ **批量置顶/取消置顶**：后端 `note_service.go` 新增 `BatchPinNotes(ids []uint, pin bool)` + `app.go` 绑定。批量操作栏新增 `#batchPinBtn` 按钮；`updateBatchBar()` 根据选中笔记状态动态切换文字（全部已置顶→"取消置顶"，否则→"置顶"）；`batchPinSelected()` 调用后端 + 本地同步 + 通知提示
- ✅ **Seed 工具 note_type**：`tools/seed/main.go` 24 条模拟笔记 50/50 分配 text/markdown 类型（含 Checklist/纯列表的为 text，含 headings/代码块/表格的为 markdown）
- ✅ **笔记本侧边栏三段式设计**：Header（--card-bg 56px + 标题 + `+` 按钮）→ List（--bg-secondary，书签隐喻笔记本项）→ Footer（color-mix 55/45 铺平，无圆角无 padding）。6 主题自适应，折叠动画防文字换行
- ✅ **笔记本 CRUD 系统**：NotebookService 7 方法（Create/Update/Delete/DeleteWithNotes/GetAll/GetAllNotesCount/EnsureDefaultNotebook）。默认笔记本（ID=1）不可删不可改名。重命名防同名检测
- ✅ **删除笔记本连带选项**：自定义确认弹窗内嵌 checkbox，勾选→硬删除笔记+软删笔记本，不勾选→笔记迁到默认笔记本。删除活跃笔记本后自动回默认首页
- ✅ **笔记本笔记隔离**：所有查询按 `activeNotebookId` 过滤，`selectAllIds` 使用 `GetNoteIDsByNotebook`。badge 在 6 种笔记操作后自动同步更新
- ✅ **切换笔记本回首页**：`switchNotebook()` 强制 `switchView('grid')` 回到当前笔记本笔记首页
- ✅ **笔记本侧栏折叠动画优化**：`white-space: nowrap` + `overflow: hidden` 防按钮文字折叠时竖排换行

---

## 九、关键记忆点

| 记忆点 | 内容 |
|--------|------|
| **项目名** | Jot（卡片式笔记桌面应用） |
| **技术栈** | Wails v2 + Go 1.26 + GORM v1.31 + glebarez/sqlite + 原生 HTML/CSS/JS |
| **数据库** | SQLite（`~/.jot/data/jot.db`），免 CGO 纯 Go 驱动，路径由 `DefaultDBPath()` 统一获取 |
| **后端结构** | `main.go → app.go → services/ → models/` + `database/` + `fontutil/` |
| **绑定方法数** | 55 个（19 个 Note 相关 + 6 个 Tag 相关 + 6 个 Notebook 相关 + 6 个数据管理 + 3 个字体设置 + 4 个排序/分页设置 + 2 个关于页面 + 3 个 Draft 草稿 + 3 个备份还原）|
| **前端视图** | 8 个：卡片网格、编辑器（模态框）、搜索结果、设置、数据管理、回收站、关于页面（覆盖层）、快捷键说明（覆盖层）|
| **前端代码量** | ~3800 行 JS + ~2974 行 CSS + ~458 行 CSS 全局样式（含 6 主题 CSS 变量 + 20+ keyframes 动画）|
| **数据流向** | 用户操作 → JS 事件 → Wails Bridge → app.go → Service → GORM → SQLite |
| **核心字段** | Note: id/title/content/color/pinned/created_at/updated_at/deleted_at/tags |
| **接口风格** | RESTful 风格方法命名（CRUD + Search + Toggle + GetTrash + Restore + Stats + Export/Import）|
| **排序规则** | `pinned DESC, updated_at DESC`（置顶优先，最新在前） |
| **交互特点** | 左击查看（只读），右击菜单（查看/编辑/置顶/删除），输入框 250ms 防抖自动搜索/回车立即搜索，标签 chip 可点击搜索，Ctrl+F 聚焦搜索，Ctrl+N 新建笔记 |
| **卡片操作** | 右上角 hover 只显示置顶按钮，编辑/删除移至右键菜单（纯文字无图标） |
| **布局** | topbar（品牌/搜索框/新建/+更多菜单），主内容区（卡片网格/搜索/设置/数据管理/回收站视图）；设置/数据管理/回收站页面的 view-header 结构统一（`← 返回` + 居中标题 + view-controls），内容区均设置 `max-width` + `margin: 0 auto` 居中 |
| **键盘快捷键** | Ctrl+F 搜索 / Ctrl+N 新建 / Ctrl+L 编辑器切换模式 / PgUp 上翻 / PgDn 下翻或触底加载下一页 / Ctrl+Home 顶部 / Ctrl+End 加载全部并到底 / E 退出子视图回首页 / 数字键 1=笔记首页 2=展开/折叠侧栏 3=数据管理 4=回收站 5=设置 6=帮助；输入框内数字键不触发；编辑器打开时 Ctrl+Home/End 和 PgUp/PgDn 不拦截，交由 textarea 原生处理 |
| **回收站** | 通过顶部 ☰ → 回收站 进入，支持全部恢复/全部清空 |
| **数据管理** | 通过顶部 ☰ → 数据管理 进入，含统计卡片 + 数据操作/快速备份/数据目录三个卡片分区 |
| **导出** | `ExportDataWithDialog()` 调用 `runtime.SaveFileDialog`，VACUUM INTO 创建 SQLite 压缩副本 → fs.CopyEx 到用户选择路径，输出 .db 文件 |
| **导入** | `ImportDatabaseWithDialog()` 弹出原生文件选择器（*.db），6 步流程：备份 → 关连接 → 覆盖文件 → 重开数据库 → 重建 Service → 清理备份；任何步骤失败自动从 .bak 恢复 + 重连；前端 Toast 提示 + 自动刷新 |
| **恢复出厂设置** | `ResetDatabase()` 清空 notes/tags/note_tags/settings 所有表，重新注入 6 个默认标签；前端切回首页 + loadNotes() 刷新笔记列表 |
| **数据管理统计卡片** | 4 张卡片（笔记总数/标签总数/回收站数/数据库大小），去图标纯文字居中，数字使用 countUp 动画递增显示；最大宽度 760px + margin:0 auto 居中 |
| **数据管理布局** | 三层结构：第一层「数据统计」（4 卡片网格 4 列，无标题）、第二层「数据操作」卡片区（导出/导入水平并排 + 恢复出厂设置独占一行）、第三层「快速备份」卡片区（备份信息标签 + 备份/还原按钮）、第四层「数据目录」卡片区（单按钮 max-width:400px）。所有卡片使用 `.data-section-card` 样式（圆角/阴影/边框/内边距），标题与设置页统一（0.938rem 无装饰条）。最大宽度 760px + margin:0 auto 居中 |
| **数据管理滚动条** | 与首页一致的覆盖式滚动条（6px 半透明灰 + 自动隐藏），`#viewData.view { padding-right: 0 }` 贴靠窗口右边缘 |
| **打开数据目录** | `app.go:OpenDataDir()` 调用 `exec.Command("explorer", dir)` 在文件管理器中打开数据库目录，数据管理第三层按钮 |
| **一键备份** | 备份到 `~/.jot/backup/jot-backup.db`（固定路径，每次覆盖）。前端按钮显示 loading 状态「⏳ 备份中…」+ disabled。备份后信息标签绿色标识 `✓ 已有备份 — 时间，大小`，无备份显示「暂无备份」|
| **一键还原** | 从 `jot-backup.db` 还原，先弹出应用自定义确认弹窗，确认后按钮显示 loading 状态。还原失败自动从 .bak 回滚。成功后刷新笔记/标签/统计 |
| **统一通知系统** | 删除旧底部堆叠 toast（`#undoToast`/`showToast`/`showUndoToast`），替换为 `NotificationManager` 单例类，从右上角浮动弹出。支持 4 种通知类型（success/error/warning/info）+ undo 类型，左侧色标条 + 图标区分，入场 `notifSlideIn` 弹性滑入，出场 `notifSlideOut` 滑出淡出。`nm.show(msg, type, duration?)` 自动 3s 消失，`nm.showUndo(msg, onUndo, duration?)` 带撤销按钮 5s 消失。替换全部 34 个旧调用点，删除旧函数 5 个 + 状态变量 4 个 |
| **自定义确认弹窗** | `.confirm-overlay` 遮罩层 + `.confirm-dialog` 卡片（背景模糊，弹簧动画），确定按钮红色 danger 色。用于还原确认和回收站操作。复用已有 `showConfirmDialog()` 函数 |
| **Ctrl+A/Ctrl+D 快捷键** | 全局阻止默认 Ctrl+A；批量模式下 Ctrl+A = 全选所有笔记、Ctrl+D = 取消全选 |
| **lint 状态** | `golangci-lint run ./...` 0 issues（errcheck 等 7 个问题已全部修复）|
| **Mock 数据** | `getMockNotes()` 3 条示例笔记，`getMockTags()` 3 个标签；通过 `mockNotes` 可变变量持久化修改 |
| **Seed 工具** | `tools/seed/main.go` 默认注入 `~/.jot/data/jot.db`（支持命令行参数指定路径）；含 24 条覆盖多领域的测试笔记 + 5 个标签 |
| **右键菜单** | 纯文字无图标，`min-width: 120px` |
| **更多菜单** | 含全部笔记/数据管理/回收站/设置/帮助五个选项，分隔线分组，`min-width: 120px` |
| **Spec 位置** | `.trae/specs/add-card-note-app/`、`.trae/specs/add-data-management/`、`.trae/specs/add-font-settings/`、`.trae/specs/add-quick-note-mode/`、`.trae/specs/add-md-rendering/`、`.trae/specs/add-about-page/`、`.trae/specs/add-misc-improvements/`、`.trae/specs/enhance-interaction-animation/`、`.trae/specs/add-draft-auto-save/` |
| **字体设置** | 设置页面新增「字体设置」分区，字体族下拉（搜索+↑↓/Enter/Escape 键盘导航）+ 大小预设/自定义。下拉选项采用延迟渲染策略：`updateFontSettingsUI()` 不调用 `renderFontFamilyOptions()`，仅用户首次点击下拉触发器时渲染 200+ 字体选项 DOM，避免首次打开设置页时大量字体节点参与布局导致 1-2 秒白屏 |
| **字体枚举** | `fontutil/fonts_windows.go` 使用 Win32 GDI EnumFontFamiliesW API 直接枚举，不依赖第三方库 |
| **配置存储** | `models/setting.go` KV 结构，`services/setting_service.go` Get/Set 读写 |
| **CSS rem 适配** | 所有 font-size 已从 px 转为 rem，通过 `--font-size-base` CSS 变量控制等比缩放 |
| **view-header 统一** | 设置/数据管理/回收站三个功能页的 view-header 均为 `← 返回` + 居中标题 + `view-controls` 结构，保证标题位置一致 |
| **内容区居中** | 设置页 `settings-content` 为 `max-width: 600px` + 居中；数据管理 `data-content` 和回收站 `trash-list` 为 `max-width: 680px` + 居中 |
| **数字键导航** | 键盘快捷键扩展：`1`=首页(清空搜索)、`2`=设置、`3`=数据管理、`4`=回收站、`5`=帮助；输入框内不触发 |
| **排序设置** | 设置页「笔记排序」支持按更新时间/创建时间/名称排序，iOS 风格分段控件（3 等分滑动指示器），持久化到 Setting `sort_order`；后端 `GetAll`/`GetByTag` 动态构建 ORDER BY |
| **分页大小** | 设置页 iOS 风格分段控件：20/40/60/80/100，选中色块带滑动动画，默认 20，持久化到 Setting `page_size` |
| **懒加载** | 所有场景（启动/CRUD）只加载第 1 页，滚动到底部（<200px）自动追加下一页；Ctrl+End 一次加载所有剩余页；底部显示「共 X 条笔记」|
| **加载动画** | CSS 圆环旋转动画（0.8s/infinite），最少显示 1 秒，确保可见 |
| **PgDn 逻辑** | 已到底时直接调用 `loadMoreNotes()` 加载下一页（不走 scroll 事件）；未到底时设置 `_keyboardScroll` 标志阻止 scroll 监听器误触发 |
| **ESC 逻辑** | 按 ESC 依次检查：关于页 → 关闭；全屏 → 退出全屏；编辑器打开 → `closeEditor()`；快捷键弹窗 → 关闭；批量模式 → 退出；搜索视图 → 清空回首页；其他子视图 → 回首页 |
| **Sort & PageSize** | 后端 `GetAll`/`GetByTag` 接受 `sortBy` 参数动态 ORDER BY，新增 4 个绑定方法：`GetSortOrder`/`SetSortOrder`/`GetPageSize`/`SetPageSize` |
| **主题系统** | 6 个主题：default（暖灰）、nord（北极蓝调）、monokai-pro（荧光粉墨）、light（亮白蓝强调）、tokyo-night（靛紫夜幕）、dark（纯黑琥珀强调）。CSS 变量体系统一在 `app.css` 的 `:root`/`[data-theme]` 中，切换通过 `document.documentElement.setAttribute('data-theme', name)`。设置页使用 iOS 风格分段控件（6 等分滑动指示器，宽度 320px），持久化到 Setting `theme`。分段控件动态计算按钮数，支持任意数量按钮 |
| **标签交互优化** | 编辑器标签点击态改为 DOM 类切换（`active`/`clicked`），不再整个重渲染 `renderTagSelector`。选中 → `filter:none + opacity:1 + ✓前缀 + box-shadow`；未选中 → `filter:saturate(0.25) brightness(0.65) + opacity:0.55`；点击脉冲动画 0.25s |
| **编辑器滚动结构** | `.editor-panel` 固定 `height: 85vh`，`.editor-body` 做 flex 布局分配（无滚动），`.editor-textarea` 设为 `flex:1; overflow-y:auto` 独占滚动。textarea 不自带滚动高度（无 `rows`/`min-height` 限制，用 `flex:1; min-height:0` 填满空间）。Editor scrollbar 6px 半透明灰独立主题色 |
| **悬浮按钮 FAB** | 右下角 `position: fixed`，z-index:100。`+`（`#fabNewNote`）始终可见，accent 圆底白字 44px；`↑`（`#backToTopBtn`）默认隐藏，主内容 scrollTop>300 淡入。hover scale(1.08)，active scale(0.95)。距右下角 28px，间距 12px，`+` 在下 |
| **右键菜单滚动锁定** | `showContextMenu` 设 `#mainContent overflow:hidden`，`hideContextMenu` 清空还原。配合 `scrollbar-gutter: stable` 防止宽度抖动 |
| **主内容区滚动条** | 滚动时显示 `var(--scrollbar-thumb)`，停止 1s 后淡出（`.scrolling` 类 0.3s transition）。6 主题独立配色（default `rgba(0,0,0,0.18)`、nord `rgba(46,52,64,0.20)`、monokai-pro `rgba(255,97,136,0.25)`、light `rgba(0,0,0,0.14)`、tokyo-night `rgba(122,162,247,0.25)`、dark `rgba(255,255,255,0.18)`），hover 加深。8px 宽，track 极淡底色 |
| **标签重名提示** | 设置页添加标签时，先在前端 `state.tags` 中查重，已存在则弹出 Toast「该标签已存在」3s 自动消失 |
| **快速笔记模式** | 设置页「快速笔记」iOS 风格开关（`.toggle-switch`），持久化到 Setting `quick_note_enabled`。`init()` 最后调用 `loadQuickNoteSetting()`，启用时自动 `openEditor(null)` + `toggleEditorFullscreen()` + 查询草稿并自动填入（有则填充 title/content）；关闭编辑器时 `closeEditor()` 自动退出全屏恢复网格视图 |
| **Markdown 渲染查看** | 查看模式（只读）将 textarea 替换为 `.md-rendered` div，使用 `marked` 渲染 Markdown 为 HTML。代码块通过 `highlight.js` 着色（注册 JS/Python/CSS/HTML/Bash/JSON）。npm 本地安装（无 CDN），Vite 打包内联。编辑模式 textarea 不变。`.md-rendered` 样式覆盖 h1-h6/列表/引用/代码块/表格/图片/链接，6 主题适配。`.md-rendered` 滚动条隐藏（`::-webkit-scrollbar { display: none }`），内边距 `0.5em 1rem 1rem 1.5em` |
| **关于页面** | 品牌名「Jot」点击弹出全屏覆盖层（`#viewAbout`），居中卡片展示：品牌 Logo（2.5rem 800w accent 色）+ 副标题 + 简介 + 项目地址链接（外链 SVG 图标，点击调用 `runtime.BrowserOpenURL` 打开 `https://gitee.com/MM-Q/jot.git`）+ 底部版本号（等宽字体，通过 verman 库 + ldflags 注入 `GitVersion`）。卡片 `border-radius: 20px`，遮罩 `backdrop-filter: blur(6px)`，弹簧动画 `cubic-bezier(0.16, 1, 0.3, 1)`。关闭方式：✕ 按钮 / 点击遮罩 / ESC 键 |
| **快捷键说明** | 更多菜单「帮助」或按数字键 `5` 弹出模态覆盖层（`#shortcutsView`），居中卡片展示所有快捷键列表，与关于页面相同弹出动画和遮罩样式。关闭方式：✕ 按钮 / 点击遮罩 / ESC 键 |
| **verman 库** | `gitee.com/MM-Q/verman v0.0.19`，通过 `-X gitee.com/MM-Q/verman.V.GitVersion=$(git describe --tags --always --dirty)` 注入版本号。`app.go:GetVersion()` 返回 `verman.V.GitVersion`（纯版本号，不含平台/Go 版本信息）|
| **窗口焦点自动刷新** | `document.addEventListener('visibilitychange', ...)` 在 `init()` 中注册。切回应用自动刷新：grid → `loadNotes()`，trash → `loadTrashNotes()`，data → `loadDataStats()`。编辑中/批量模式时不刷新 |
| **批量全选加载全部 ID** | `toggleSelectAll()` 判断 `selectedNoteIds.size === state.totalAllNotes`。全选时调用 `selectAllIds()` → `App.GetAllNoteIDs()` 拉取所有 ID 塞入 Set。降级：后端不可用时回退当前页 |
| **搜索框逻辑** | `input` 事件 250ms 防抖自动搜索，清空时 `loadNotes()` 刷新。`keydown(Enter)` 非空立即搜索（跳过防抖），空内容刷新笔记。`renderCardGrid()` 中 footer 清理移到空状态判断之前，避免提前 return 导致 footer 残留 |
| **分段控件重构** | 主题选择（6 主题：默认/北极/粉墨/浅色/夜幕/深色）和笔记排序（更新时间/创建时间/名称）从下拉菜单/radio 改为 iOS 风格分段控件（`.segmented-control`）。分段控件动态计算按钮数及指示器位置，支持任意数量按钮 |
| **默认标签颜色** | 6 个默认标签使用不同色：待办#F59E0B、工作#3B82F6、生活#10B981、个人#8B5CF6、学习#EC4899、重要#EF4444 |
| **动画系统** | 全局 `:root` CSS 动画变量（`--anim-duration-fast: 150ms`、`--anim-ease-spring: cubic-bezier(0.16,1,0.3,1)` 等）+ 13 个 keyframes（fadeIn/fadeInUp/fadeInDown/scaleIn/slideUp/slideDown/slideInRight/shrinkOut/countUp/pulseOnce/spin/elasticScale/shake）。通用工具类 `.anim-fade-in`/`.anim-slide-up`/`.anim-scale-in`/`.anim-stagger-*`（支持 20 项交错延迟 0.02-0.4s）。`prefers-reduced-motion` 媒体查询一键降级所有动画。所有动画使用 `will-change` + `transform`/`opacity` 保证 GPU 合成层性能 |
| **搜索结果滚动条** | `.search-results` 容器通过 `scrollbar-gutter: stable` 预占滚动条空间 + `margin-right: -8px` 抵消父容器 gutter 预留，使滚动条贴靠窗口右边缘，与主内容区一致的滚动条显示/1s 淡出逻辑 |
| **编辑器双模式** | 编辑器新增纯文本/预览双模式，底部状态栏中间胶囊按钮组切换（`.editor-modes`），`data-mode="edit|preview"` 控制 textarea/预览区显隐。查看模式自动切预览，编辑默认纯文本。marked + 防抖渲染，各滚各的 |
| **全屏动画** | 全屏切换不再使用 JS inline transition，改为 CSS class 切换。`.editor-panel` 默认 `transition` 包含 `width/height/max-width/max-height/border-radius`（0.35s spring）。`.editor-panel.fullscreen` 移除 `!important`，靠 class 优先级自然覆盖。`.editor-overlay.fullscreening` 增加 backdrop-filter(10px) + 深色背景，全屏时周围更沉浸。editor-body 在过渡期间短暂淡出/淡入防内容跳跃 |
| **CSS 变量补齐** | 6 主题均已定义 `--bg-secondary`（略深于 `--bg`）+ `--text-tertiary`（略浅于 `--text-muted`），此前 7 处引用为未定义变量 |
| **确认弹窗** | `.confirm-overlay`（`position:fixed;inset:0;flex center;opacity 过渡;pointer-events`），`.visible` 切换显示。JS 用 `classList.add/remove('visible')`，不再用 `style.display` |
| **Ctrl+L 切换模式** | 编辑器打开时 `Ctrl+L` 在纯文本/预览间切换，快捷键在 `handleKeyboardNavigation` 中注册，快捷键说明页已展示 |
| **新建默认标题** | 新建笔记自动填入 `YYYY-MM-DD HH:mm ☺️`（`padStart(2,'0')` 补齐两位数），标题可手动修改后保存 |
| **置顶不更新时间** | 后端 `TogglePin` 使用 `s.db.Model(note).UpdateColumn("pinned", note.Pinned)`，GORM 的 `UpdateColumn` 不触发 `BeforeUpdate` hook，`UpdatedAt` 保持不变 |
| **右键导出 Markdown** | 右键菜单「导出」→ `ExportNoteAsMarkdown(id)` → 标题特殊符号/空白→下划线（`\ / : * ? " < > \|`）→ `runtime.SaveFileDialog` 默认文件名 `标题.md` → `os.WriteFile` 写入 `# 标题\n\n内容`，成功/失败通知 |
| **批量标签操作** | 批量工具栏 +标签/-标签 按钮。点击打开 `.batch-tag-overlay`（毛玻璃 + 居中弹入动画）。添加/移除模式统一流程：全部标签以 `.batch-tag-chip` 展示（彩色圆点，移除模式不可移除标签加 `.disabled` 灰色禁用）→ 点击切换 `.selected` 态（双环高亮边框）→ 底部确认按钮显示已选数量 → 执行后 `loadNotes()` 刷新但**不退出批量模式、不清空选择**。移除模式空态：选中笔记无任何标签时通知提示，不弹窗。后端 `BatchAddTagToNotes`/`BatchRemoveTagFromNotes` 遍历笔记 IDs 逐个操作 |
| **草稿自动保存与恢复** | 新建 `drafts` 表（ID 固定 1，字段 ID/Title/Content/CreatedAt/UpdatedAt），`DraftService` 提供 SaveDraft(upsert)/GetDraft(nil, nil)/ClearDraft。app.go 绑定 SaveDraft/GetDraft/ClearDraft 三个方法。前端 `startAutoSave()` 分支：有 editingNoteId → UpdateNote（编辑），无 → SaveDraft（新建草稿），3s 防抖，空内容不保存。`loadNotes()` 完成后延迟 1s GetDraft 检测（编辑器打开时跳过），有则 showConfirmDialog 弹窗恢复/放弃。清除时机：createNote 成功、取消按钮（明确放弃）、恢复弹窗恢复/放弃；叉号/蒙层/ESC 不清除（保留下次恢复）。快速笔记启用启动时 `loadQuickNoteSetting()` 自动查草稿填入编辑器。autoSaveIndicator 提示文字 2s 后 textContent 清空恢复布局 |
| **笔记类型（NoteType）** | Note 模型新增 `NoteType string` 字段（`gorm:"size:20;default:text"`），支持 `"text"`（纯文本）和 `"markdown"`（Markdown）两种。新建笔记默认 `"text"`。类型切换器为 header 工具栏 T/M 图标按钮（`#editorTypeToggle`，32×32，与全屏按钮同风格），查看只读模式隐藏。纯文本模式隐藏底部「编辑/预览」切换按钮（`#editorModes`），查看模式 text/`<pre>` 原始显示跳过 marked，markdown 走现有渲染。旧笔记 `note_type=""` 默认按 text 处理。创建/更新由前端传入 `noteType` 参数 |
| **置顶按钮** | 圆形图标按钮（`border-radius: 50%`），始终可见，分 4 级透明度：默认 20% → 卡片 hover 50% → 按钮 hover 100% + 放大 1.15x → 已置顶 100% + accent-lighter 背景 + 阴影。置顶操作 `togglePin()` 局部 `renderCardGrid('none')` 不再重载全部笔记 |
| **批量置顶** | 后端 `BatchPinNotes(ids, pin)` + 前端 `#batchPinBtn`。按钮文字动态：全部已置顶→「取消置顶」，否则→「置顶」。批量模式下 pin 按钮可见但 `pointer-events: none` 禁止交互 |
| **笔记本侧边栏三段式设计** | Header（`--card-bg`，56px 与 topbar 对齐，标题 + `+` 按钮）→ List（`--bg-secondary`，书签隐喻笔记本项 + badge 药丸计数器）→ Footer（`color-mix(55% bg-secondary + 45% card-bg)`，新建按钮铺满无圆角）。`color-mix()` 实现 6 主题自适应，无需硬编码各主题颜色 |
| **笔记本项书签隐喻** | 每项左侧 3px `::before` 指示条（默认透明 → hover 半透明 → active 全亮 + 发光）。hover 时 `translateX(3px)` 微弹 + accent 色背景。active 态 `inset box-shadow` 内陷感 + accent 色文字 + 加粗。名称 `text-overflow: ellipsis` 截断长名。badge 药丸形，active/hover 跟随 accent 变色 |
| **笔记本删除流程** | 自定义确认弹窗内嵌 checkbox「同时永久删除该笔记本中的 N 条笔记」。勾选 → `DeleteNotebookWithNotes()`（硬删除所有笔记 + 软删笔记本），不勾选 → `DeleteNotebook()`（笔记迁移到默认笔记本）。删除活跃笔记本后：`activeNotebookId=1` + `switchView('grid')` + `resetPagination()` 自动回到默认笔记本笔记首页 |
| **笔记本切换** | `switchNotebook(id)` 设置 `activeNotebookId` → 清空搜索 → `switchView('grid')` 强制回到网格首页 → `resetPagination()` + `loadNotes()` + `renderNotebookList()`。切换笔记本总是回到该笔记本首页 |
| **Notebook 后端模型** | `models/notebook.go` — ID/Name/CreatedAt/UpdatedAt/DeletedAt，GORM 软删除。`notebook_service.go` — 7 个公开方法 + 1 个私有 `isNameTaken()`。默认笔记本（ID=1）由 `EnsureDefaultNotebook()` 在启动时自动创建保护 |

---

> **报告结束** | 已完成项目记忆更新（2026-06-07），后续可基于此报告回答项目相关问题。
