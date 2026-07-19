# 品牌名点击改为返回首页，帮助参考中新增关于入口

## 概要

将品牌名"Jot"的点击行为从"打开关于页面"改为"返回笔记首页"，与其他页面返回逻辑保持一致；同时在"帮助参考"子菜单中新增"关于"入口以保留关于页面的访问途径。

## 现状分析

| 元素                     | 当前行为                                        | 所在位置                                                                 |
| ---------------------- | ------------------------------------------- | -------------------------------------------------------------------- |
| `.brand-name`（Jot文字）   | `stopPropagation(); showAbout()` 打开关于浮层     | [main.js:5458-5462](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) |
| `.topbar-brand`（整个品牌区） | `switchView('grid')` 返回首页（不含 `loadNotes()`） | [main.js:5240-5244](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) |
| 各视图返回按钮                | `switchView('grid'); loadNotes()`           | [main.js:595-625](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 等 |
| 帮助参考子菜单                | "快捷键说明" + "MD 语法"                           | [index.html:85-92](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html)   |
| 菜单点击事件分发               | if-else 链式判断 `data-action`                  | [main.js:5189-5220](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) |

两层事件同时绑定在品牌区，`.brand-name` 用 `stopPropagation()` 覆盖了父级 `.topbar-brand`。

## 修改计划（4 处改动）

### 1. 移除 `.brand-name` 点击事件

* **文件**: `frontend/src/main.js`

* **位置**: 第 5458-5462 行

* **操作**: 删除以下代码块

  ```js
  // 关于页面
  document.querySelector('.brand-name').addEventListener('click', (e) => {
      e.stopPropagation();
      showAbout();
  });
  ```

* **效果**: 点击".brand-name"文字不再被拦截，事件冒泡到 `.topbar-brand`，统一执行返回首页逻辑。

### 2. 对齐 `.topbar-brand` 返回行为

* **文件**: `frontend/src/main.js`

* **位置**: 第 5243 行

* **操作**: 在 `switchView('grid');` 后添加 `loadNotes();`

  ```js
  // 修改前
  switchView('grid');

  // 修改后
  switchView('grid');
  loadNotes();
  ```

* **效果**: 与其他视图返回按钮完全一致（`switchView('grid'); loadNotes()`）。

### 3. 帮助参考子菜单新增"关于"项

* **文件**: `frontend/index.html`

* **位置**: 第 90-91 行之间（MD 语法项之后）

* **操作**: 添加以下 DOM 元素

  ```html
  <div class="dropdown-item" data-action="about"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>关于</div>
  ```

* **Svg 图标说明**: 使用 info/help 风格图标（圆圈 + i 符号），与"快捷键说明"（键盘图标）、"MD 语法"（文档图标）风格统一。

### 4. 菜单点击事件增加 `about` 分发

* **文件**: `frontend/src/main.js`

* **位置**: 第 5217 行之后（`case 'help'` 结束之后）

* **操作**: 添加 else-if 分支

  ```js
  } else if (item.dataset.action === 'about') {
      showAbout();
  }
  ```

* **效果**: 点击"关于"菜单项时调用已有的 `showAbout()` 函数打开关于页面浮层。

## 实施后行为变化

| 操作             | 修改前                     | 修改后                        |
| -------------- | ----------------------- | -------------------------- |
| 点击品牌文字"Jot"    | 打开关于页面                  | 返回笔记首页（与父级行为一致）            |
| 点击品牌区域空白处      | 返回笔记首页（无 `loadNotes()`） | 返回笔记首页（含 `loadNotes()` 刷新） |
| 菜单 > 帮助参考 > 关于 | 不存在                     | 打开关于页面浮层                   |
| 所有其他行为         | 不变                      | 不变                         |

## 验证步骤

1. 点击品牌名"Jot" → 应返回笔记首页，笔记列表刷新
2. 点击品牌区域空白处 → 同上
3. 打开更多菜单 → 帮助参考 → 关于 → 应弹出关于页面浮层
4. 关于页面浮层关闭按钮及背景点击关闭正常
5. Escape 键在有关于浮层时关闭浮层，否则返回首页（已有逻辑不变）

