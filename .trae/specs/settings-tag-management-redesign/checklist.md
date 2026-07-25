# 设置页 - 标签管理卡片重构 Checklist

## HTML 结构调整
- [ ] 添加表单 `.tag-add-form` 已移到 `.tag-list` 上方
- [ ] 预设色块选择器 `.color-presets` 已添加到原生 color input 之后
- [ ] 预设色块包含 8 种颜色 + 1 个自定义入口（+ 图标）
- [ ] 每个标签项模板中包含标签计数显示位 `.tag-count`

## CSS 样式
- [ ] `.tag-list` 已改为 CSS Grid 布局（`auto-fill, minmax(140px, 1fr)`）
- [ ] `.tag-item` 已改为卡片样式：白色/深色背景、圆角、左侧 3px 色条 + 颜色圆点
- [ ] `.tag-add-form` 已通过分隔线或间距与标签列表视觉区分
- [ ] `.color-presets` 有 hover 缩放和选中态勾选标记
- [ ] `.tag-empty` 有 SVG 图标 + 描述文字
- [ ] 删除动画（`.tag-deleting` 缩小淡出）

## JS 逻辑
- [ ] `renderTagList()` 使用新卡片结构（颜色圆点 + 名称 + 计数 + 删除按钮）
- [ ] 预设色块点击更新 `selectedColor` 并标记选中态
- [ ] 自定义入口点击弹出原生 `<input type="color">`
- [ ] `createTag()` 读取 `selectedColor` 而非 `els.newTagColor.value`
- [ ] 删除时先加 `.tag-deleting` 类，动画结束后调用后端 API

## 构建
- [ ] `npm run build` 无错误
