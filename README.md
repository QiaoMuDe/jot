<div align="center">

# Jot — 卡片式笔记桌面应用

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat-square&logo=go) ![Wails](https://img.shields.io/badge/Wails-v2.9.2-DF367C?style=flat-square&logo=wails) ![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite) ![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

**Jot** 是一款基于 [Wails v2](https://wails.io/) 构建的轻量级卡片式笔记桌面应用，采用 **Go + 原生 Web 技术栈**（无 Vue/React），界面清爽、交互流畅、数据本地存储。

[✨ 特性](#-核心特性) · [🚀 安装](#-安装指南) · [🛠️ 开发](#-开发) · [⌨️ 快捷键](#-快捷键) · [🔗 仓库](https://gitee.com/MM-Q/jot.git)

</div>

---

## ✨ 核心特性

### 📝 笔记管理

- **卡片式笔记网格** — 笔记本 → 笔记卡片 → 编辑器的三步交互范式，直观的文件夹-文件-编辑结构
- **CodeMirror 6 编辑器** — 专业 Markdown 编辑体验，支持行号、撤销重做、查找替换、Tab 缩进、语法高亮
- **12 套系统主题 + 11 套代码高亮主题** — 全局 CSS 变量联动（`--bg`/`--accent`/`--border` 等），所有组件自动适配
- **笔记本（目录）系统** — 创建、编辑、删除笔记本，按笔记本筛选笔记
- **标签系统** — 自定义标签（名称 + 颜色），按标签筛选笔记，支持无限滚动标签选择器
- **笔记排序与分页** — 按更新时间/创建时间/名称排序，每页 20-100 条可配置
- **搜索弹窗** — 200ms 防抖 + 笔记本/日期范围/排序/标签筛选器，支持全文搜索

### 🤖 AI 助手

- **自研 AI 对话引擎** — 基于自研 `aicli` 适配层（底层 `go-openai` + Ollama 原生 API），统一流式接口
- **双 Provider 支持** — OpenAI 兼容（DeepSeek、通义千问等）或 Ollama 本地模型
- **API 配置预设** — 多 API 配置管理，一键切换预设
- **流式输出** — 逐块推送 + Markdown 渲染 + 代码高亮（hljs）
- **深度思考** — 支持 `reasoning_content` 字段，思维链可折叠展示
- **联网搜索** — Tavily API 集成，搜索结果静默注入 AI 上下文，设置页可配置返回条数
- **卡片召回** — 基于 2-gram 分词的相似笔记搜索，AI 回复前自动召回相关笔记并注入上下文
- **引用笔记** — 手动选择笔记引用到对话中，支持标签筛选 + 无限滚动
- **多会话管理** — 创建/切换/删除/重命名会话，侧栏折叠，Token 数持久化
- **上下文大小显示** — 实时展示对话 Token 用量
- **更多技能** — 翻译、文本润色、总结等 5 项一键技能，互斥选择
- **优化表达** — 输入框内嵌一键优化按钮，支持还原原文
- **AI 消息右键菜单** — 复制、保存为笔记、删除

### 🗂️ 数据管理

- **数据统计面板** — 笔记/标签/回收站/笔记本/AI 会话/AI 消息/数据库大小 7 项统计
- **备份/还原** — 一键导出导入完整数据（`.jbackup` 格式）
- **数据库瘦身 VACUUM** — 释放空间
- **回收站** — 软删除机制，支持还原/永久删除，混合显示笔记和笔记本条目

### 🎨 设计系统

- **纯手写 CSS** — 无 UI 框架依赖，极致轻量
- **CSS 变量主题系统** — 12 套主题，统一设计 Tokens（圆角/阴影/间距/语义色）
- **过渡动画** — 骨架屏 shimmer、stagger 延迟、hover 分层反馈、弹性缓动
- **统一的滚动条** — 6px 细条，联动全部 12 主题
- **通知系统** — NotificationManager 单例，4 种类型 + undo 撤销

### 🔧 其他

- **字体设置** — 字体族 + 大小，联动 CSS 变量
- **Markdown 语法手册** — 10 张语法卡片，双栏源码/预览
- **快捷键说明页** — 可滚动列表，一键呼出
- **响应式布局** — 全屏/小窗口适配

---

## ⌨️ 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+1` | 笔记（首页） |
| `Ctrl+2` | AI 助手 |
| `Ctrl+3` | 数据管理 |
| `Ctrl+4` | 设置 |
| `Ctrl+5` | 搜索弹窗 |
| `Ctrl+6` | Markdown 语法手册 |
| `Ctrl+7` | 快捷键说明页 |
| `Ctrl+8` | 回收站 |
| `Ctrl+9` | 切换侧栏折叠 |
| `Ctrl+N` | 新建笔记 |
| `Ctrl+,` | 打开设置 |
| `Escape` | 关闭当前视图/弹窗 |

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
| **AI 适配层** | 自研 aicli | — | 底层 `go-openai` + `ollama/ollama/api` |
| **前端构建** | Vite | v3.2.11 | 前端打包 |
| **前端技术** | 原生 HTML/CSS/JS | — | UI 渲染（无框架） |
| **本地存储** | localStorage | — | UI 状态持久化 |

### 前端文件结构

| 文件 | 行数 | 说明 |
|------|------|------|
| `frontend/src/main.js` | ~5500 | 前端核心逻辑 |
| `frontend/src/js/ai-chat.js` | ~2000 | AI 对话 JS 逻辑 |
| `frontend/src/css/variables.css` | ~210 | 12 主题 CSS 变量 |
| `frontend/src/css/components/ai-chat.css` | ~1580 | AI 对话全部样式 |
| `frontend/src/css/components/settings-panel.css` | — | 设置页样式 |
| `frontend/src/css/components/data-view.css` | — | 数据管理页样式 |
| `frontend/src/css/components/main-content.css` | — | 主页内容样式 |

### 后端结构

| 文件 | 行数 | 说明 |
|------|------|------|
| `app.go` | ~1060 | Wails 绑定层（70+ API） |
| `internal/services/ai_service.go` | ~360 | AI 对话服务 |
| `internal/services/note_service.go` | ~570 | 笔记 CRUD 服务 |
| `internal/services/recall_service.go` | ~80 | 卡片召回服务 |
| `internal/aicli/` | ~540 | AI 客户端（client/openai/ollama） |
| `internal/models/` | — | GORM 数据模型 |

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
| **联网搜索怎么用？** | 在设置中配置 Tavily API Key，对话时开启"联网搜索"开关即可 |
| **可以导出数据吗？** | 可以，在数据管理页面点击"导出数据"，一键备份为 `.jbackup` 文件 |

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
