<div align="center">

# Jot — 卡片式笔记桌面应用

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat-square&logo=go) ![Wails](https://img.shields.io/badge/Wails-v2.9.2-DF367C?style=flat-square&logo=wails) ![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite) ![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

**Jot** 是一款基于 [Wails v2](https://wails.io/) 构建的轻量级卡片式笔记桌面应用，采用 **Go + 原生 Web 技术栈**（无 Vue/React），界面清爽、交互流畅、数据本地存储。

[✨ 特性](#-核心特性) · [🚀 安装](#-安装指南) · [🛠️ 开发](#-开发) · [🔗 仓库](https://gitee.com/MM-Q/jot.git)

</div>

---

## ✨ 核心特性

### 📝 笔记管理

- **卡片式笔记网格** — 笔记本 → 笔记卡片 → 编辑器的三步交互范式，直观的文件夹-文件-编辑结构
- **CodeMirror 6 编辑器** — 专业 Markdown 编辑体验，支持行号、撤销重做、查找替换、Tab 缩进、语法高亮
- **14 套系统主题 + 11 套代码高亮主题** — 全局 CSS 变量联动（`--bg`/`--accent`/`--border` 等），所有组件自动适配（2026-07 完成配色全面重构）
- **笔记本（目录）系统** — 创建、编辑、删除笔记本，按笔记本筛选笔记
- **标签系统** — 自定义标签（名称 + 颜色），按标签筛选笔记，支持无限滚动标签选择器
- **笔记排序与分页** — 按更新时间/创建时间/名称排序，每页 20-100 条可配置
- **搜索弹窗** — 200ms 防抖 + 笔记本/日期范围/排序/标签筛选器，支持全文搜索

### 🤖 AI 助手

- **自研 AI 对话引擎** — 基于自研 `aicli` 适配层（底层 `go-openai` + Ollama 原生 API），统一流式接口
- **双 Provider 支持** — OpenAI 兼容（DeepSeek、通义千问等）或 Ollama 本地模型
- **API 配置预设** — 多 API 配置管理，一键切换预设，支持配置预设导入导出
- **流式输出** — 逐块推送 + Markdown 渲染 + 代码高亮（hljs）
- **深度思考** — 支持 `reasoning_content` 字段，思维链可折叠展示
- **多来源联网搜索** — 支持 Tavily 通用搜索、知乎搜索、全网搜索三个独立来源，可同时开启
- **卡片召回** — 基于 2-gram 分词的相似笔记搜索，AI 回复前自动召回相关笔记并注入上下文
- **引用笔记与文件上传** — 手动选择笔记引用到对话中，支持拖拽上传文件
- **角色扮演笔记** — 指定笔记作为 AI 的角色/身份设定
- **更多技能** — 10 项互斥一键技能（翻译、编程、写作、解题答疑、需求规格、文本润色、内容摘要、文案生成、工作总结、提示词生成）
- **优化表达** — 输入框内嵌一键优化按钮，支持还原原文
- **多会话管理** — 创建/切换/删除/重命名会话，侧栏折叠，Context Size 实时显示
- **消息操作** — 编辑、删除、重新发送、重新生成（再生），基于消息 ID 的精确操作
- **消息懒加载** — 分页加载历史消息，长会话流畅不卡顿
- **AI 消息右键菜单** — 复制、保存为笔记、删除

### 🗂️ 数据管理

- **笔记本管理** — 创建、编辑、删除笔记本，按笔记本筛选笔记
- **批量管理** — FAB 入口进入选择模式，批量置顶/删除笔记
- **待办清单** — 完整的待办 CRUD，带筛选和动画交互
- **数据统计面板** — 笔记/标签/回收站/笔记本/AI 会话/AI 消息/数据库大小 7 项统计
- **备份/还原** — 一键导入/导出完整数据（`.jbackup` 格式），支持导出为 `.db` 文件
- **数据库瘦身 VACUUM** — 释放空间
- **回收站** — 软删除机制，支持还原/永久删除/全部清空，混合显示笔记和笔记本条目

### 🎨 设计系统

- **纯手写 CSS** — 无 UI 框架依赖，极致轻量
- **CSS 变量主题系统** — 14 套主题，统一设计 Tokens（圆角/阴影/间距/语义色）
- **过渡动画** — 骨架屏 shimmer、stagger 延迟、hover 分层反馈、弹性缓动
- **统一的滚动条** — 6px 细条，联动全部 14 主题
- **通知系统** — NotificationManager 单例，4 种类型 + undo 撤销

### 🔧 其他

- **字体设置** — 字体族 + 大小，联动 CSS 变量
- **锁屏密码** — SHA-256 哈希存储，毛玻璃锁屏遮罩
- **Mermaid 图表** — Markdown 代码块按需渲染，源码/视图切换
- **更多菜单** — 毛玻璃精工卡设计，双层阴影 + 弹性入场动画
- **Markdown 语法手册** — 10 张语法卡片，双栏源码/预览
- **快捷键说明页** — 可滚动列表，一键呼出
- **响应式布局** — 全屏/小窗口适配

---

## 🏗️ 技术栈

| 层级 | 技术 | 版本 | 用途 |
|------|------|------|------|
| **桌面框架** | Wails v2 | v2.9.2 | 桌面窗口 + Go↔JS Bridge |
| **后端语言** | Go | go1.22+ | 后端业务逻辑 |
| **数据库** | SQLite | — | 本地数据存储 |
| **数据库驱动** | glebarez/sqlite | v1.11 | 纯 Go SQLite 驱动（无 CGO） |
| **ORM** | GORM | v1.25 | 对象关系映射 |
| **编辑器** | CodeMirror 6 | @codemirror/view v6.26 | 笔记 Markdown 编辑器 |
| **Markdown 渲染** | marked | v12.0 | Markdown → HTML |
| **代码高亮** | highlight.js | v11.10 | 代码块语法高亮 |
| **图表渲染** | Mermaid | v11.4 | Markdown 代码块图表按需渲染 |
| **AI 适配层** | 自研 aicli | — | 底层 `go-openai` + `ollama/ollama/api` |
| **前端构建** | Vite | v3.2.11 | 前端打包 |
| **前端技术** | 原生 HTML/CSS/JS | — | UI 渲染（无框架） |
| **本地存储** | localStorage | — | UI 状态持久化 |

---

## 🚀 安装指南

### 前置依赖

- **Go** ≥ 1.22
- **Wails CLI** v2.9+（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
- **Node.js** ≥ 16

### 从源码构建

```bash
git clone https://gitee.com/MM-Q/jot.git
cd jot
wails build
```

构建产物：`./build/bin/jot.exe`

### 下载发布版

前往 [Releases](https://gitee.com/MM-Q/jot/releases) 页面下载最新发布版安装包。

---

## 🛠️ 开发

```bash
# 开发模式（前端热重载）
wails dev

# 代码格式化 + 静态分析
golangci-lint fmt ./... && golangci-lint run ./...
```

---

## ❓ FAQ

| 问题 | 回答 |
|------|------|
| **数据存在哪里？** | 所有数据存储在本地 SQLite 文件中，默认位于用户数据目录下的 `.jot/data/jot.db` |
| **AI 对话需要什么？** | 需要 API Key，支持 OpenAI 兼容服务商（DeepSeek、通义千问等）或本地 Ollama 模型 |
| **联网搜索怎么用？** | 在设置中配置 API Key，对话时开启"联网搜索"开关即可，支持 Tavily/知乎/全网搜索三个来源 |
| **可以导出数据吗？** | 可以，在数据管理页面可导出为 `.jbackup`（完整备份）或 `.db`（SQLite 数据库文件）|

---

## 🤝 贡献指南

1. **Fork** 本仓库
2. **创建特性分支**：`git checkout -b feat/amazing-feature`
3. **遵循现有代码风格**，golangci-lint 零警告
4. **提交并发起 Pull Request**

---

## 📄 许可证

本项目采用 **MIT License** 开源。

---

## 📬 相关链接

| 资源 | 链接 |
|------|------|
| 🪧 项目仓库 | [https://gitee.com/MM-Q/jot.git](https://gitee.com/MM-Q/jot.git) |
| 🐛 提交 Issue | [https://gitee.com/MM-Q/jot/issues](https://gitee.com/MM-Q/jot/issues) |
| 🛠️ Wails 框架 | [https://wails.io](https://wails.io) |
| 🗃️ GORM ORM | [https://gorm.io](https://gorm.io) |
| 🔌 Tavily Search | [https://tavily.com](https://tavily.com) |

---

<div align="center">

**如果 Jot 对你有帮助，欢迎 ⭐ Star 支持！**

[![Gitee Stars](https://img.shields.io/badge/dynamic/json?label=Stars&query=$.stargazers_count&url=https://gitee.com/api/v5/repos/MM-Q/jot&style=flat-square&color=yellow)](https://gitee.com/MM-Q/jot)

</div>
