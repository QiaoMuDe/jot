# Tasks

- [x] Task 1: Go 后端 — 新增 `CleanupOrphanImages()` API
  - 在 `app.go` 中新增方法，接收方为 `*App`，返回 `int`（删除数量）
  - 实现逻辑：
    1. 使用 `os.UserHomeDir()` 定位 `~/.jot/images/` 目录
    2. `os.ReadDir(imageDir)` 列出所有文件
    3. 从数据库查询所有笔记（含软删除）的 content：`a.db.Model(&models.Note{}).Unscoped().Pluck("content", &contents)`
    4. 构建引用集合：遍历 contents，用 `strings.Contains(content, "/images/"+filename)` 匹配
    5. 遍历目录文件，不在引用集合中的 → `os.Remove(filepath.Join(imageDir, filename))`，计数
    6. 返回删除数量
  - 目录不存在或为空时直接返回 0，不报错
  - 添加注释说明清理逻辑

- [x] Task 2: 前端 — 新增"清理未引用图片"按钮
  - 在 `frontend/index.html` 的"数据清理"分组（`clearCompletedTodosBtn` 之后）新增按钮 DOM：
    - `id="cleanupOrphanImagesBtn"`
    - 图标使用垃圾桶/图片相关 SVG
    - label: "清理未引用图片"
    - desc: "删除笔记中不再使用的图片文件"
  - 注意按钮之间要有 `dar-divider`

- [x] Task 3: 前端 — 绑定事件与实现清理函数
  - 在 `frontend/src/main.js` 中：
    - 在 `els` 对象中注册 `cleanupOrphanImagesBtn` 元素引用
    - 绑定点击事件调用 `cleanupOrphanImages()`
  - 在 `frontend/src/js/data-management.js` 中新增 `cleanupOrphanImages()` 函数：
    - 调用 `showConfirmDialog` 确认（提示："确定要清理未引用的图片吗？这将删除笔记中不再使用的图片文件。"）
    - 确认后调用 `window.go.main.App.CleanupOrphanImages()`
    - 根据返回值显示 Toast（如"已清理 2 张未引用图片" 或 "没有需要清理的未引用图片"）
    - 捕获异常并显示错误 Toast

- [x] Task 4: 存储优化按钮联动
  - 在 `data-management.js` 的 `vacuumDatabase()` 函数中，VACUUM 成功后自动执行 `CleanupOrphanImages()`
  - 合并结果显示，如原 Toast "数据库已瘦身" → 改为 "数据库已瘦身，已清理 2 张未引用图片"

# Task Dependencies

- Task 1 是 Task 2/3/4 的前置依赖
- Task 2 和 Task 3 可并行（一个负责 DOM，一个负责 JS 逻辑）
- Task 4 依赖 Task 1（API）和 Task 3（cleanupOrphanImages 函数）
