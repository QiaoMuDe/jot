# Tasks

- [ ] Task 1: HTML 结构调整 — 添加表单上移、定义预设色块 HTML、增加标签计数显示位
      - 将 `.tag-add-form` 移到 `.tag-list` 上方
      - 在 `#newTagColor` 的 `<input type="color">` 后面新增预设色块选择器 `<div class="color-presets">`（8 个预设色块 + 1 个自定义入口）
      - 在 `.tag-list` 中每个标签项增加标签使用计数的显示位（`<span class="tag-count">`）
- [ ] Task 2: CSS 样式重写 — 标签项卡牌化、添加区视觉分离、动画效果、空状态
      - `.tag-list`：从 `flex-wrap: wrap` 改为 `display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr))` 网格布局
      - `.tag-item`：改为卡片样式（`card-bg` 背景、圆角、左侧 3px 色条 + 颜色圆点）
      - `.tag-add-form`：从底部移到头部后用分隔线与标签列表区分
      - `.color-presets`：预设色块网格，选中态勾选标记，hover 缩放
      - `.tag-count`：小字号灰色标签计数
      - `.tag-empty`：带 SVG 图标的空状态样式
      - 动画：添加标签时淡入、删除时淡出缩小、hover 交互效果
- [ ] Task 3: JS 逻辑更新 — renderTagList 重写、createTag 颜色选择逻辑、预设色块事件绑定
      - 重写 `renderTagList()`：每个标签项使用新卡片结构（颜色圆点 + 名称 + 计数 + 删除按钮）
      - 预设色块点击事件：点击色块 → 更新 `selectedColor` → 标记选中态 → 选中自定义入口时弹出原生 `<input type="color">`
      - `createTag()`：读取当前 `selectedColor` 而非 `els.newTagColor.value`
      - 删除动画：点击删除按钮时先加 `.tag-deleting` 类 → 动画结束后调用后端 API 删除
- [ ] Task 4: 构建验证 — npm run build 通过
