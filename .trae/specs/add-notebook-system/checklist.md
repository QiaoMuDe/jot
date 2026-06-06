# Checklist - 笔记本层级分组系统

## 后端数据模型
- [x] Notebook 模型定义正确（id/name/sort_order/timestamps）
- [x] Note 模型新增 `NotebookID uint` 字段（default:0, index）
- [x] Notebook 模型加入 AutoMigrate

## 默认笔记本 + 存量迁移
- [x] 首次启动（notebooks 表空）自动创建「默认笔记本」（id=1）
- [x] notebook_id=0 的存量笔记自动迁移到默认笔记本
- [x] 后续启动跳过创建

## 后端服务层
- [x] NotebookService.Create() 正常工作
- [x] NotebookService.Update()（重命名）正常工作
- [x] NotebookService.Delete() 删除笔记本 + 笔记迁入默认笔记本
- [x] NotebookService.GetAll() 返回所有未删除笔记本
- [x] NotebookService.GetAllNotesCount() 返回各笔记本笔记数

## NoteService 扩展
- [x] GetByNotebook() 按笔记本分页查询
- [x] GetAllNoteIDsByNotebook() 返回笔记本内全部 ID

## App 绑定层
- [x] 6 个笔记本绑定方法正确暴露（Create/Rename/Delete/GetAll/GetCounts）
- [x] GetNotes/Search 支持 notebookID 筛选参数
- [x] CreateNote 接收并存储 notebookID

## 前端侧栏 UI
- [x] 侧栏显示全部笔记本列表，默认笔记本排在最前面
- [x] 每个笔记本项显示名称 + 笔记数 badge
- [x] 激活的笔记本有高亮视觉指示（accent 色背景）
- [x] 侧栏底部有「✚ 新建笔记本」按钮
- [x] 侧栏右上角有折叠按钮
- [x] 侧栏展开宽度 200px，折叠宽度 44px
- [x] 折叠/展开时主内容区自适应
- [x] 折叠状态通过 `localStorage` 持久化
- [x] 侧栏无「全部笔记」入口

## 前端内容隔离筛选
- [x] 应用启动默认选中「默认笔记本」，只显示其笔记
- [x] 切换笔记本时仅加载该 notebook_id 的笔记
- [x] 切换笔记本时清除搜索内容和页码
- [x] 搜索时始终在激活笔记本范围内搜索
- [x] 没有「查看全部笔记」的功能
- [x] ESC 回到首页不切换笔记本

## 前端笔记本 CRUD
- [x] 新建笔记本弹窗 → 创建成功 → 侧栏刷新 → 自动切换到该笔记本
- [x] 右键笔记本菜单（重命名/删除）
- [x] 重命名内联编辑 → 回车/失焦保存 → 侧栏更新
- [x] 删除确认弹窗 → 笔记迁入默认笔记本 → 侧栏刷新
- [x] 删除当前激活笔记本后自动切到默认笔记本
- [x] 「默认笔记本」右键菜单无「删除」选项

## 新建笔记归属
- [x] 当前在哪个笔记本下，新建笔记就自动归哪个笔记本
- [x] 创建成功后笔记只出现在该笔记本下

## 删除笔记本
- [x] 删除笔记本时其下笔记不丢失，迁入默认笔记本

## 验证
- [ ] `golangci-lint run ./...` 0 issues（当前环境无 golangci-lint，已验证 go build ./... 和 go vet ./... 通过）
- [x] Seed 工具 5 个笔记本 + 笔记正确分配
- [x] 启动后默认处于「默认笔记本」，仅显示该笔记本笔记
