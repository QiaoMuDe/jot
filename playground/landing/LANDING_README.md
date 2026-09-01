# Jot Landing Page — 开发维护手册

## 项目概述

Jot 落地页（landing page），单页静态网站，用于介绍 Jot 卡片式笔记桌面应用。采用纯原生 HTML/CSS/JS 构建，无框架依赖，通过 JSON 配置文件管理媒体资源。

---

## 目录结构

```
landing/
├── index.html          # 页面结构（入口）
├── css/
│   └── style.css       # 全部样式（按区块注释组织）
├── js/
│   └── main.js         # 全部交互逻辑（按功能模块组织）
├── media.json          # 截图 + 视频数据配置（增删媒体只需改此文件）
├── images/             # 图片资源目录（截图、视频封面）
├── videos/             # 视频资源目录
├── serve.go            # 单文件服务器（静态资源已 go:embed 嵌入二进制）
├── serve.py            # 旧版 Python 预览服务器（备选，读磁盘文件）
└── LANDING_README.md   # 本文件
```

---

## 本地开发

### 启动预览

```bash
cd landing
go run serve.go                 # 默认端口 8123，自动打开浏览器（推荐，仅需 Go 环境）
go run serve.go -port 9000      # 指定端口
go run serve.go -no-open        # 不自动打开浏览器
go run serve.go -host 0.0.0.0   # 局域网/公网可访问
```

Go 版仅使用标准库，无第三方依赖（目录下 `go.mod` 仅为记录模块与 Go 版本，serve.go 本身不引入任何外部包）。

### 服务器部署

```bash
go build -o jot-landing serve.go   # 编译单文件二进制（无需 go.mod）
./jot-landing                      # 任意目录均可运行，无需携带任何静态文件
```

所有静态资源（HTML/CSS/JS/图片/视频/JSON）在编译时已通过 `go:embed` 打包进二进制，部署只需拷贝**一个文件**。

**重要**：素材已内嵌，更新页面代码或素材（如替换 `videos/` 下的视频、`media.json` 文案）后，需要**重新编译**才能生效。

> 旧版 Python 服务器 `serve.py` 仍保留可用（读磁盘文件，改素材即时生效，适合边改边看）：`python serve.py [--port 9000] [--no-open]`

### 文件结构规范

- **index.html** 只放页面结构标签，不写内联样式或脚本
- **css/style.css** 只放样式，每新增一个区块，在对应注释区域追加，或新建一个注释区域
- **js/main.js** 只放交互逻辑，每新增一个功能，在文件末尾追加一个新模块（用 `/* ========== XXX ========== */` 分隔）

---

## 页面区块（按从上到下顺序）

| 区块 | section id | 说明 |
|------|-----------|------|
| 导航栏 | `#navbar` | 固定顶部，滚动后毛玻璃效果 |
| Hero | `#hero` | 深色全屏首屏，背景网格 + 光晕粒子动画 |
| 特性 | `#features` | 9 张毛玻璃卡片（笔记网格/AI 助手/编辑器/智能检索/标签/联网能力/数据管理/本地存储/异构导入），hover 浮起，滚动入场动画 |
| 智能检索 | `#ai-recall` | 向量语义 + 关键词双通道工作流 SVG 动画 + 3 个要点 |
| Agent 智能体 | `#agent` | Chat/Agent/Plan 三态说明 + 16 个工具徽章 + 6 个要点 |
| 异构转换 | `#file-import` | 支持格式徽章 + 清洗转换流程 SVG + 3 个要点 |
| 截图 | `#screenshots` | 从 `media.json` 动态渲染，点击弹出 Lightbox |
| 视频 | `#videos` | 从 `media.json` 动态渲染，点击卡片弹出播放器 |
| 技术栈 | `#tech-stack` | 17 个技术徽章，双行无缝滚动，hover 弹跳缩放 |
| CTA 联系 | `#cta` | 深色渐变卡片，引导联系作者 |
| 页脚 | `footer` | 版权信息 |

---

## 设计规范

### CSS 变量（`:root`）

```css
--primary: #2563EB;          /* 主色（蓝色） */
--primary-light: #3B82F6;
--primary-dark: #1D4ED8;
--accent: #F97316;           /* 强调色（橙色） */
--bg: #F8FAFC;               /* 浅色背景 */
--bg-dark: #0F172A;          /* 深色背景 */
--text: #1E293B;             /* 正文色 */
--text-light: #64748B;       /* 辅助文字色 */
--radius: 16px;              /* 圆角 */
--radius-lg: 24px;           /* 大圆角 */
--ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1);  /* 弹性缓动 */
--ease-out: cubic-bezier(0.16, 1, 0.3, 1);         /* 出缓动 */
--max-width: 1200px;         /* 内容区最大宽度 */
--nav-height: 72px;          /* 导航栏高度 */
```

### 动画规范

