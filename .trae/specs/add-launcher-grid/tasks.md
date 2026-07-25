# Tasks

- [ ] Task 1: 创建启动器 CSS 样式文件
  - 新建 `frontend/src/css/components/launcher.css`
  - 定义 `.launcher` 全屏遮罩容器（fixed, z-index: 2100）
  - 定义 `.launcher-mask` 背景半透明遮罩
  - 定义 `.launcher-panel` 居中面板（宽度 ~520px）
  - 定义 `.launcher-search` 搜索框样式（搜索图标 + 输入框 + placeholder 文字）
  - 定义 `.launcher-grid` 4 列 grid 布局（gap: 10px）
  - 定义 `.launcher-item` 卡片样式（flex column 排列图标+文字，hover 高亮）
  - 定义 `.launcher-item.selected` 键盘选中状态
  - 定义入场动画（`@keyframes launcherMaskIn`, `launcherPanelIn`, `launcherItemIn`）
  - 定义离场动画（`.launcher.closing` 状态下的 mask 和 panel 过渡）
  - 定义卡片 stagger 动画延迟（`nth-child` 每项延迟 20ms）
  - 定义 `.launcher.visible` 控制显示状态
  - 在 `frontend/src/css/index.css` 中添加 `@import './components/launcher.css'`

- [ ] Task 2: 在 index.html 中添加启动器 DOM 结构
  - 在 `#app` 内合适位置添加 `.launcher` 容器（隐藏，默认 `display: none`）
  - DOM 结构：
    ```html
    <div id="launcher" class="launcher">
      <div class="launcher-mask"></div>
      <div class="launcher-panel">
        <div class="launcher-search">
          <svg><!-- 搜索图标 --></svg>
          <input id="launcherInput" type="text" placeholder="搜索菜单项...">
        </div>
        <div class="launcher-grid" id="launcherGrid">
          <!-- JS 动态渲染 13 个卡片 -->
        </div>
      </div>
    </div>
    ```

- [ ] Task 3: 实现启动器 JS 逻辑
  - 在 `main.js` 或新建 `frontend/src/js/launcher.js` 中实现
  - 定义 `launcherItems` 数组（13 项，含 action、label、svg）
  - 实现 `renderLauncherItems()`: 根据 `launcherItems` 渲染网格卡片到 `#launcherGrid`
  - 实现 `openLauncher()`: `.launcher` 设为 flex 显示 → requestAnimationFrame 后添加 `.visible` class → 聚焦输入框 → 重置选中索引
  - 实现 `closeLauncher()`: 添加 `.closing` class → 监听 transitionend/animationend → 移除 `.visible`、`.closing` → display 设为 none
  - 实现 `filterLauncherItems()`: 根据输入框 value 切换卡片的 `.hidden` class（搜索过滤）
  - 实现 `handleLauncherKeydown(e)`: 捕获方向键、Enter、Tab
  - 实现 `getVisibleItems()`: 返回当前可见卡片列表
  - 实现 `selectLauncherItem(index)`: 移除旧的 `.selected`，添加新的
  - 实现 `executeLauncherAction(action)`: 调用 `closeLauncher()` 后执行对应的操作函数
  - 事件绑定：输入框 input 事件（过滤）、输入框 keydown（导航）、卡片点击（执行）、遮罩点击（关闭）

- [ ] Task 4: 注册全局键盘快捷键 + ESC 关闭
  - 在 `handleKeyboardNavigation()` 中新增：
    - `Ctrl+P` 分支：阻止默认行为，切换启动器开关
    - ESC 分支：在引用笔记浮层判断之后、搜索弹窗判断之前，优先检查并关闭启动器

## Task Dependencies

- Task 1 (CSS) 和 Task 2 (HTML) 无依赖，可并行
- Task 3 (JS) 依赖 Task 2 (DOM 结构就绪)
- Task 4 (快捷键) 依赖 Task 3 (启动器逻辑就绪)
