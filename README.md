<div align="center">

# Jot — 卡片式笔记桌面应用

![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go) ![Wails](https://img.shields.io/badge/Wails-v2.12.0-DF367C?style=flat-square&logo=wails) ![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite) ![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

轻量级卡片式笔记桌面应用，基于 Wails v2 构建，Go + 原生 Web 技术栈，数据本地存储。内置 AI Agent 智能体与本地语义检索，让笔记不仅能记，更能被理解。

[✨ 特性](#-特性) · [🛠️ 技术栈](#️-技术栈) · [🚀 快速开始](#-快速开始) · [🔗 仓库](https://gitee.com/MM-Q/jot.git)

</div>

---

## ✨ 特性

- **笔记管理** — 卡片网格 + CodeMirror 6 Markdown 编辑器（查找替换、语法高亮、全屏）、笔记本/标签/搜索/日历/回收站，支持导出与图片
- **AI 助手** — eino 驱动，OpenAI 兼容端点，Agent / Plan 双模式（Agent 自动调度工具、Plan 先规划后执行），流式输出、深度思考、13 项 AI 技能（翻译/编程/写作/解题/需求规格/润色/摘要/文案/工作总结/提示词生成/人物档案/角色扮演/深度研究）、16 个内置工具（管理笔记/待办/标签/笔记本/JSON 处理/网页读取/长文本摘要/创建与更新计划等），MCP 服务器扩展（设置页完整管理 + 连接池预热 + 断线重连），每个工具可单独启用/禁用，向量+关键词混合召回
- **数据管理** — 数据统计、向量索引、批量管理、待办清单、备份/还原、数据库瘦身
- **设计系统** — 11 套主题、纯手写 CSS、统一滚动条与弹性动效
- **其他** — 无边框窗口、字体设置、锁屏密码、Mermaid 图表、命令启动器（`Ctrl+P`）、全局快捷键、日志系统

## 🛠️ 技术栈

Go 1.26+ · Wails v2 · SQLite（纯 Go 驱动，无 CGO）· GORM · CodeMirror 6 · marked · highlight.js · Mermaid · sqlite-vec · eino · Vite

## 🚀 快速开始

**前置依赖**：Go ≥ 1.26、Wails CLI v2.9+、Node.js ≥ 16

```bash
git clone https://gitee.com/MM-Q/jot.git
cd jot
wails build    # 产物输出至 ./build/bin/
wails dev      # 开发模式（前端热重载）
```

发布版安装包请前往 [Releases](https://gitee.com/MM-Q/jot/releases) 下载。

## 🤝 贡献

Fork 本仓库 → 创建特性分支 → 遵循现有代码风格（golangci-lint 零警告）→ 提交 PR。

## 📄 许可证

本项目采用 **MIT License** 开源。

## 📬 相关链接

| 资源 | 链接 |
|------|------|
| 🪧 项目仓库 | [https://gitee.com/MM-Q/jot.git](https://gitee.com/MM-Q/jot.git) |
| 🐛 提交 Issue | [https://gitee.com/MM-Q/jot/issues](https://gitee.com/MM-Q/jot/issues) |
| 🛠️ Wails 框架 | [https://wails.io](https://wails.io) |
| 🔍 sqlite-vec | [https://github.com/asg017/sqlite-vec](https://github.com/asg017/sqlite-vec) |

---

<div align="center">

**如果 Jot 对你有帮助，欢迎 ⭐ Star 支持！**

[![Gitee Stars](https://img.shields.io/badge/dynamic/json?label=Stars&query=$.stargazers_count&url=https://gitee.com/api/v5/repos/MM-Q/jot&style=flat-square&color=yellow)](https://gitee.com/MM-Q/jot)

</div>
