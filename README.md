<div align="center">

# Jot — 卡片式笔记桌面应用

![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go) ![Wails](https://img.shields.io/badge/Wails-v2.12.0-DF367C?style=flat-square&logo=wails) ![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite) ![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

**Jot** 是一款基于 [Wails v2](https://wails.io/) 构建的轻量级卡片式笔记桌面应用，采用 **Go + 原生 Web 技术栈**（无 Vue/React），界面清爽、交互流畅、数据本地存储。

[✨ 特性](#-核心特性) · [🚀 安装](#-安装指南) · [🛠️ 开发](#-开发) · [🔗 仓库](https://gitee.com/MM-Q/jot.git)

</div>

---

## ✨ 核心特性

### 📝 笔记管理

- **卡片式笔记网格** — 笔记本 → 笔记卡片 → 编辑器的三步交互范式，直观的文件夹-文件-编辑结构
- **CodeMirror 6 编辑器** — 专业 Markdown 编辑体验，支持行号、撤销重做、查找替换（`Ctrl+F`/`Ctrl+H`）、Tab 缩进、语法高亮、自动换行、全屏编辑（`Ctrl+E`）
- **编辑器操作菜单** — 40+ 操作项，覆盖格式化（JSON/XML/HTML/CSS/JS/SQL/CSV/YAML/TOML）、文本转换（7 项）、文本清理（5 项）、编码解码（6 项）、MD 语法插入（22 项）
- **笔记类型（文件扩展名）** — 支持 `.md`/`.txt`/`.py`/`.js` 等任意后缀，一键切换纯文本/Markdown 模式（`Ctrl+L`），按语言加载对应语法高亮
- **TOC 目录侧栏** — 编辑器内自动生成标题大纲，点击锚点跳转，可开关
- **14 套系统主题 + 13 套代码高亮主题** — 全局 CSS 变量联动（`--bg`/`--accent`/`--border` 等），所有组件自动适配；代码高亮主题可独立配置且联动 AI 代码块
- **笔记本（目录）系统** — 创建、编辑、删除笔记本，按笔记本筛选笔记，支持笔记单条/批量移动笔记本
- **标签系统** — 自定义标签（名称 + 颜色），按标签筛选笔记，支持无限滚动标签选择器与批量添加/移除标签
- **笔记排序与分页** — 按更新时间/创建时间/名称排序，每页 20-100 条可配置
- **搜索弹窗（`Ctrl+F`）** — 200ms 防抖 + 笔记本/日期范围/排序/标签筛选器，支持全文搜索
- **笔记右键菜单** — 复制、导出、查看、编辑、移动、置顶、删除
- **图片支持** — 拖拽/粘贴插入本地图片，点击放大（灯箱预览），支持清理未被引用的孤儿图片
- **笔记导出** — 一键导出单条笔记为 Markdown 文件
- **大文件优化** — 超大 Markdown 笔记自动切换为纯文本模式，保证流畅
- **笔记日历视图** — 按创建日期浏览笔记，日历网格 + 墨水圆点统计 + 按月摘要 + 按日笔记列表，支持视图切换

### 🤖 AI 助手

- **eino 驱动对话引擎** — 基于 eino 库 + `einocli` 薄适配层（OpenAI 兼容协议），统一流式接口
- **OpenAI 兼容端点** — 支持 DeepSeek、通义千问等兼容服务商（已移除 Ollama 原生协议）
- **API 配置预设** — 多 API 配置管理，一键切换预设，内置服务商预设，支持配置预设导入导出、测试连接
- **流式输出** — 逐块推送 + Markdown 渲染 + 代码高亮（hljs），支持随时停止生成
- **深度思考** — 支持 `reasoning_content` 字段，思维链可折叠展示
- **多来源联网搜索** — 支持 Tavily 通用搜索、知乎搜索、全网搜索三个独立来源，可同时开启；发送前自动调用 AI 精化搜索词（查询优化）
- **卡片召回（混合检索）** — 向量检索（sqlite-vec 余弦距离）+ GSE 关键词检索（LIKE 匹配）双路并行召回，按命中优先级排序（双命中 > 仅向量 > 仅关键词），命中块自动补充相邻块上下文，去重合并后注入 AI 对话；支持按笔记本过滤召回范围
- **笔记切块优化** — 切块前注入元数据前缀（标题/标签/创建时间）提升检索命中率，空行保留为段落分隔符避免过早切散，注入 AI 时自动剥离前缀减少 token 消耗
- **引用笔记与文件上传** — 手动选择笔记引用到对话中（支持全选、键盘操作、笔记本/标签筛选），支持拖拽上传文件
- **办公文件智能转换** — 集成 markitdown，上传 `.docx`/`.xlsx`/`.xls`/`.pptx`/`.pdf`/`.epub`/`.zip` 等文件自动转为 Markdown 注入上下文（60s 超时保护）
- **角色扮演** — 指定笔记作为 AI 的角色/身份设定
- **12 项技能** — 10 项互斥一键技能（翻译、编程、写作、解题答疑、需求规格、文本润色、内容摘要、文案生成、工作总结、提示词生成）+ 人物档案 + 角色扮演
- **优化表达** — 输入框内嵌一键优化按钮，支持还原原文
- **多会话管理** — 创建/切换/删除/重命名/置顶会话，会话导出为 Markdown，侧栏折叠，Context Size 实时显示
- **消息操作** — 编辑、删除、重新发送、重新生成（再生）、停止生成、耗时显示，基于消息 ID 的精确操作
- **消息懒加载** — 分页加载历史消息，长会话流畅不卡顿
- **AI 消息右键菜单** — 复制、保存为笔记、删除

### 🗂️ 数据管理

- **数据统计面板** — 信笺风格展示笔记/标签/回收站/笔记本/AI 会话/AI 消息/向量索引/数据库大小 8 项统计
- **向量索引状态** — 实时查看已量化笔记数、片段总数、占用字节，支持一键全量量化/按笔记本量化/按 ID 量化
- **批量管理** — FAB 入口进入选择模式，批量置顶/删除/移动/添加或移除标签
- **待办清单** — 完整的待办 CRUD，带筛选和动画交互，FAB 内联输入
- **备份/还原** — 一键导入/导出完整数据（`.zip` 格式，含图片），支持一键备份/还原到数据目录并展示备份状态
- **数据库瘦身 VACUUM** — 释放空间
- **回收站** — 软删除机制，支持还原/永久删除/全部清空，混合显示笔记和笔记本条目，可配置自动清理天数
- **维护工具** — 恢复出厂设置、清理未引用图片、清空 AI 会话、清空已完成待办、打开数据/日志目录

### 🎨 设计系统

- **纯手写 CSS** — 无 UI 框架依赖，极致轻量
- **CSS 变量主题系统** — 14 套主题，统一设计 Tokens（圆角/阴影/间距/语义色）
- **过渡动画** — 骨架屏 shimmer、stagger 延迟、hover 分层反馈、弹性缓动
- **统一的滚动条** — 6px 细条，联动全部 14 主题
- **通知系统** — NotificationManager 单例，4 种类型 + undo 撤销

### 🔧 其他

- **无边框窗口** — Frameless 窗口 + 自绘标题栏与窗口控制按钮，自定义拖拽区
- **字体设置** — 字体族 + 大小，联动 CSS 变量，实时预览
- **锁屏密码** — SHA-256 哈希存储，毛玻璃锁屏遮罩
- **Mermaid 图表** — Markdown 代码块按需渲染，源码/视图切换
- **启动器网格** — `Ctrl+P` 呼出命令启动器，搜索并跳转任意功能
- **全面快捷键** — `Ctrl+F` 全局搜索、`Ctrl+N` 新建笔记、`Ctrl+J` AI 侧栏、`Ctrl+Q` 退出、`F11` 全屏等
- **日志系统** — 基于 fastlog 的文件日志，支持设置页动态调整日志级别、一键打开日志目录
- **关于页面** — 基于 verman 构建时注入版本号并展示
- **更多菜单** — 毛玻璃精工卡设计，双层阴影 + 弹性入场动画
- **Markdown 语法手册** — 10 张语法卡片，双栏源码/预览，附目录导航与"打开编辑器试试"
- **快捷键说明页** — 可滚动列表，一键呼出
- **响应式布局** — 全屏/小窗口适配

---

## 🏗️ 技术栈

| 层级 | 技术 | 版本 | 用途 |
|------|------|------|------|
| **桌面框架** | Wails v2 | v2.12.0 | 桌面窗口 + Go↔JS Bridge |
| **后端语言** | Go | go1.26+ | 后端业务逻辑 |
| **数据库** | SQLite | — | 本地数据存储 |
| **数据库驱动** | glebarez/sqlite | v1.11 | 纯 Go SQLite 驱动（无 CGO） |
| **ORM** | GORM | v1.31 | 对象关系映射 |
| **编辑器** | CodeMirror 6 | @codemirror/view v6.43 | 笔记 Markdown 编辑器 |
| **Markdown 渲染** | marked | v18.0 | Markdown → HTML |
| **代码高亮** | highlight.js | v11.11 | 代码块语法高亮 |
| **图表渲染** | Mermaid | v11.16 | Markdown 代码块图表按需渲染 |
| **向量检索** | sqlite-vec | — | SQL 内向量余弦距离检索 |
| **中文分词** | go-ego/gse | v1.0 | AI 混合检索关键词分词 + 停用词过滤 |
| **文档转换** | 自研 markitdown 集成 | — | PDF/Word/Excel/PPT/EPUB 等 → Markdown |
| **联网搜索** | hekmon/tavily + zhihu-go | — | Tavily/知乎/全网搜索 |
| **AI 适配层** | einocli（薄适配层） | — | 基于 eino（`cloudwego/eino` + `eino-ext`）驱动 OpenAI 兼容端点 |
| **日志库** | MM-Q/fastlog | v1.6 | 文件日志 |
| **版本注入** | MM-Q/verman | v0.0.19 | 构建时注入版本号 |
| **前端构建** | Vite | v3.2.11 | 前端打包 |
| **前端技术** | 原生 HTML/CSS/JS | — | UI 渲染（无框架） |
| **本地存储** | localStorage | — | UI 状态持久化 |

---

## 🚀 安装指南

### 前置依赖

- **Go** ≥ 1.26
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
| **数据存在哪里？** | 所有数据存储在本地 SQLite 文件中，位于用户目录下的 `.jot/data/jot.db`，图片存储于 `.jot/images/`，备份存储于 `.jot/backup/`，日志存储于 `.jot/logs/` |
| **AI 对话需要什么？** | 需要 API Key，支持 OpenAI 兼容服务商（DeepSeek、通义千问等），在设置页「预设管理」中配置 API 地址、Key 与模型 |
| **联网搜索怎么用？** | 在设置中配置 API Key（Tavily 或知乎 Access Secret），对话时开启"联网搜索"开关即可，支持 Tavily/知乎/全网搜索三个来源 |
| **AI 卡片召回怎么用？** | 对话时开启"卡片召回"开关即可。AI 回复前自动从笔记库中召回相关笔记片段注入上下文，支持向量检索 + 关键词检索双路混合召回。需先在数据管理页面配置 Embedding 模型并执行量化索引 |
| **可以导出数据吗？** | 可以，在数据管理页面可导出为 `.zip`（完整备份，含图片）或 `.db`（SQLite 数据库文件），也可一键备份/还原到数据目录 |

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
| 🔍 sqlite-vec | [https://github.com/asg017/sqlite-vec](https://github.com/asg017/sqlite-vec) |

---

<div align="center">

**如果 Jot 对你有帮助，欢迎 ⭐ Star 支持！**

[![Gitee Stars](https://img.shields.io/badge/dynamic/json?label=Stars&query=$.stargazers_count&url=https://gitee.com/api/v5/repos/MM-Q/jot&style=flat-square&color=yellow)](https://gitee.com/MM-Q/jot)

</div>
