# Checklist

## 面板结构
- [x] 设置页侧边栏出现「MCP 服务器」导航项，点击可切换至对应面板
- [x] 面板含「添加服务器」按钮、列表容器与空态容器；表单对话框骨架存在且层级低于确认对话框

## 列表与数据
- [x] 打开面板/初始化时调用 `GetMCPServers` 渲染列表，条目含名称、传输徽标、命令/URL 摘要、启用开关、编辑/删除按钮
- [x] 空列表显示空态提示

## 新增/编辑表单
- [x] 新增与编辑共用一个表单，编辑时正确预填充记录
- [x] 传输方式切换正确显隐 stdio 组（命令/参数/环境变量）与 sse/http 组（URL）
- [x] 前端校验生效（名称必填、stdio 命令必填、sse/http URL 必填、环境变量需 `KEY=VALUE`）
- [x] 参数文本域转数组、环境变量文本域转对象后正确提交 `SaveMCPServer`
- [x] 保存成功关闭表单并刷新列表；失败保留表单并展示后端中文错误

## 行内操作
- [x] 启用开关切换即持久化，失败恢复原状态并提示
- [x] 删除需经 `showConfirmDialog` 确认，确认后调用 `DeleteMCPServer` 并刷新列表

## 样式与构建
- [x] 列表/徽标/空态/表单样式与既有设置页视觉一致，主题变量自适应，`prefers-reduced-motion` 降级
- [x] `wailsjs/go/main/App.js` 已生成 `GetMCPServers`/`SaveMCPServer`/`DeleteMCPServer` 绑定
- [x] `npm run build` 与 `wails build` 均成功
