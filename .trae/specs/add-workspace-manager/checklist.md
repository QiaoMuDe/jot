# Checklist

## 后端

- [x] `internal/services/workspace_service.go` 实现四个能力（列表/上传/下载/删除）+ 类型定义，遵循 services 层代码规范
- [x] 上传重名自动改名生效（`xxx (1).ext`，文件与目录均适用），下载到桌面重名同样自动改名
- [x] 目录上传/下载递归复制；目录删除需显式 recursive=true；工作区根目录拒绝删除
- [x] 沙箱校验生效：`../` 逃逸与 symlink/junction 逃逸被拒绝且返回中文错误，无文件副作用
- [x] 单项失败不中断批次（逐项返回结果）
- [x] `workspace_service_test.go` 覆盖上述场景且全绿
- [x] `go build ./...` + `go vet ./...` 无告警
- [x] `app.go` 5 个绑定方法齐全，对话框取消不报错，logger 规范

## 前端

- [x] 顶栏工作区按钮（文件夹图标）位于折叠侧栏与新建会话之间，点击打开 Modal
- [x] Modal 中部文件树按目录分组（目录在前、名称升序），目录可展开/折叠
- [x] 上传/下载/删除/刷新全部操作按钮统一位于**顶部操作栏**（无底部操作栏）；下载/删除未选中时禁用（降透明度 + hover 提示）
- [x] checkbox 多选 + 「已选 N 项」实时更新
- [x] 上传文件/上传目录成功复制进工作区，重名自动改名，上传中按钮 loading 禁用，完成后刷新列表 + 通知
- [x] 批量下载到桌面同名相对路径；删除带 danger 二次确认（含目录提示递归），取消不执行
- [x] 空工作区展示空状态引导上传，不渲染空列表
- [x] Modal 缩放+淡入开合 150–300ms，尊重 prefers-reduced-motion；关闭按钮/遮罩/ESC（收口 handleKeyboardNavigation）均可关闭且无残留监听器
- [x] 竞态防护：列表加载使用代际序号 `workspaceLoadSeq`，异步/定时器统一清理
- [x] SVG 图标统一 lucide 风格（stroke-width 与现有按钮一致），无 emoji；删除按钮 danger 色与下载分离
- [x] 样式沿用主题 CSS 变量（11 套主题兼容），文件树行 hover/选中态即时响应

## 构建与文档

- [x] `npm run build` 通过；`wails generate module` 后 wailsjs 绑定无缺漏
- [x] AGENTS.md 临时记忆已更新（三步法：删最旧→顺移→末尾追加）
