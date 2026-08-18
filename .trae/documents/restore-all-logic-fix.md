# 全部恢复（RestoreAll）逻辑修复计划

## 1. 概要

修复回收站"全部恢复"功能的三种场景处理，确保笔记能正确回到原笔记本（或默认笔记本）。

**当前 bug**：回收站笔记所属笔记本如果**也在回收站**（软删除），`RestoreAll` 会把笔记直接迁到默认笔记本（id=1），不会恢复那个笔记本。这跟用户预期不一致——用户希望"笔记本和笔记一起回来"。

***

## 2. 现状分析

### 涉及文件

| 文件                                                                                                                        | 关键位置                                             |
| ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| [internal/services/note\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L415-L434)         | `NoteService.RestoreAll` —— 主要修复点                |
| [internal/services/notebook\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/notebook_service.go#L383-L394) | `NotebookService.RestoreAllFromTrash` —— 已正确，无需改 |
| [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L732-L741)                                                                | `App.RestoreAllNotes` —— 透传，无需改                  |
| [frontend/src/js/trash-page.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/trash-page.js#L175-L211)                  | `restoreAllNotes` —— 顺序无需改（backend 自包含）          |

### 当前 `RestoreAll` 行为（[note\_service.go:415-434](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L415-L434)）

```go
// Step 1: 笔记本"不存在或已软删除" → 笔记迁到默认
UPDATE notes SET notebook_id = 1
WHERE deleted_at IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM notebooks
      WHERE notebooks.id = notes.notebook_id
        AND notebooks.deleted_at IS NULL  -- 只排除软删除
  )

// Step 2: 恢复所有笔记
UPDATE notes SET deleted_at = NULL WHERE deleted_at IS NOT NULL
```

**问题**：把"笔记本已软删除（在回收站）"和"笔记本已被永久删除"两种场景**混为一谈**，都迁到默认笔记本。这跟用户期望不符。

### 关键不变量（验证后确认）

* **默认笔记本（id=1）永远不会被软删除**——`NotebookService.Delete`（[line 96-99](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/notebook_service.go#L96-L99)）、`DeleteWithNotes`（[line 145-148](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/notebook_service.go#L145-L148)）、`RestoreFromTrash`（[line 334-337](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/notebook_service.go#L334-L337)）都有 `if id == 1` 守卫

* 因此 `notebook_id = 1` 永远指向"活"的默认笔记本，**不需要把 id=1 纳入恢复范围**

* 前端调用顺序是 `RestoreAllNotes` → `RestoreAllTrashNotebooks`（[trash-page.js:194-201](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/trash-page.js#L194-L201)），但修复后 `RestoreAllNotes` 应**自包含**地处理"父笔记本在回收站"的情况，不再依赖调用顺序

### 同问题的函数（待评估是否一并修复）

* [note\_service.go:508-523](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L508-L523) `BatchRestore` —— 用相同 SQL 模式

* [note\_service.go:526-557](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L526-L557) `Restore` (单条) —— `notebook` 软删除时也走"迁默认"分支

***

## 3. 目标行为（三场景）

| 场景              | 笔记 notebook\_id 指向                | 处理                               |
| --------------- | --------------------------------- | -------------------------------- |
| ① 父笔记本在回收站（软删除） | 在 trash notebooks 表中能找到           | **先恢复笔记本** → 笔记恢复到原 notebook\_id |
| ② 父笔记本存活        | notebooks 表中 `deleted_at IS NULL` | 直接恢复笔记到原 notebook\_id            |
| ③ 父笔记本永久删除/不存在  | notebooks 表中无该 id 记录              | 笔记迁到默认（id=1）后恢复                  |

***

## 4. 实施步骤

### 步骤 1：重写 `NoteService.RestoreAll`（核心改动）

**文件**：[internal/services/note\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L415-L434)

**改动**：把当前函数（19 行）替换为三阶段处理（约 30 行）：

```go
// RestoreAll 批量恢复回收站中所有已软删除的笔记。
// 按笔记所属笔记本的状态分三种场景处理:
//  1) 父笔记本在回收站（软删除）→ 先恢复该笔记本, 笔记回到原 notebook_id
//  2) 父笔记本存活 → 笔记直接回到原 notebook_id
//  3) 父笔记本已被永久删除/不存在 → 笔记迁到默认笔记本 (id=1) 后恢复
//
// 默认笔记本 (id=1) 因 Delete/DeleteWithNotes/RestoreFromTrash 都有 id==1 守卫,
// 永远不会被软删除, 因此不在恢复范围。
func (s *NoteService) RestoreAll() error {
    // Stage 1: 恢复回收站笔记引用的、且本身在回收站的非默认笔记本
    if err := s.db.Unscoped().Exec(`
        UPDATE notebooks
        SET deleted_at = NULL
        WHERE deleted_at IS NOT NULL
          AND id IN (
              SELECT DISTINCT notebook_id
              FROM notes
              WHERE deleted_at IS NOT NULL
                AND notebook_id != 0
                AND notebook_id != 1
          )
    `).Error; err != nil {
        s.logger.Errorw("NoteService.RestoreAll 失败(恢复关联笔记本)", fastlog.Error(err))
        return err
    }

    // Stage 2: 父笔记本已永久删除或不存在 → 迁到默认笔记本
    if err := s.db.Unscoped().Model(&models.Note{}).
        Where("deleted_at IS NOT NULL AND notebook_id != 1").
        Where("notebook_id NOT IN (SELECT id FROM notebooks)").
        Update("notebook_id", 1).Error; err != nil {
        s.logger.Errorw("NoteService.RestoreAll 失败(迁默认笔记本)", fastlog.Error(err))
        return err
    }

    // Stage 3: 取消所有回收站笔记的 deleted_at
    if err := s.db.Unscoped().Model(&models.Note{}).
        Where("deleted_at IS NOT NULL").
        Update("deleted_at", nil).Error; err != nil {
        s.logger.Errorw("NoteService.RestoreAll 失败(恢复笔记)", fastlog.Error(err))
        return err
    }
    return nil
}
```

**为什么这么写**：

* **Stage 1 用 raw SQL**：`UPDATE ... WHERE id IN (SELECT ... FROM notes ...)` 走 GORM 的链式 API 比较绕，raw SQL 清晰且 SQLite 优化器会处理子查询。GORM 也支持子查询，但代码可读性下降

* **Stage 1 排除** **`notebook_id IN (0, 1)`**：`notebook_id=0` 是历史脏数据（项目里没清理过），`notebook_id=1` 永远不在 trash，跳过避免无意义的 IN 列表

* **Stage 2 的** **`NOT IN (SELECT id FROM notebooks)`**：用子查询判断"笔记本是否存在"，比 `NOT EXISTS` 更直观，SQLite/GORM 都会优化

* **不引入 transaction**：当前 DB 配置 `SkipDefaultTransaction: true`，每个 UPDATE 独立提交；保持与项目现有风格一致。如果未来要保证强原子性，可以再包 `s.db.Transaction(...)`

### 步骤 2（可选，建议）：一并修复 `BatchRestore` 和 `Restore`

**原因**：用户没明确要求，但这两个函数有**完全相同的 bug**——单条恢复时如果父笔记本在回收站，笔记会被错误地迁到默认。

**改动建议**：

* [note\_service.go:508-523](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L508-L523) `BatchRestore`：把 `Where("NOT EXISTS ...")` 拆成两阶段

  * 先对 `id IN ?` 的笔记，查询其 notebook\_id，对在 trash 的恢复

  * 再对笔记本不存在的迁默认

  * 最后恢复 deleted\_at

* [note\_service.go:526-557](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L526-L557) `Restore` (单条)：把 `Where("deleted_at IS NULL").First(&notebook, note.NotebookID)` 的分支处理改为：

  * 先 `Unscoped().First(&notebook, note.NotebookID)` 看是否存在

  * 存在且软删除 → 恢复笔记本

  * 不存在 → 迁默认

**决策点**：是否在本次一起改？请用户在 Phase 4 之后回复。

### 步骤 3：前端无需改动

* 现有的 `RestoreAllNotes` → `RestoreAllTrashNotebooks` 顺序可保留

* 修复后 `RestoreAllNotes` 自包含地处理"父笔记本在回收站"场景

* `RestoreAllTrashNotebooks` 仍然有用：恢复**没有笔记引用**的空笔记本（如手动软删除但里面没笔记）

***

## 5. 假设与决策

| 假设                         | 决策                        | 理由                               |
| -------------------------- | ------------------------- | -------------------------------- |
| 默认笔记本（id=1）不会被软删除          | 排除 `notebook_id=1` 在恢复范围外 | 不变量已通过代码验证                       |
| 原子性非必需                     | 不引入 `db.Transaction`      | 保持项目现有风格（SkipDefaultTransaction） |
| BatchRestore/Restore 是否一并修 | **待用户决定**                 | 用户只点名 RestoreAll，但同 bug 也存在      |
| 前端调用顺序                     | 不调整                       | 修复后 `RestoreAllNotes` 自包含，不依赖顺序  |
| 多条恢复 vs 单条恢复逻辑一致           | 待用户决定                     | 见步骤 2                            |

***

## 6. 验证步骤

### 6.1 数据库层验证（自动化测试 / 手动 SQL）

构造三种场景的数据，调用 `NoteService.RestoreAll()` 后检查结果：

```sql
-- 前置：准备 3 个笔记本 (id=2, 3, 4) 和 3 条笔记
-- 软删除笔记本 id=2，笔记 id=10 也软删除，note.notebook_id=2
-- 笔记本 id=3 保持存活，笔记 id=11 软删除，note.notebook_id=3
-- 硬删除笔记本 id=4，笔记 id=12 软删除，note.notebook_id=4

-- 调用 RestoreAll 后预期:
-- - notebook id=2: deleted_at = NULL
-- - note id=10: deleted_at = NULL, notebook_id = 2
-- - notebook id=3: deleted_at IS NULL (没变)
-- - note id=11: deleted_at = NULL, notebook_id = 3
-- - notebook id=4: 不存在
-- - note id=12: deleted_at = NULL, notebook_id = 1
```

### 6.2 Wails 集成验证

* 删除几个笔记本及其下的笔记 → 进回收站确认都能看到

* 点"全部恢复" → 控制台日志应该出现 `RestoreAllNotes 成功` 和 `RestoreAllTrashNotebooks 成功`

* 返回笔记页 → 笔记应回到原笔记本

* 硬删除某个笔记本（先放进回收站，再从回收站永久删除），然后软删除其下的笔记 → 点"全部恢复" → 笔记应到默认笔记本

### 6.3 边界情况

* 回收站为空 → 调用应直接成功，0 行受影响

* 只有一个默认笔记本（id=1）的场景 → 笔记 notebook\_id=1 的不应被错误处理

* 多个笔记引用同一个已软删除笔记本 → 只恢复一次该笔记本

***

## 7. 不改的内容

* **前端**：调用顺序、动画、文案都不变

* **NotebookService.RestoreAllFromTrash**：已正确，无需改

* **App.RestoreAllNotes / RestoreAllTrashNotebooks**：透传 layer，无需改

* **Wails bindings**：JS 函数签名不变

* **数据库 schema**：无表结构变更

* **数据模型**：无新模型

***

## 8. 风险评估

| 风险                                | 可能性 | 影响 | 缓解                             |
| --------------------------------- | --- | -- | ------------------------------ |
| Stage 1 raw SQL 在某些 GORM 版本有兼容问题  | 低   | 致命 | 项目用 GORM v2，Exec 原生支持          |
| 软删除笔记本被恢复后, 其他外部引用失效              | 极低  | 低  | 笔记本关系都在前端 JS 层缓存, 刷新即可         |
| 与 `RestoreAllTrashNotebooks` 重复执行 | 低   | 无害 | 第二调用是 idempotent（已恢复的再设置 NULL） |
| BatchRestore/Restore 不一致          | 中   | 中  | 在步骤 2 决策点跟用户确认                 |

