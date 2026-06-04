# Tasks

- [X] Task 1: 后端引入 verman 库并添加绑定方法
  - 安装 `gitee.com/MM-Q/verman` 依赖
  - 在 `app.go` 中添加 `GetVersion()` 方法返回 `verman.V.Version()`
  - 在 `app.go` 中添加 `OpenProjectURL(url string)` 方法调用 `runtime.BrowserOpenURL`

- [X] Task 2: 前端 HTML 添加关于页面视图
  - 在 `index.html` 中新增 `#viewAbout` 覆盖层
  - 包含项目名称、简介、版本号展示区、项目地址链接、关闭按钮

- [X] Task 3: 前端 CSS 添加关于页面样式
  - 在 `style.css` 中添加关于页面覆盖层、卡片、标题、链接、装饰等样式

- [X] Task 4: 前端 JS 添加关于页面交互逻辑
  - 品牌名 `.brand-name` 添加点击事件 → 调用 `GetVersion()` → 渲染关于页
  - 项目链接点击事件 → 调用 `OpenProjectURL()`
  - 关闭按钮和背景点击关闭
  - ESC 按键关闭关于页面

# Task Dependencies

- Task 1 无依赖
- Task 2 无依赖
- Task 3 无依赖
- Task 4 依赖 Task 1
