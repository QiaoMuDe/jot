# 编辑器拖拽图片上传 Spec

## Why

目前用户只能通过 Ctrl+V 粘贴图片到编辑器。拖拽图片文件从文件管理器到编辑器是同等自然的操作，且与粘贴互补——粘贴适合截图（剪贴板已有）、拖拽适合已有图片文件。拖拽上传可以复用粘贴上传的完整链路（SaveImage + 插入 Markdown），实现成本极低。

## What Changes

- **CM6 编辑器新增 `dragover`/`drop` 事件监听** — 检测拖入的图片文件，阻止浏览器默认行为（避免页面跳转）
- **拖拽逻辑复用粘贴核心** — 抽取 `uploadImage(file)` 函数，paste 和 drop 共用：FileReader → base64 → SaveImage → 光标处插入 `![](/images/uuid_name.ext)`
- **拖拽区域视觉反馈** — 拖拽悬停时编辑器区域显示半透明边框提示，松开后消失

## Impact

- Affected code: [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)（核心修改）、[style.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/style.css)（拖拽悬停样式）

## ADDED Requirements

### Requirement: 拖拽图片上传

系统 SHALL 支持用户将图片文件从文件管理器拖拽到 CodeMirror 编辑器中完成上传插入。

#### Scenario: 拖拽图片到编辑器
- **WHEN** 用户从文件管理器拖拽一个或多个图片文件到 CM6 编辑器区域
- **THEN** 编辑器区域显示拖拽悬停视觉反馈（半透明虚线边框）
- **THEN** 释放后，每个图片文件依次上传：`FileReader.readAsDataURL` → `SaveImage` → 光标处插入 `![](/images/uuid_name.ext)`
- **THEN** 光标定位到最后一个插入的图片语法末尾

#### Scenario: 拖拽非图片文件
- **WHEN** 用户拖拽非图片文件到编辑器
- **THEN** 不做任何处理，保持编辑器原有行为（浏览器默认行为被阻止，但文件被忽略）

#### Scenario: 非 .md 笔记
- **WHEN** 用户在非 `.md` 后缀笔记的编辑器中拖拽图片
- **THEN** 不做任何处理（与粘贴逻辑一致）

### Requirement: 视觉反馈

系统 SHALL 在图片被拖入编辑器区域时提供视觉反馈。

#### Scenario: 拖拽悬停
- **WHEN** 图片文件被拖入编辑器区域且尚未释放
- **THEN** 编辑器容器添加 `dragover` 类，显示 `.cm-editor.dragover { outline: 2px dashed var(--accent-color); }` 样式
- **WHEN** 拖拽离开编辑器区域或释放后
- **THEN** 移除 `dragover` 类

### Requirement: 代码重构

系统 SHALL 将粘贴和拖拽的图片上传逻辑抽取为共享函数。

#### Scenario: 提取 uploadImage 函数
- **WHEN** 粘贴或拖拽触发图片上传
- **THEN** 调用同一 `uploadImage(file, cmEditor)` 函数
- **THEN** 函数内部完成：FileReader 读取 → base64 → SaveImage → 光标处插入 Markdown

## MODIFIED Requirements

无。

## REMOVED Requirements

无。
