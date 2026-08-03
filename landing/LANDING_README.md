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
├── media.json          # 截图数据配置（增删图只需改此文件）
├── serve.py            # 本地预览服务器
└── LANDING_README.md   # 本文件
```

---

## 本地开发

### 启动预览

```bash
cd landing
python serve.py                 # 默认端口 8123，自动打开浏览器
python serve.py --port 9000     # 指定端口
python serve.py --no-open       # 不自动打开浏览器
```

服务器基于 Python 标准库 `http.server`，无需安装额外依赖。

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
| 特性 | `#features` | 8 张毛玻璃卡片，hover 浮起，滚动入场动画 |
| 截图 | `#screenshots` | 从 `media.json` 动态渲染，点击弹出 Lightbox |
| 技术栈 | `#tech-stack` | 16 个技术徽章，hover 弹跳缩放 |
| 统计数据 | `#stats` | 4 项数字，滚动触发递增动画（easeOutExpo） |
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

- **768px**：导航栏隐藏非 CTA 链接，特性网格降单列，截图网格降单列，统计网格 2 列，Lightbox 内边距缩小
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

### 占位图说明

当前使用 `https://via.placeholder.com/800x500/...` 在线占位图，后续替换为真实截图时：
1. 将真实截图放入 `landing/images/` 目录
2. 修改 `media.json` 中对应条目的 `src` 为 `"images/真实文件名.png"`
3. 建议尺寸：宽 800px × 高 500px（16:9 比例）

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