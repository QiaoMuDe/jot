# TOC 按钮：空内容/无标题处理 & 后缀切换同步

## 概述

修复两个问题：
1. 正文无内容或没有提取到标题时，点击大纲按钮不展开侧栏，改为通知提示
2. 底部左下角「自定义后缀」修改后，大纲按钮未同步显示/隐藏

---

## 1. 空内容/无标题时点击按钮不展开侧栏

### 现状

`_initTocToggle()` 的 click handler 直接切换 `toc-visible`，不做任何检查。

### 改动

**文件**: `frontend/src/main.js` — `_initTocToggle()` 函数内的 click handler

**逻辑**:
```
点击 TOC 按钮时：
  1. 获取 els.mdRendered 的纯文本内容（trim）
  2. 获取 els.mdRendered 中 h1-h6 的数量
  3. 如果内容为空 → nm.show('正文暂无内容，无法生成目录', 'info')，return
  4. 如果标题数量为 0 → nm.show('当前文档未提取到标题', 'info')，return
  5. 以上均通过 → 正常切换 toc-visible / active 状态
```

### 关键代码位置

- `_initTocToggle` 当前 click handler（约第 3637-3641 行）
- `getEditorContent()`（第 2942-2944 行）— 获取 CM6 编辑器内容
- `_extractHeadingsFromDOM()`（第 3520-3527 行）— 获取渲染后的标题数组
- `nm.show(msg, 'info')` — 通知系统

---

## 2. 底部后缀修改后同步大纲按钮

### 现状

`saveFileExt()` 保存后缀后更新了 `editorTypeToggle` 显示和 `editorModes` 可见性，但**没有**同步 `tocToggleBtn` 的 `show-in-preview` 类。

### 改动

**文件**: `frontend/src/main.js` — `saveFileExt()` 函数

**位置**: 在以下两行之后（约第 3039 行）：
```js
els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
```

**添加的代码**：
```js
// 同步大纲按钮可见性
if (els.tocToggleBtn) {
    els.tocToggleBtn.classList.toggle('show-in-preview', isMd);
}
```

### `toggleFileExt` 状态

已在之前的改动中正确处理（第 3068-3071 行），无需修改。

---

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/main.js` | 修改 `_initTocToggle` click handler + `saveFileExt` 添加 TOC 同步 |

仅一个文件，不涉及 HTML/CSS。

---

## 验证

1. 打开空的 .md 笔记 → 点击大纲按钮 → 不展开侧栏 → 显示"正文暂无内容，无法生成目录"通知
2. 打开有内容但无标题的 .md 笔记 → 点击大纲按钮 → 不展开侧栏 → 显示"当前文档未提取到标题"通知
3. 打开有标题的 .md 笔记 → 点击大纲按钮 → 正常展开侧栏
4. 底部左下角点击后缀 `.md` → 修改为 `.txt` → 保存 → 大纲按钮隐藏
5. 底部左下角点击后缀 `.txt` → 修改为 `.md` → 保存 → 进入预览模式 → 大纲按钮显示
6. 诊断无错误
