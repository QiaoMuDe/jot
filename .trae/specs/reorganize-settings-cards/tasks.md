# Tasks
- [x] Task 1: 将快速笔记卡片合并到编辑器选项
  - [ ] `index.html`: 删除快速笔记独立卡片（\<div class="settings-section"\>...\</div\> 第 319-334 行）
  - [ ] `index.html`: 在编辑器选项卡片底部追加快速笔记开关，改卡片标题为"编辑器"
  - [ ] `main.js`: 确认 `loadFontSettings()` → `quickNoteToggle` 加载路径无误
- [x] Task 2: 将笔记排序和分页大小合并为"笔记列表"卡片
  - [ ] `index.html`: 删除排序和分页两个独立卡片，创建新的"笔记列表"卡片
  - [ ] `index.html`: 在新卡片内上下排列排序分段控件和分页分段控件，保留原有 id/class
  - [ ] `main.js`: 确认 `loadSortSettings()` 和 `loadPageSizeSetting()` 在新 DOM 结构下正常工作
  - [ ] CSS: 检查合并后布局间距是否需要微调
- [x] Task 3: 将 AI 联网搜索卡片合并到对话增强
  - [ ] `index.html`: 删除联网搜索独立卡片，将其配置项移入对话增强卡片底部
  - [ ] `index.html`: 改对话增强卡片标题为"对话与搜索"
  - [ ] `main.js`: 确认 `loadAISettings()` 等函数在新 DOM 结构下正常工作

# Task Dependencies
- [Task 1]、[Task 2]、[Task 3] 无依赖关系，可并行执行
