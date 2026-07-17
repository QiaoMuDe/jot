# 标签管理卡片重设计划

## 摘要
重新设计设置页中的"标签管理"卡片，提升视觉品质和交互体验，添加微动效和更精致的标签组件。

## 当前状态分析
- **HTML**: [index.html#L633-L651](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L633-L651) — 简单的 `.tag-management` > `.tag-list` + `.tag-add-form`
- **CSS**: [settings-panel.css#L192-L245](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L192-L245) — flexbox 布局，彩色标签 chip + 圆形删除按钮
- **JS**: [main.js#L2946-L2962](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2946-L2962) — `renderTagList()` 渲染纯色标签
- **创建/删除**: [main.js#L1202-L1229](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1202-L1229) + [main.js#L2817-L2830](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2817-L2830)
- **现有主题**: 12 套系统主题均定义 `--tag-delete-bg` / `--tag-delete-hover-bg` CSS 变量
- **问题**: 无动画、视觉简陋、颜色选择器为原生 `<input type="color">`、空状态仅为文字

## 改动文件及具体变更

### 1. `frontend/index.html` — 标签管理卡片 HTML 结构调整

**当前 HTML:**
```html
<div class="tag-management" id="tagManagement">
    <div class="tag-list" id="tagList">
        <!-- 标签列表由 JS 动态渲染 -->
    </div>
    <div class="tag-add-form">
        <input type="text" id="newTagName" class="settings-input" placeholder="新标签名称" />
        <input type="color" id="newTagColor" class="settings-color-input" value="#6366f1" />
        <button class="btn btn-save btn-sm" id="addTagBtn">添加</button>
    </div>
</div>
```

**变更为:**
```html
<div class="tag-management" id="tagManagement">
    <!-- 标签列表 + 空状态容器 -->
    <div class="tag-list" id="tagList"></div>
    <!-- 添加表单 -->
    <div class="tag-add-form">
        <input type="text" id="newTagName" class="tag-input" placeholder="输入标签名称按 Enter 创建" />
        <div class="tag-color-presets" id="tagColorPresets">
            <!-- 9 个预设色圈由 JS 渲染 -->
        </div>
        <button class="tag-add-btn" id="addTagBtn">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            添加
        </button>
    </div>
</div>
```

**说明:**
- 移除原生 `<input type="color">`，替换为 9 个预设颜色圆圈的 `.tag-color-presets`
- 输入框添加 `.tag-input` 类（新样式替代旧 `.settings-input`）
- 按钮改为 `.tag-add-btn`（含 SVG 加号图标）
- 添加表单结构保持不变，便于 JS 复用

### 2. `frontend/src/css/components/settings-panel.css` — 全新标签样式（替换旧样式）

**删除旧选择器群（约 50 行）：**
- `.tag-management`（现有配置）
- `.tag-list`（现有配置）
- `.tag-item`（现有配置）
- `.tag-delete-btn`（现有配置）
- `.tag-delete-btn svg`（现有配置）
- `.tag-delete-btn:hover`（现有配置）
- `.tag-add-form`（现有配置）
- `.settings-color-input`（如有重复，保留全局定义）

**新增 CSS：**

```css
/* =============================================
   Section O: Tag Management (Redesigned)
   ============================================= */

.tag-management {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── 标签列表 ── */
.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  min-height: 36px;
}

/* ── 标签芯片 ── */
.tag-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px 7px 16px;
  border-radius: 20px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #fff;
  cursor: default;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1),
              box-shadow 0.15s ease;
  transform-origin: center;
  animation: tagEnter 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  box-shadow: 0 1px 3px rgba(0,0,0,0.15);
  position: relative;
  user-select: none;
}

.tag-item:hover {
  transform: translateY(-1px) scale(1.03);
  box-shadow: 0 3px 10px rgba(0,0,0,0.18);
}

/* 入口动画（JS 应用 stagger 延迟） */
@keyframes tagEnter {
  from {
    opacity: 0;
    transform: scale(0.7) translateY(6px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

/* ── 删除按钮（悬浮在 tag 右侧） ── */
.tag-delete-btn {
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: rgba(255,255,255,0.25);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  flex-shrink: 0;
  transition: background 0.15s, transform 0.2s ease;
  opacity: 0.7;
}

.tag-delete-btn svg {
  width: 10px;
  height: 10px;
}

.tag-delete-btn:hover {
  background: rgba(255,255,255,0.45);
  transform: scale(1.2) rotate(90deg);
  opacity: 1;
}

.tag-delete-btn:active {
  transform: scale(0.9);
}

/* ── 删除动画 ── */
.tag-item.removing {
  animation: tagRemove 0.25s cubic-bezier(0.5, 0, 0.75, 0) forwards;
  pointer-events: none;
}

@keyframes tagRemove {
  to {
    opacity: 0;
    transform: scale(0.5) translateY(-6px);
  }
}

/* ── 添加表单 ── */
.tag-add-form {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.tag-input {
  flex: 1;
  min-width: 160px;
  padding: 9px 14px;
  border: 1.5px solid var(--border);
  border-radius: 20px;
  font-size: 0.8125rem;
  color: var(--text-primary);
  outline: none;
  font-family: inherit;
  background: var(--input-bg);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.tag-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(from var(--accent) r g b / 0.12);
}

.tag-input::placeholder {
  color: var(--text-muted);
  font-weight: 400;
}

/* ── 预设色圈 ── */
.tag-color-presets {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-shrink: 0;
}

.tag-color-dot {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1),
              border-color 0.15s,
              box-shadow 0.15s;
  flex-shrink: 0;
  box-sizing: border-box;
}

.tag-color-dot:hover {
  transform: scale(1.2);
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.tag-color-dot.active {
  border-color: var(--text-primary);
  transform: scale(1.15);
  box-shadow: 0 0 0 2px var(--card-bg), 0 0 0 4px var(--text-primary);
}

/* ── 添加按钮 ── */
.tag-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 18px;
  border: none;
  border-radius: 20px;
  background: var(--accent);
  color: #fff;
  font-size: 0.8125rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: transform 0.15s cubic-bezier(0.34, 1.56, 0.64, 1),
              box-shadow 0.15s,
              background 0.15s;
  flex-shrink: 0;
  white-space: nowrap;
}

.tag-add-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 3px 10px rgba(from var(--accent) r g b / 0.3);
}

.tag-add-btn:active {
  transform: scale(0.96);
}

/* ── 空状态 ── */
.tag-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px 20px;
  gap: 8px;
  color: var(--text-muted);
  font-size: 0.8125rem;
  animation: tagEmptyFadeIn 0.4s ease both;
  width: 100%;
  border: 1.5px dashed var(--border);
  border-radius: 16px;
}

.tag-empty-icon {
  width: 32px;
  height: 32px;
  opacity: 0.5;
}

@keyframes tagEmptyFadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}
```

### 3. `frontend/src/main.js` — JS 修改（约 40 行）

**`renderTagList()` 重构：**
- 每个标签项添加 `style="animation-delay: ${index * 40}ms"` 实现 stagger 入场
- 删除时先加 `removing` class，再用 `setTimeout` 实际删除
- 空状态改为 `tag-empty` 结构（含 SVG 图标 + 文案）
- 删除按钮 SVG 保留 `SVGS.windowClose`

**添加预设色圈渲染逻辑：**
- 在 `renderTagList()` 同级或新增 `renderTagColorPresets()` 函数
- 9 个预设色值：`#6366f1`, `#ec4899`, `#f43f5e`, `#f97316`, `#eab308`, `#22c55e`, `#14b8a6`, `#06b6d4`, `#8b5cf6`
- 初始化时设置 `els.newTagColor` 为 value，并对应当前选中项加 `active` class

**`renderTagList` 调用点确认：**
- `loadTags()` 中已调用 `renderTagList()`，无需改动
- `window.loadTags` 已暴露全局，无需改动

**事件绑定：**
- 新增 `els.tagColorPresets` DOM 引用
- 预设色圈点击事件：切换 `.active` 并同步到 `els.newTagColor.value`
- `els.addTagBtn` 和 `els.newTagName keydown Enter` 逻辑不变

### 4. `frontend/src/css/variables.css` — 保留现有 CSS 变量

- `--tag-delete-bg` / `--tag-delete-hover-bg` 在新设计中不再使用（删除按钮改用 `rgba(255,255,255,0.25)` + `:hover` 状态），考虑保留旧变量以兼容其他引用或移除
- **决定**: 保留变量，不做改动 —— 无副作用且避免影响其他潜在引用

## 设计决策说明

| 决策 | 选项 | 理由 |
|------|------|------|
| **美学方向** | 圆润、柔和、卡片式 | 与项目整体卡片风格一致；标签使用 pill 形状更现代、视觉更统一 |
| **颜色选择** | 9 预设色圈替代原生 `<input type="color">` | 操作更直观（点选而非打开取色器），视觉更精致，减少用户操作步骤 |
| **动画** | 入场 stagger + hover scale + 删除 fade-out + 按钮 spring | 符合项目 `var(--anim-easing-spring)` 传统，微动效提升感知质量 |
| **删除按钮** | 悬浮白色半透明圆 + hover 旋转 90° | 比纯色 X 更精致，旋转动效增加交互趣味 |
| **空状态** | 虚线边框容器 + SVG 图标 + 文案 | 视觉占位引导用户操作，比纯文字更专业 |

## 验证步骤
1. `npm run build` 构建前端，确认无编译错误
2. 打开设置页 → 标签管理卡片，观察：
   - 标签芯片入场动画（stagger 延迟）
   - 点击预设色圈切换颜色
   - 创建标签后新标签入场动画
   - 删除标签后移除动画
   - 空状态显示虚线边框容器
3. 切换 12 个系统主题，确认标签样式正确适配
4. 在编辑器模态框中确认标签选择器（`renderTagSelector`）不受影响
