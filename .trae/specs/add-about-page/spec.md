# 关于页面功能 Spec

## Why

为应用添加一个"关于"页面，用户点击左上角品牌标识后可查看项目信息，包括名称、简介、版本号和项目地址。

## What Changes

- **Go 后端**: 引入 `gitee.com/MM-Q/verman` 库用于版本信息管理；新增 `GetVersion()` 绑定方法返回版本信息；新增 `OpenProjectURL()` 调用 `runtime.BrowserOpenURL` 打开项目地址
- **构建配置**: 利用已有 `Rnx.toml` 中 `-ldflags` 配置注入版本信息（`appName`、`gitVersion`）
- **前端 HTML**: 新增关于页面覆盖层（`#viewAbout`），含项目名、简介、版本号和可点击项目地址
- **前端 CSS**: 关于页面样式（居中卡片、科技感装饰、链接样式）
- **前端 JS**: 品牌名点击事件 → 调用后端获取版本 → 渲染关于页

## Impact

- 新增绑定方法: `GetVersion()` → 返回 `verman.Info` 对象
- 新增绑定方法: `OpenProjectURL()` → 打开浏览器
- Affected specs: 无
- Affected code:
  - `app.go` — 新增绑定方法
  - `frontend/index.html` — 新增 `#viewAbout`
  - `frontend/src/style.css` — 关于页样式
  - `frontend/src/main.js` — 关于页交互逻辑
  - `go.mod` / `go.sum` — 新增依赖

## ADDED Requirements

### Requirement: 关于页面

系统 SHALL 在用户点击左上角品牌标识时显示关于页面。

#### Scenario: 点击品牌标识打开关于页

- **WHEN** 用户点击左上方 "Jot" 品牌文字
- **THEN** 弹出关于页面覆盖层，展示项目名称、简介、版本号和项目链接

#### Scenario: 关于页展示内容

- **WHEN** 关于页面打开
- **THEN** 显示以下内容：
  - 项目名称: "Jot"
  - 简介: 描述项目为卡片式笔记应用
  - 版本号: 从 `verman.V.Version()` 获取
  - 项目地址: "https://gitee.com/MM-Q/jot.git"（可点击链接）

#### Scenario: 点击项目地址

- **WHEN** 用户点击项目地址链接
- **THEN** 调用 `runtime.BrowserOpenURL()` 在默认浏览器中打开项目地址

#### Scenario: 关闭关于页

- **WHEN** 用户点击关闭按钮、覆盖层背景或按下 ESC 键
- **THEN** 关于页面关闭

### Requirement: 后端版本信息

系统 SHALL 通过 `verman` 库提供版本信息。

- **WHEN** 前端调用 `GetVersion()`
- **THEN** 返回包含应用名称、版本号、构建时间等信息的对象

### Requirement: 浏览器打开链接

系统 SHALL 提供通过后端在默认浏览器中打开 URL 的能力。

- **WHEN** 前端调用 `OpenProjectURL(url)`
- **THEN** 系统默认浏览器打开该 URL，返回确认消息
