# 预设配置条目品牌徽章（替代原服务商标识）计划

## Summary

为设置页「对话连接 / 量化连接」的**配置预设下拉列表项**与**预设管理列表行**添加**智能品牌徽章**（圆角方形色块）：

* 按预设的 `base_url` 域名离线识别常见 AI 服务商（DeepSeek / OpenAI / 通义 / Kimi / 智谱 / 硅基流动 / Ollama / Gemini / Anthropic / Groq / xAI / Mistral / MiniMax / 火山豆包 / 讯飞 / 百度千帆 / 腾讯混元 / 零一万物 / 百川 / OpenRouter / Gitee AI 等），显示**品牌配色 + 1\~2 字简称**；

* 识别不出时回退为**域名/名称首字符 + 哈希稳定配色**；

* 下拉触发按钮（`presetLabel` / `aiEmbedPresetLabel`）同步前置小号徽章，保持视觉连贯。

纯前端改动，不涉及后端、数据库、HTML 结构。恢复原来 `preset-provider-badge` 的"条目前有标识"观感，同时具备品牌辨识度。

## Current State Analysis

* 设置页预设下拉：[renderPresetList](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L1939-L1972) 的列表项只渲染 `p.name` 纯文本，无任何标识。

* 预设管理列表行：[createPresetRowElement](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3314-L3349) 显示名称 + URL + 编辑/删除按钮，同样无标识。

* 历史版本曾用 `preset-provider-badge` 文本徽章（"OpenAI"/"Ollama"）标注服务商，随服务商概念移除后一并删除（git 历史 `ef7d4fa` / `a591c92` 可证）。