- **入场动画**：使用 `.reveal` 类 + IntersectionObserver，初始 `opacity: 0; transform: translateY(30px)`，进入视口后变为 `visible`（`opacity: 1; transform: translateY(0)`），缓动 `var(--ease-out)`，时长 0.7s
- **hover 效果**：translateY(-6px) 浮起 + 阴影加深，缓动 `var(--ease-out)`，时长 0.35s
- **点击效果**：`transform: scale(0.96)` 或 `scale(0.98)`，即时反馈
- **尊重无障碍**：`@media (prefers-reduced-motion: reduce)` 时禁用所有动画
- **Lightbox 动画**：淡入 0.3s + 缩放弹入 0.35s（`var(--ease-spring)`）

### 按钮样式

- `.btn-primary`：蓝色填充 + 阴影，hover 上浮 + 阴影加深
- `.btn-secondary`：半透明毛玻璃边框，hover 增加透明度
- 所有按钮点击：`scale(0.96)` 收缩

### 响应式断点

- **768px**：导航栏隐藏非 CTA 链接，特性网格降单列，截图网格降单列，视频网格降单列，统计网格 2 列，Lightbox / 视频弹窗内边距缩小
- **480px**：Hero 标题缩小，统计数字缩小
- **769px ~ 1024px**：特性网格 2 列

---

## 媒体管理

### 截图管理

所有截图数据通过 `media.json` 配置，格式如下：

```json
{
  "screenshots": [
    {
      "id": "唯一标识",
      "src": "图片路径或 URL",
      "alt": "图片替代文本",
      "caption": "图片说明文字",
      "feature": "所属功能分类"
    }
  ]
}
```

**增删改操作**：
- **增加截图**：在 `screenshots` 数组中新增一个对象，将图片文件放入 `landing/images/` 目录，`src` 写相对路径 `images/xxx.png`
- **删除截图**：删除数组中对应对象，再删除对应图片文件
- **修改截图**：改对应对象的字段即可，`src` 可指向本地路径或外部 URL
- **调整顺序**：调整数组中的对象顺序即可

### 截图替换说明

当前截图使用 `landing/images/` 目录下的本地图片（`1.jpg` ~ `4.jpg`），按 `media.json` 中 `screenshots` 数组的顺序对应（当前目录仅含占位文件，真实素材待补充）。替换截图时：
1. 将新截图放入 `landing/images/` 目录（文件名保持不变，或同步修改 `media.json` 的 `src`）
2. 建议尺寸：宽 800px × 高 500px（16:9 比例）
3. 占位图可临时使用 `https://via.placeholder.com/800x500/...` 在线地址，上线前替换为本地图片

### 视频管理

视频数据同样通过 `media.json` 配置，格式如下：

```json
{
  "videos": [
    {
      "id": "唯一标识",
      "src": "视频路径或 URL",
      "poster": "封面图路径（可选）",
      "title": "视频标题",
      "caption": "视频说明文字"
    }
  ]
}
```

**增删改操作**：
- **增加视频**：将视频文件放入 `landing/videos/` 目录，在 `videos` 数组中新增一个对象，`src` 写相对路径 `videos/xxx.mp4`
- **删除视频**：删除数组中对应对象，再删除对应视频文件
- **修改视频**：改对应对象的字段即可，`src` 可指向本地路径或外部 URL
- **封面图**：`poster` 可选。不填时页面加载后**自动截取视频 1 秒处的画面**作为预览图，无需手动准备封面；视频不存在或截帧失败时回退深色渐变 + 播放按钮
- **格式建议**：mp4（H.264 编码）兼容性最好；建议控制单个视频体积在几十 MB 以内

**封面与播放逻辑**：
- 未配置 `poster` 的视频，**页面加载即主动截帧**生成预览图（隐藏 video 预加载元数据 → seek 到 1 秒 → canvas 截帧），截帧成功后替换卡片封面；配置了 `poster` 的优先显示配置图
- 主动截帧失败时，**打开视频后自动补截一次**作为兜底（`autoCapturePoster`）
- 点击视频卡片弹出居中播放器（仅播放时才真正加载视频），关闭弹窗后自动暂停并释放资源

---

## 常见维护场景

### 新增一个页面区块

1. 在 `index.html` 中对应位置编写新 section 的 HTML 结构
2. 在 `css/style.css` 末尾追加该区块的样式
3. 在 `js/main.js` 末尾追加该区块的交互逻辑（如有）
4. 如需导航栏链接，在 `nav-links` 中添加 `<a href="#新section-id">名称</a>`

### 修改现有区块内容

直接编辑 `index.html` 中对应 section 的 HTML 标签文本即可，样式和逻辑无需改动。

### 修改动画效果

- 统一动画参数在 `css/style.css` 的 `:root` 变量中调整
- 入场动画阈值在 `js/main.js` 的 IntersectionObserver 配置中调整

---

## 技术约束

- 纯原生技术栈：无 Vue/React 等框架，无 jQuery 等库依赖
- 字体：Inter（Google Fonts），已预连字体 CDN
- 图标：全部使用内联 SVG，无图标字体依赖
- 浏览器兼容：现代浏览器（Chrome/Firefox/Edge/Safari），不支持 IE
- 所有代码使用 ES5 语法（`var` / `function`），避免 ES6+ 语法，确保广泛兼容