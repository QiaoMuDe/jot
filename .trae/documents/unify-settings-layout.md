# 设置页卡片布局统一计划

## 问题

设置页中不同卡片的设置项布局不一致：

| 布局形态 | 示例 | 问题 |
|---------|------|------|
| `[Label] ┆ [描述............开关]` | 深度思考、卡片召回、知乎搜索、启用密码 | 开关在中间区域 |
| `[Label] ┆ [输入框/按钮........]` | API Key、模型选择、截断字数 | 输入框在右侧区域 |
| `[Label] ┆ [描述........] ┆ [分段控件]` | 日志级别、日志目录（旧体系） | 三段式但使用不同类名 |

用户要求统一为：**名称（左）| 描述（中）| 控件（右）**

---

## 方案

新建统一的 **三列布局模式**，适用于所有 `.ai-setting-item` 使用的设置项：

```
[.ai-setting-label 80px]  [.ai-setting-desc flex:1]  [.ai-setting-control auto]
```

### CSS 变更（1 处）

**文件**: `frontend/src/css/components/settings-panel.css`

1. 新增 `.ai-setting-desc` 类（位于 label 和 control 之间）：
   - `flex: 1` — 撑满中间区域
   - `font-size: 0.8rem; color: var(--text-muted)`
   - `min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis` — 长文本省略

2. 调整 `.ai-setting-control`：
   - 取消 `max-width: 380px` 限制，改为 `width: auto; flex-shrink: 0`
   - 或保留 `max-width` 但使用 `max-width: 320px;` （比原来窄，给描述留空间）

3. 调整 `.ai-setting-toggle-row`：
   - 取消 `justify-content: space-between`
   - 改为 `display: flex; align-items: center; gap: 6px`（与 control 一致）
   - 或直接移除 `.ai-setting-toggle-row`，统一使用 `.ai-setting-control`

### HTML 变更（涉及约 15 个设置项）

**文件**: `frontend/index.html`

统一结构为：

```html
<div class="ai-setting-item">
  <span class="ai-setting-label">设置项名称</span>
  <span class="ai-setting-desc">描述文字（可选，无描述则留空或隐藏）</span>
  <div class="ai-setting-control">控件内容</div>
</div>
```

#### 具体调整：

**类型 A — Toggle 开关项（描述在控件内 → 移到中间）**

| 项 | 当前结构 | 调整方式 |
|----|---------|---------|
| 深度思考 | 描述在 `.ai-setting-toggle-row` 内 | 提取描述到 `.ai-setting-desc`，toggle 放入 `.ai-setting-control` |
| 卡片召回 | 同上 | 同上 |
| 知乎搜索 | 同上 | 同上 |
| 全网搜索 | 同上 | 同上 |
| Tavily搜索 | 同上 | 同上 |
| 启用密码 | 同上 | 同上 |

**类型 B — 输入框/按钮项（描述或后缀提示在控件内 → 判断是否移到中间）**

| 项 | 当前结构 | 调整方式 |
|----|---------|---------|
| AI Base URL | hint 在控件下方单独一行 | hint 提到 `.ai-setting-desc` |
| AI API Key | 无描述 | `.ai-setting-desc` 留空 |
| 模型选择 | 无描述 | `.ai-setting-desc` 留空 |
| 引用截断 | "字符/条" 是控件一部分 | 留在控件内（非描述，是控件单位） |
| 卡片召回数 | "条/次" 是控件一部分 | 同上 |
| 搜索结果数 | "条/次" 是控件一部分 | 同上 |
| Access Secret | 换行提示链接 | 链接留在控件下方或提取到 `.ai-setting-desc` |
| 设置密码 | 无描述 | `.ai-setting-desc` 留空 |
| 回收站清理 | "天前的笔记..." 在控件内 | 提取到 `.ai-setting-desc` |

#### 保留现有特殊布局（不改动）

- 外观卡片 (`.font-setting-row`) — 独立布局体系
- 编辑器卡片 (`.font-setting-row`) — 独立布局体系  
- 日志设置卡片 (`.settings-item` 旧体系) — 已为三段式，保留

---

## 改动范围汇总

| 文件 | 改动类型 | 涉及行数 |
|------|---------|---------|
| `settings-panel.css` | 新增 `.ai-setting-desc` + 调整 `.ai-setting-control` 和 `.ai-setting-toggle-row` | ~20 行 |
| `index.html` | 重构第 504-634 行（对话与搜索）+ 第 714-763 行（锁屏+回收站）的设置项结构 | ~60 行 |
| `main.js` | 检查是否有依赖 `.ai-setting-toggle-row` 或 `.ai-setting-hint` 类的 JS 选择器 | 可能需要少量调整 |

---

## 验证

1. 所有 toggle 开关的开关按钮在右侧同一垂直线上
2. 所有输入框/按钮在右侧同一垂直线上
3. 描述文字在中间列，超出省略
4. 浏览器宽度变化时，中间列自适应，右侧控件布局稳定
