# 修复 AI 助手标题偶发错位

## 当前状态分析

### 问题现象

AI 助手页面的标题 "AI 助手" 偶发出现水平未居中对齐的情况。

### 根因分析

#### 当前 CSS 实现

```html
<div class="view-header">                     <!-- flex; justify-content: space-between -->
    <button class="back-btn">← 返回</button>   <!-- 左 -->
    <h2 class="view-title">AI 助手</h2>        <!-- position: absolute; left: 50%; transform: translateX(-50%) -->
    <div class="view-controls">...</div>       <!-- 右 -->
</div>
```

标题使用 `position: absolute; left: 50%; transform: translateX(-50%)` 居中。

#### 问题 1：`margin-left: -16px` 未被取消

```css
/* main-content.css - 基础 .view-header */
.view-header {
    margin-left: -16px;  /* 所有 view 都会继承 */
}

/* ai-chat.css - AI 聊天覆盖 */
#viewAiChat .view-header {
    position: relative;
    padding: 24px 16px 0;
    margin-bottom: 24px;
    /* 没有覆盖 margin-left! */
}

#viewAiChat.view {
    padding: 0;  /* 基础 .view 是 padding: 24px 32px */
}
```

其他 view 有 `padding: 24px 32px`，`margin-left: -16px` 用于抵消部分内边距。但 AI chat view 的 `padding: 0`，`margin-left: -16px` 导致 header 视觉上左移 16px。

虽然绝对定位的 `left: 50%` 是基于 view-header 的 padding box 计算，但 flex 布局的 `justify-content: space-between` 只处理 back-btn 和 view-controls 两个非绝对定位子元素。**绝对定位的标题脱离了 flex 流，它的居中计算依赖 view-header 的宽度，而 view-header 的宽度受负 margin 影响在不同渲染时机可能有细微差异。** 这可能是偶发错位的根因。

#### 问题 2：绝对定位脱离 flex 流

标题 `position: absolute` 后，flex 容器只排列两个子元素（back-btn + view-controls）。当页面初始化渲染时序有差异时（如 view 切换过渡、flex 布局正在计算中），绝对定位元素的 `left: 50%` 基于的容器宽度可能与最终稳定宽度不同，导致瞬间错位后未能完全恢复。

### 修复方案

改用 **CSS Grid 三列布局** 替换当前方案，标题作为正常文档流元素，由 grid 保证居中对齐，不依赖绝对定位。

#### 具体改动

**文件：** [ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)

```css
/* 原代码 */
#viewAiChat .view-header {
    position: relative;
    padding: 24px 16px 0;
    margin-bottom: 24px;
}

#viewAiChat .view-title {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    margin: 0;
    white-space: nowrap;
    cursor: pointer;
    transition: color 0.15s ease;
}

/* 改为 */
#viewAiChat .view-header {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    padding: 24px 16px 0;
    margin-bottom: 24px;
    /* 取消从基础 .view-header 继承的负 margin */
    margin-left: 0;
}

#viewAiChat .view-title {
    text-align: center;
    margin: 0;
    white-space: nowrap;
    cursor: pointer;
    transition: color 0.15s ease;
    /* 移除 position: absolute; left: 50%; transform: translateX(-50%) */
}
```

**原方案问题得到修复：**

* 标题不再是绝对定位 → 在正常文档流中由 grid 保证居中

* `margin-left: 0` 显式取消基础 `.view-header` 的负 margin → 避免 AI chat view 的 `padding: 0` 导致 header 左偏

* grid 三列布局：左列 `1fr`（back-btn）、中列 `auto`（标题）、右列 `1fr`（view-controls）→ 无论两侧内容宽度如何变化，标题永远居中

**无需改动：**

* HTML 结构不变

* JS 逻辑不变（标题的 dblclick 事件等不受影响）

### 验证方式

1. 打开 AI 助手页面，检查标题是否完美居中
2. 反复切换视图（从其他 view 切到 AI chat），观察标题是否始终居中
3. 调整窗口宽度，标题应保持居中
4. 检查 `margin-left: -16px` 是否已被成功取消（浏览器 DevTools 检查 computed style）

