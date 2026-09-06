# Jot Landing Page — 开发维护手册

## 项目概述

Jot 落地页（landing page），单页静态网站，用于介绍 Jot 会思考的卡片笔记。采用纯原生 HTML/CSS/JS 构建，无框架依赖，通过 JSON 配置文件管理媒体资源。

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
| Hero | `#hero` | 深色全屏首屏：星野粒子网络（鼠标连线/排斥）+ 旋转轨道环 + 预加载进度条 + 打字机字幕 + 标题逐字 reveal |
| 特性 | `#features` | 9 张直角 HUD 卡片（笔记网格/AI 助手/编辑器/智能检索/标签/联网能力/数据管理/本地存储/异构导入），3D 倾斜 + 光标光晕 + 四角括号撑开发光，错帧弹簧入场 |
| 智能检索 | `#ai-recall` | 向量语义 + 关键词双通道工作流 SVG 动画 + 3 个要点 |
| Agent 智能体 | `#agent` | Chat/Agent/Plan 三态说明 + 16 个工具徽章（含 `manage_memory` 长期记忆工具）+ 6 个要点 |
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
--bg0: #05080f;               /* 深空底色 */
--bg1: #0a1120;               /* 次级底色 */
--panel: #0d1626;             /* 面板/卡片底 */
--line: rgba(148,203,255,.10);        /* 细描边（HUD 靶框） */
--line-strong: rgba(62,231,255,.28);  /* 强描边 */
--cyan: #3ee7ff;              /* 主青（强调光/连接线） */
--cyan-soft: rgba(62,231,255,.14);
--amber: #ffc46b;             /* 强调琥珀 */
--ink: #dbeafe;               /* 主文字 */
--ink-dim: #8aa3c4;           /* 次级文字 */
--ink-faint: #5b7196;         /* 弱文字 */
--font-display: 'Orbitron','Noto Sans SC',sans-serif;  /* 展示字体 */
--font-body: 'Noto Sans SC','PingFang SC','Microsoft YaHei',sans-serif;
--nav-h: 72px;                /* 导航栏高度 */
--maxw: 1180px;               /* 内容区最大宽度 */
--ease: cubic-bezier(.22,.9,.32,1);      /* 默认缓动 */
--ease-spring: cubic-bezier(.34,1.56,.64,1);  /* 弹性过冲（入场/角撑开/按钮回弹） */
--radius: 2px;                /* HUD 直角 */
--radius-lg: 6px;             /* 大直角 */
```

### 动画规范

- **入场动画（普通元素）**：`.reveal` 类 + IntersectionObserver，初始 `opacity: 0; transform: translateY(34px)`，进视口后 `.in` 过渡到可见，缓动 `var(--ease)`，时长 .9s；可用 `.r1`~`.r9` 类做错帧延迟
- **特性卡**：`translateY(44px) scale(.985)` → `none`，`var(--ease-spring)` 弹簧过冲 + 按索引错帧（r1~r9）
- **Hero 标题**：逐字 `.cl` reveal（上浮 + 去模糊 + 弹簧），每字 `70ms` 错帧；`--i` 控制时序
- **hover 效果**：按钮 `translateY(-2px)` + 青光阴影；特性卡 3D 倾斜 + 角括号 `scale(1.18)` 撑开发光
- **点击效果**：所有按钮 `.btn:active` → `transform: scale(.96)`，即时反馈 + 回弹
- **光带**：CTA 面板 `.sweep` 半透明青光带 `5.5s` 无限横向扫过
- **粒子网络**：Hero canvas 星野连线，鼠标进入产生琥珀连线 + 粒子排斥（`initStars`）
- **尊重无障碍**：`@media (prefers-reduced-motion: reduce)` 禁用所有动画
- **Lightbox/视频弹窗**：淡入 0.3s + 缩放弹入 0.35s（`var(--ease-spring)`）

### 按钮样式

- `.btn-primary`：青色描边 + 渐变上下缘，`::after` 光带扫过（shine），hover 上浮 + 青光阴影
- `.btn-ghost`：细描边，hover 描边+文字变青 + 青色微光
- 所有按钮点击 `.btn:active`：`scale(0.96)` 收缩

### 响应式断点

- **768px**：导航栏隐藏非 CTA 链接，特性网格降单列，截图网格降单列，视频网格降单列，Lightbox / 视频弹窗内边距缩小
- **480px**：Hero 标题缩小
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
3. 建议直接放入真实界面截图（深色主题应用截图为佳，与页面风格统一）；上线前务必用本地图片

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
- **封面图**：`poster` 建议填写，指向 `images/` 下的封面帧；缺失时卡片仅显示播放键 + 深色渐变底（无自动截帧）
- **格式建议**：mp4（H.264 编码）兼容性最好；建议控制单个视频体积在几十 MB 以内

**封面与播放逻辑**：
- **`poster` 为必填**：main.js 直接读取 `item.poster` 作为封面 `<img>`（无自动截帧功能）。封面缺失时卡片仅显示居中播放键 + 深色渐变底
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
- 字体：展示用 **Orbitron**（`.hero-en`/kicker/序号等），正文用 **Noto Sans SC**（Google Fonts），已预连字体 CDN
- 图标：全部使用内联 SVG，无图标字体依赖
- 浏览器兼容：现代浏览器（Chrome/Firefox/Edge/Safari），不支持 IE
- 所有代码使用 ES5 语法（`var` / `function`），避免 ES6+ 语法，确保广泛兼容