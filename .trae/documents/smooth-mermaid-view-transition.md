# Mermaid 源码/渲染视图切换动画优化

## 总结

将 Mermaid 代码块在源码视图和渲染视图之间的切换从粗暴的 `display: none/block` 改为带 `opacity` 和 `transform` 过渡的平滑交叉淡入淡出，并锁住容器高度防止布局跳动。

## 当前状态分析

### 抖动根因

当前切换逻辑（[toggleMermaidView, main.js L3880-3898](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)）：

```js
// 切到渲染
rendered.style.display = '';    // 瞬间出现
pre.style.display = 'none';     // 源码瞬间消失

// 切回源码
rendered.style.display = 'none'; // 渲染图瞬间消失
pre.style.display = '';          // 源码瞬间出现
```

两个问题：
1. **`display: none/block` 不可过渡** — 浏览器直接从 DOM 中移除/添加元素，无任何动画
2. **`pre` 和 `.mermaid-rendered` 高度不同** — 容器 `.pre-wrapper` 在切换时高度突变

### 涉及文件

- `frontend/src/main.js` — `toggleMermaidView()`、`renderSingleMermaid()`
- `frontend/src/css/components/editor.css` — `.mermaid-rendered` 样式（L1311-1326）

## 变更方案

### 核心理念

用一个状态类 `.show-rendered` 控制 `.pre-wrapper`，CSS 过渡处理所有视觉变化，JS 仅负责：
1. 切换状态类
2. 在过渡期间锁定容器高度防止跳动

### CSS 变更（[editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css)）

```css
/* container：过渡期间不能剪切溢出内容 */
.pre-wrapper.has-mermaid {
    transition: height 0.25s ease;
}

/* 源码默认可见 */
.pre-wrapper.has-mermaid pre {
    opacity: 1;
    transform: translateY(0);
    transition: opacity 0.25s ease, transform 0.25s ease;
}

/* 渲染图默认隐藏（偏下位置，为滑入做准备） */
.pre-wrapper.has-mermaid .mermaid-rendered {
    opacity: 0;
    pointer-events: none;
    transform: translateY(8px);
    transition: opacity 0.25s ease, transform 0.25s ease;
}

/* 渲染视图状态：源码淡出上滑，渲染图淡入归位 */
.pre-wrapper.has-mermaid.show-rendered pre {
    opacity: 0;
    transform: translateY(-8px);
    pointer-events: none;
}
.pre-wrapper.has-mermaid.show-rendered .mermaid-rendered {
    opacity: 1;
    transform: translateY(0);
    pointer-events: auto;
}
```

**过渡方向逻辑**：
- 源码→渲染：源码 `translateY(0→-8px)` 上滑淡出，渲染图 `translateY(8px→0)` 上滑淡入
- 渲染→源码：渲染图 `translateY(0→8px)` 下滑淡出，源码 `translateY(-8px→0)` 下滑淡入

**为什么不用 `position: absolute` 防高度跳动**：改用 JS 锁高度方式更灵活，不依赖固定高度。

### JS 变更（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)）

#### 1. 重写 `toggleMermaidView`

```js
async function toggleMermaidView(btn) {
    const wrapper = btn.parentNode;
    const pre = wrapper.querySelector('pre');
    const rendered = wrapper.querySelector('.mermaid-rendered');
    if (!pre || !rendered) return;

    if (wrapper.classList.contains('show-rendered')) {
        // 切换到源码视图
        lockHeight(wrapper);
        wrapper.classList.remove('show-rendered');
        btn.textContent = '渲染';
        btn.title = '渲染为 Mermaid 图表';
        releaseHeight(wrapper);
    } else {
        // 切换到渲染视图：立即更新按钮文字，异步渲染
        btn.textContent = '源码';
        btn.title = '显示源码';
        lockHeight(wrapper);

        const mermaidCode = pre.dataset.mermaidCode;
        if (mermaidCode) {
            mermaid.initialize({ theme: getMermaidTheme() });
            const id = 'mermaid-' + Date.now() + '-' + Math.random().toString(36).slice(2, 8);
            try {
                const { svg } = await mermaid.render(id, mermaidCode);
                rendered.innerHTML = svg;
            } catch (err) {
                console.warn('Mermaid render error:', err);
                rendered.innerHTML = `<div class="mermaid-error">Mermaid 渲染失败：${err.message}</div>`;
            }
        }

        wrapper.classList.add('show-rendered');
        releaseHeight(wrapper);
    }
}
```

#### 2. 新增辅助函数

```js
function lockHeight(el) {
    el.style.height = el.offsetHeight + 'px';
}

function releaseHeight(el) {
    // 过渡结束后释放固定高度，让容器自适应
    setTimeout(() => { el.style.height = ''; }, 280);
}
```

#### 3. 删除/弃用 `renderSingleMermaid`

内联到 `toggleMermaidView` 中（不再单独调用），原函数保留但不再使用。

注意：原本 `renderSingleMermaid` 中直接操作 `style.display` 的方式移除。

## 涉及文件

| 文件 | 变更 | 说明 |
|---|---|---|
| `editor.css` | 新增过渡规则 ~20 行 | `pre` 和 `.mermaid-rendered` 的 opacity/transform 过渡，容器 height 过渡 |
| `main.js` | 重写 `toggleMermaidView` ~40 行 | 改用 class 切换 + 高度锁定，渲染逻辑内联 |

## 动画时间线

**源码 → 渲染（~300ms 总时长）**：
```
0ms:     锁高度 → 开始渲染 SVG（异步）
~50ms:   SVG 渲染完成
50ms:    添加 .show-rendered → pre 开始 0.25s 淡出上滑，rendered 开始 0.25s 淡入上滑
300ms:   释放高度 → 容器自适应
```

**渲染 → 源码（~250ms 总时长）**：
```
0ms:     锁高度 → 移除 .show-rendered → 过渡开始
250ms:   释放高度 → 容器自适应
```

## 验证步骤

1. `cd frontend && npx vite build` — 构建无错误
2. 运行应用，点击"渲染"按钮，观察源码到渲染图的过渡是否丝滑
3. 点击"源码"按钮，观察渲染图到源码的过渡是否丝滑
4. 验证渲染失败时错误提示也能平滑过渡
5. 确认复制按钮「已复制」隐藏渲染按钮的动画不受影响
