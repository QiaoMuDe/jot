# 对话与搜索卡片 — 设置项布局统一

## 当前状态分析

对话与搜索卡片中共有 11 个设置项，分两类布局：

### A. 可改造项（9 项）
| 项 | 当前结构 | 问题 |
|---|---------|------|
| 深度思考 | `ai-setting-toggle-row`（desc + toggle 同行） | 无中间 desc 列，toggle 位置不统一 |
| 引用截断 | `ai-setting-control`（input + "字符/条"） | 无中间 desc 列 |
| 最大文件限制数 | `ai-setting-control`（input + "MB"） | 无中间 desc 列 |
| 卡片召回 | `ai-setting-toggle-row`（desc + toggle 同行） | 同深度思考 |
| 卡片召回数 | `ai-setting-control`（input + "条/次"） | 无中间 desc 列 |
| 搜索结果数 | `ai-setting-control`（input + "条/次"） | 无中间 desc 列 |
| 知乎搜索 | `ai-setting-toggle-row`（desc + toggle 同行） | 同深度思考 |
| 全网搜索 | `ai-setting-toggle-row`（desc + toggle 同行） | 同深度思考 |
| Tavily搜索 | `ai-setting-toggle-row`（desc + toggle 同行） | 同深度思考 |

### B. 保持不动项（2 项）
| 项 | 原因 |
|---|------|
| Access Secret（553-572）| `flex-wrap:wrap` 描述在下一行，用户要求不改 |
| Tavily API Key（573-592）| 同上 |

## 改造方案

### 通用结构

```html
<div class="ai-setting-item">
    <span class="ai-setting-label">标签</span>
    <span class="font-setting-desc">描述文本</span>
    <div class="ai-setting-control">控件</div>
</div>
```

### 逐项变更

#### 1. 深度思考（510-518）
- 移除 `.ai-setting-toggle-row` 包裹
- 添加 `<span class="font-setting-desc">发送消息时启用深度思考</span>`
- toggle 开关放入 `.ai-setting-control` 中

#### 2. 引用截断（520-526）
- 添加 `<span class="font-setting-desc">设置 AI 引用笔记内容的最大字符数</span>`
- 保留 `.ai-setting-control` 中的 input + "字符/条" hint

#### 3. 最大文件限制数（528-534）
- 添加 `<span class="font-setting-desc">限制上传文件的最大大小</span>`
- 保留 input + "MB" hint

#### 4. 卡片召回（536-544）
- 同 深度思考：移除 toggle-row，添加 desc，toggle 放入 control

#### 5. 卡片召回数（546-552）
- 添加 `<span class="font-setting-desc">设置每次召回的最大笔记数量</span>`
- 保留 input + "条/次" hint

#### 6. 搜索结果数（594-600）
- 添加 `<span class="font-setting-desc">设置每次联网搜索的返回结果数量</span>`
- 保留 input + "条/次" hint

#### 7. 知乎搜索（602-610）
- 同 深度思考：移除 toggle-row，添加 desc，toggle 放入 control

#### 8. 全网搜索（612-620）
- 同 深度思考

#### 9. Tavily搜索（622-630）
- 同 深度思考

### 无需修改项
- Access Secret（553-572）— 保持不动
- Tavily API Key（573-592）— 保持不动

### 文件
- 仅修改 `frontend/index.html`（对话与搜索卡片 HTML 结构调整）
- 无 CSS/JS 改动（复用已有的 `.font-setting-desc` 和 `.ai-setting-control`）

## 验证
1. 打开设置页 → 对话与搜索卡片
2. 9 个改造项的控件都在右侧统一对齐
3. toggle 开关在右、input 在右、"字符/条"等 hint 文字跟在 input 后面同行
4. Access Secret 和 Tavily API Key 保持原样（描述在下一行）
