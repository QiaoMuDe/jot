# 笔记列表设置卡片三栏布局改造计划

## 总结
将"笔记列表"设置卡片中的两个设置项（排序方式、每页条数）从旧式布局改为与"日志设置"等其他卡片一致的三栏布局：标签在左、描述在中间、控件在右。

## 当前状态分析
- **排序方式**（index.html:660-668）：使用 `.page-size-control` 容器，标签 `<label>` 在控件上方，不是三栏布局
- **每页条数**（index.html:669-680）：使用 `.page-size-control` 容器，标签在上方 + 分段控件 + 右侧独立 `.page-size-label`（显示"X 条 / 页"）
- **JS**（main.js:402）：`els` 中定义了 `pageSizeLabel`
- **JS**（main.js:1647）：点击分页按钮时更新 `els.pageSizeLabel.textContent`
- **JS**（main.js:7753）：加载设置时更新 `els.pageSizeLabel.textContent`
- **CSS**（settings-panel.css:354-418）：`.page-size-control` 和 `.page-size-label` 仅在此处使用（`main-content.css` 中同名不同类，不影响）

## 目标布局（三栏）
```
Row 1: [排序方式(80px固定)]  [描述文本(flex:1)]     [分段控件(右对齐)]
Row 2: [每页条数(80px固定)]  [描述文本(flex:1)]     [分段控件(右对齐)]
```

其中"每页条数"的"X 条 / 页"信息从右侧标签移到中间描述中。

## 改动内容

### 1. index.html — 笔记列表卡片结构调整
**路径**: `frontend/index.html` 第 660-680 行

**改动**: 将两个 `.page-size-control` 容器替换为 `.ai-setting-item` 三栏布局

- 排序方式行:
  - 标签: `.ai-setting-label` → "排序方式"
  - 描述: `.font-setting-desc` → "选择笔记列表的默认排序方式"
  - 控件: 保留 `segmented-control sort-segmented#sortControl`，添加 `style="flex-shrink:0;"`

- 每页条数行:
  - 标签: `.ai-setting-label` → "每页条数"
  - 描述: `.font-setting-desc#pageSizeSettingDesc` → "每页显示 20 条"（动态更新）
  - 控件: 保留 `segmented-control#pageSizeControl`，添加 `style="flex-shrink:0;"`
  - **移除** `page-size-label#pageSizeLabel`（信息并入中间描述）

### 2. main.js — els 引用更新
**路径**: `frontend/src/main.js`

| 位置 | 改动 |
|------|------|
| ~402 | 移除 `pageSizeLabel: $('pageSizeLabel')`（DOM 中不再有该元素） |
| ~（新增） | 新增 `pageSizeSettingDesc: $('pageSizeSettingDesc')` |
| ~1647 | 将 `els.pageSizeLabel.textContent = \`${size} 条 / 页\`` 改为 `els.pageSizeSettingDesc.textContent = \`每页显示 ${size} 条\`` |
| ~7753 | 将 `els.pageSizeLabel.textContent = \`${cfg.page_size} 条 / 页\`` 改为 `els.pageSizeSettingDesc.textContent = \`每页显示 ${cfg.page_size} 条\`` |

### 3. CSS — 清理旧样式（可选）
**路径**: `frontend/src/css/components/settings-panel.css`

`page-size-control` 和 `page-size-label` 在 settings-panel.css 中不再使用，可移除（但需确认 `main-content.css` 中的同名类不受影响，它们是独立定义且用于主内容区，无影响）。

## 不变的部分
- 分段控件的 HTML 结构、JS 绑定逻辑（`initSortSettings` / `moveIndicator`）保持不变
- CSS 分段控件样式（`segmented-control` / `segmented-btn` / `segmented-indicator`）不变
- 排序和分页的保存/加载/业务逻辑不变

## 验证步骤
1. 运行 `npx vite build` 确认前端构建无错误
2. 在开发环境中打开设置页，检查笔记列表卡片：
   - 排序方式：标签在左，描述在中间，三段控件在右
   - 每页条数：标签在左，描述在中间（显示"每页显示 X 条"），五段控件在右
3. 点击切换分页大小，确认描述文字同步更新
4. 刷新页面，确认已保存的分页大小正确加载且描述一致
