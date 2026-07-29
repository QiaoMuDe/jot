# 修复 MD 语法页面 TOC 锚点链接被全局拦截器误认为外链的问题

## 总结

MD 语法页面 TOC 目录项的 `<a href="#md-ref-card--headings">` 锚点链接，被全局链接拦截器（BrowserOpenURL）误拦截，导致点击时既执行 TOC 的 `scrollIntoView` 平滑滚动，又通过 `window.runtime.BrowserOpenURL` 在系统浏览器中打开完整 URL。

## 当前状态分析

```javascript
// frontend/src/main.js:5913-5920
document.addEventListener('click', function (e) {
    const link = e.target.closest('a');
    if (link && link.href && !link.href.startsWith('#') && !link.href.startsWith('javascript:')) {
        e.preventDefault();
        window.runtime.BrowserOpenURL(link.href);
    }
});
```

**根因**：在 DOM API 中，`link.href` 属性始终返回**已解析的绝对 URL**（如 `https://localhost:34115/#md-ref-card--headings`），而非 HTML 属性中的原始值 `#md-ref-card--headings`。因此 `!link.href.startsWith('#')` 永远为 `true`，任何 `<a>` 标签都进入外链处理分支。

**影响范围**：
- MD 语法页面的 TOC 锚点链接（10 个，`#md-ref-card--headings` 等）
- TOC 的 `scrollIntoView` 仍会执行（TOC 专属 handler 先于全局 handler 绑定？——实际是执行顺序的问题，两个 handler 都会执行）
- 其他使用 `#` 锚点的 `<a>` 标签（如注释、文档内的锚点导航）

**正确的边界判断**应使用 `link.getAttribute('href')` 获取原始属性值，或使用 `link.hash` 检测是否包含片段标识符。

## 修改方案

**文件**：`frontend/src/main.js` 第 5916 行

**修改**：将条件判断中的 `link.href.startsWith('#')` 替换为对原始属性值 `link.getAttribute('href')` 的检查。

修改前：
```javascript
if (link && link.href && !link.href.startsWith('#') && !link.href.startsWith('javascript:')) {
```

修改后：
```javascript
if (link && link.href && !link.getAttribute('href').startsWith('#') && !link.href.startsWith('javascript:')) {
```

**为什么这样改**：
- `link.getAttribute('href')` 返回 HTML 中写入的原始值（如 `#md-ref-card--headings`），而非绝对 URL
- `javascript:` 伪协议检查同样适用原始值或全 URL，但 `link.href.startsWith('javascript:')` 在 WebView2 中也可能有问题，不过目前没有 JS 协议链接的用例，可以保留原判断或同样改为 `getAttribute`
- 不影响外链的正常拦截——外链的 `href` 属性值以 `http://` 或 `https://` 开头，`getAttribute('href')` 同样返回绝对 URL

**备选方案**：也可以使用 `link.hash` 判断：
```javascript
if (link && link.href && !link.hash && !link.href.startsWith('javascript:')) {
```
`link.hash` 返回空字符串（无片段时）或 `#xxx`（有片段时）。不过 `getAttribute('href')` 方案更直观统一。

**推荐方案**：使用 `getAttribute('href')`，因为它语义清晰，且与 `javascript:` 判断保持一致的检查方式。

## 验证步骤

1. 启动应用，进入 MD 语法页面
2. 点击任意 TOC 项（如「⌗ 标题」），确认：
   - 页面平滑滚动到对应卡片 ✔️
   - 系统浏览器不会自动打开 ✔️
3. 进入设置页或笔记详情中带有外链的页面，点击外链，确认：
   - 系统浏览器正常打开外链 ✔️
4. 确认其他 `#` 锚点链接行为不受影响