* 可用数据：[APIProfile](file:///d:/峡谷/Dev/本地项目/jot/internal/models/api_profile.go) 含 `name` / `base_url` / `api_key`，徽章可从 `base_url` 的域名离线识别。

* 前端为原生 ESM（Vite 构建）：`main.js` 通过 `import ... from './js/*.js'` 引用模块；构建脚本 `npm run build`。

* 设计令牌（[variables.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/variables.css)）：`--radius-sm: 6px`、`--text-*`、多套明/暗主题；下拉项复用 `theme-select-item`（`.active` 时背景为主题色、文字白色）。

## Proposed Changes

### 1. 新建 `frontend/src/js/preset-brand.js`（品牌识别 + 徽章生成）

**为什么**：识别逻辑与 DOM 生成独立成模块，避免继续膨胀 9000+ 行的 main.js，便于维护与复用（下拉项、管理行、触发按钮三处共用）。

**内容**：

* 导出 `createPresetBadge(baseURL, name, small=false)` → 返回徽章 `<span class="preset-brand-badge[ sm]">` DOM（内联 `background` / `color`，尺寸形态走 CSS）。

* 导出 `detectBrand(baseURL)` → 匹配到返回 `{ short, bg }`；否则 `null`。

* 内部服务商识别表 `BRANDS`：每项 `{ keywords: string[], short, bg }`。匹配规则：解析 `new URL(baseURL).hostname` 转小写后，包含任一 keyword 即命中（顺序优先：先精确 host 匹配、再 contains 匹配）。内置清单（品牌色均为白字可读的单色）：

  * `openai` → `O` / `#10A37F`；`deepseek` → `DS` / `#4D6BFE`；`dashscope`、`aliyun` → `QW` / `#615CED`；`moonshot` → `K` / `#7C3AED`；`bigmodel`、`z.ai` → `GLM` / `#3859FF`；`siliconflow` → `SF` / `#2563EB`；`localhost`、`127.0.0.1`、`0.0.0.0`、`11434` → `本地` / `#374151`；`generativelanguage` → `GM` / `#4285F4`；`anthropic` → `CL` / `#D97757`；`groq` → `GQ` / `#F55036`；`x.ai`、`grok` → `XA` / `#374151`；`mistral` → `MS` / `#FF7000`；`minimax` → `MM` / `#7B61FF`；`volces` → `DB` / `#3370FF`；`stepfun` → `SP` / `#3B82F6`；`xf-yun` → `XF` / `#2479FF`；`baidubce` → `BF` / `#2932E1`；`hunyuan`、`tencent` → `HY` / `#0052D9`；`baichuan` → `BC` / `#3B6CFF`；`lingyiwanwu`、`01.ai` → `01` / `#0EA5E9`；`openrouter` → `OR` / `#6B4AEA`；`gitee.ai` → `GE` / `#C71D23`；`360.cn` → `360` / `#E60012`。

* 兜底：未命中时用 `hashColor`（字符串 FNV 哈希）从 10 色固定色板 `["#0EA5E9","#8B5CF6","#F59E0B","#10B981","#EF4444","#6366F1","#EC4899","#14B8A6","#F97316","#3B82F6"]` 取稳定色；`short` 取 `name` 去空白后首字符（中文亦可），为空则取 host 首个非空字符段首字母，再空则 `?`。

* 失败兜底：`baseURL` 为空或 URL 解析失败 → 直接用 `name` 首字符 + 色板色，不抛异常。

### 2. 修改 `frontend/src/main.js`

* 顶部 import：`import { createPresetBadge } from './js/preset-brand.js';`

* **`renderPresetList`**（L1939）：

  * 每个下拉项 `item.classList.add('preset-option')`（配合 CSS 做 flex 布局）；

  * 在 `nameSpan` 之前 `item.appendChild(createPresetBadge(p.base_url, p.name))`；

  * label 更新逻辑抽为内部 helper `setPresetTriggerLabel(labelEl, profile)`：有匹配预设 → 清空 label 后 append 小号徽章 + 名称文本 span（名称 span 加类 `.preset-option-name` 以便 ellipsis）；无匹配 → 文本 `选择预设`；空列表 → 文本 `无预设配置`（保持现行为）。

* **`createPresetRowElement`**（L3314）：`nameRow` 中在 `nameSpan` 前 `appendChild(createPresetBadge(p.base_url, p.name))`。

### 3. 修改 `frontend/src/css/components/settings-panel.css`（在 `/* ── API 配置预设 ── */` 段落内追加）

* `.preset-option { display:flex; align-items:center; gap:8px; }` — 下拉项布局（覆盖 theme-select-item 默认块级）。

* `.preset-brand-badge { flex-shrink:0; display:inline-flex; align-items:center; justify-content:center; width:22px; height:22px; border-radius:6px; font-size:10px; font-weight:600; color:#fff; line-height:1; user-select:none; }` — 通用徽章。

* `.preset-brand-badge.sm { width:18px; height:18px; font-size:9px; border-radius:5px; }` — 触发按钮小号徽章。

* `#presetLabel, #aiEmbedPresetLabel { display:flex; align-items:center; gap:6px; min-width:0; }` — 触发按钮 label 支持徽章+文本。

* `.preset-option-name { min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }` — 名称超长截断。

* 备注：下拉项 `.active` 时背景为主题色，徽章自带品牌底色不受影响，辨识度保留。

### 4. 不改动的部分

* 不修改后端、数据库、`index.html`、预设弹窗逻辑、其他 theme-select 用途（模型下拉等不受影响）。

## Assumptions & Decisions

* 品牌识别仅依赖 `base_url` 域名，**纯离线**，不发任何网络请求。

* 品牌色用单一品牌代表色 + 白字，不随明/暗主题切换——所选色均在明暗主题下与白字对比充足（饱和度 50%+、明度 40%\~60% 区间）。

* 徽章文本优先级：品牌简称（识别命中）> 预设名称首字符 > host 首字母。

* 触发按钮 label 加小号徽章属于视觉连贯的必要补充（用户选择"圆角方形色块"形态，列表与按钮保持一致）。

* 服务商清单覆盖常见 OpenAI 兼容端点；未收录的新域名自动落入哈希色兜底，无需维护黑名单。

## Verification

1. `cd frontend && npm run build`（Vite 构建）通过，无 import/语法错误。
2. 逻辑走查 `detectBrand` 用例：`https://api.deepseek.com/v1` → DS/#4D6BFE；`https://api.openai.com/v1` → O；`http://localhost:11434/v1` → 本地；`https://custom.example.com/v1` + name=默认配置 → 色板色 + "默"；空 baseURL → 色板色 + name 首字符。
3. 明暗主题各抽查一处（如 default / dark / tokyo-night）：徽章文字对比度、下拉项 active 态下徽章可辨、触发按钮名称 ellipsis 正常。
4. 回归：对话/量化两个模块的预设下拉、管理列表、切换/新增/编辑/删除流程不受影响（仅视觉新增元素）。

