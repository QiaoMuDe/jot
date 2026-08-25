# Plan: 更新 AGENTS.md 记忆点

## Summary
将本次 MCP 服务器分享与导入功能的变更记录写入 AGENTS.md 的"关键记忆点"（Section 九），同时按维护规范滚动更新记忆点编号。

## Current State Analysis

### Section 九（关键记忆点）
- 当前有 10 条记忆点（编号 1-10），按时间升序排列
- 最旧的是 **记忆点 1：笔记副本创建 + 前端 ESLint 全量清零 + AI 输入长度限制**
- 维护规范：删除最旧→顺移编号→末尾追加新条目

### 本次变更涉及的新文件/模块
- `internal/services/mcp_import.go`（**新增**）：JSON 解析 + 三格式容错 + 字段校验 + 批量入库
- `internal/models/mcp_server.go`（修改）：新增 `ImportMCPServerItem` 结构体
- `app.go`（修改）：新增 `ImportMCPServers` + `ParseMCPServersImport` binding
- `frontend/src/main.js`（修改）：导入对话框 + 分享按钮 + `warmupMCPServers(silent)` 参数
- `frontend/src/css/components/settings-panel.css`（修改）：头部三按钮 + 分享行按钮样式
- `frontend/index.html`（修改）：头部三按钮组 + 分享行按钮 + 导入对话框 DOM

## Proposed Changes

### 1. 更新 Section 九（关键记忆点）

按维护规范执行三步：

**第一步**：删除最旧的"记忆点 1：笔记副本创建 + 前端 ESLint 全量清零 + AI 输入长度限制"

**第二步**：将剩余条目顺移重新编号
- 原 2→1（笔记首页加载优化）
- 原 3→2（编辑器切换闪烁修复）
- 原 4→3（回收站全部清空/恢复 动画死锁）
- 原 5→4（AI 消息 Meta Chip 显示）
- 原 6→5（笔记搜索打分排序）
- 原 7→6（MCP 客户端迁移到官方 go-sdk）
- 原 8→7（MCP 服务器工具精细化控制）
- 原 9→8（AI 会话持久化对话摘要）
- 原 10→9（AI 助手消息区/输入区重构）

**第三步**：在末尾追加新条目作为 **记忆点 10：MCP 服务器分享与导入（三格式容错 + 两阶段校验 + 后端解析日志 + 按钮 UI 统一）**

新记忆点内容概要：
- 后端新增 `ParseMCPServersImport`（仅校验不入库）+ `ImportMCPServers`（解析+校验+入库），JSON 解析/错误日志全部在后端，前端零解析逻辑
- 三格式容错：裸数组 / `{servers:[...]}` / 单个对象
- 字段校验含空白/KEY 特殊字符/transport 合法性（与 service.Save 一致）
- 前端两阶段流程：校验→关对话框→入库→通知（失败保留对话框+textarea）
- `warmupMCPServers` 加 `options.silent` 参数，导入成功后静默调用避免双通知
- 头部按钮统一 2 字文案（分享/导入/添加），同一套 `mcp-server-accent-btn` 样式
- 分享按钮每行独立 + 头部分享全部

### 2. 不需更新的其他 Section
- Section 二（核心功能模块表）：MCP 服务器管理在 memory points 中已有记录，不需在模块表中新增行
- Section 十（待优化点）：本次变更属于功能新增而非待优化项归档

## Verification Steps
- [ ] 記憶点 1（最旧）已删除
- [ ] 記憶点 2-10 已顺移重新编号为 1-9
- [ ] 新记忆点 10 已追加在末尾
- [ ] 编号连续（1→10），无跳号/重复
- [ ] 总条目数 = 10（不超出上限）
