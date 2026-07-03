# Checklist

- [x] HTML 中 FAB 组 DOM 顺序为：`fabNewNote` → `fabAI` → `backToTopBtn`
- [x] AI 按钮使用合适的 SVG 图标（与项目中 AI 菜单图标风格一致）
- [x] AI 按钮具有 `id="fabAI"` 和 `class="fab fab-ai"`
- [x] 网格视图下 FAB 组显示三个按钮，非网格视图隐藏
- [x] CSS 中 `.fab-group` 使用 `flex-direction: column`（非 `column-reverse`）
- [x] CSS 中 `.fab-group.scrolled` 将 `bottom` 从 `28px` 变为 `90px`，带 transition 动画
- [x] `.fab-ai` 样式独立于 `.fab-add`，使用不同的主题色（如紫色变体）
- [x] `.fab-top.visible` 正确控制回到顶部按钮的显隐（opacity + pointer-events）
- [x] JS 中 `els` 对象包含 `fabAI` 引用
- [x] 点击 AI 按钮调用 `switchView('ai-chat')` 跳转到 AI 助手页面
- [x] 滚动超过 300px 时，`fabGroup` 获得 `.scrolled` 类 + `backToTopBtn` 获得 `.visible` 类
- [x] 回到顶部（scrollTop <= 300）时，移除 `.scrolled` 和 `.visible` 类
- [x] 回到顶部按钮点击后页面平滑滚动到顶部
- [x] AI 按钮在非 grid 视图下自动隐藏（复用 FAB 组整体显隐逻辑）
