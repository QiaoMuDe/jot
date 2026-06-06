<div align="center">

# 📝 Jot — 卡片式笔记桌面应用

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go) ![Wails](https://img.shields.io/badge/Wails-2.x-DF367C?style=flat-square&logo=wails) ![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite) ![License](https://img.shields.io/badge/License-MIT-green?style=flat-square) ![Platform](https://img.shields.io/badge/Platform-Windows-blue?style=flat-square)

**Jot** 是一款基于 [Wails v2](https://wails.io/) 构建的轻量级卡片式笔记桌面应用，采用 Go + 原生 Web 技术栈，界面清爽、交互流畅、数据本地存储。

[✨ 特性](#-核心特性) · [🚀 安装](#-安装指南) · [🛠️ 开发](#-开发) · [🔗 仓库](https://gitee.com/MM-Q/jot.git)

</div>

---

## ✨ 核心特性

| 类别 | 功能 |
|------|------|
| 🃏 **卡片式笔记** | 分页懒加载，支持多种排序、置顶/取消置顶、彩色标识、Markdown 双模式编辑 |
| 📁 **笔记本管理** | 三段式侧边栏，书签式隐喻设计，新建/切换/删除（带笔记迁移或级联删除） |
| 🏷️ **标签管理** | 多对多关联，点击筛选，6 个默认标签 + 自定义，批量添加/移除 |
| 🔍 **全文搜索** | 标题 + 内容模糊搜索，250ms 防抖 + 回车即时触发 |
| ♻️ **回收站** | 软删除可恢复，支持批量操作 + 全部清空 |
| 📤 **数据管理** | 一键导出/导入、备份还原、数据统计、恢复出厂设置 |
| 🎨 **6 套主题** | Default / Nord / Monokai Pro / Light / Tokyo Night / Dark，**窗口标题栏与边框跟随主题同步变色** |
| ⌨️ **快捷键** | `Ctrl+F` 搜索 / `Ctrl+N` 新建 / `Ctrl+L` 切换编辑器模式 / 方向键翻页 / 数字键导航 |
| 📝 **Markdown 渲染** | 纯文本/Markdown 双模式编辑，marked + highlight.js 渲染，防抖实时预览 |
| 🔔 **通知系统** | 右上角浮动通知，4 种类型色彩区分，自动消失 |
| 🚀 **快速笔记** | 启动即全屏编辑，自动草稿保存与恢复，捕捉灵感不怕丢失 |
| 🖥️ **窗口自适应** | Windows 原生窗口颜色自适应（DWM API），启动无颜色闪烁，主题切换无缝同步 |

---

## 🚀 安装指南

### 前置依赖

- **Go** ≥ 1.25
- **Wails CLI** v2.12+（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
- **Node.js** ≥ 16

### 从源码构建

```bash
git clone https://gitee.com/MM-Q/jot.git
cd jot
wails build
```

构建产物：`./build/bin/jot.exe`

---

## 🛠️ 开发

```bash
# 开发模式（前端热重载）
wails dev

# 代码格式化 + 静态分析
golangci-lint fmt ./... && golangci-lint run ./...
```

---

## 🤝 贡献指南

1. **Fork** 本仓库
2. **创建特性分支**：`git checkout -b feat/amazing-feature`
3. **遵循现有代码风格**，`golangci-lint` 零警告
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

---

<div align="center">

**如果 Jot 对你有帮助，欢迎 ⭐ Star 支持！**

[![Gitee Stars](https://img.shields.io/badge/dynamic/json?label=Stars&query=$.stargazers_count&url=https://gitee.com/api/v5/repos/MM-Q/jot&style=flat-square&color=yellow)](https://gitee.com/MM-Q/jot)

</div>
