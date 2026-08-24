# 修复AI助手会话侧边栏重命名输入框高度问题

## 摘要

AI助手模块会话侧边栏中，点击重命名时出现的 `contentEditable` 输入框高度太低（仅由文字行高撑起），需要增加内边距使其与会话条目（`.ai-session-item`）的视觉高度一致。

## 当前状态分析

### 问题根因

* 会话条目 `.ai-session-item` 有 `padding: 9px 4px`，整体高度较大

* 重命名时 `.ai-session-item-title` 获得焦点，样式仅添加 `padding: 0 4px`

* 导致输入框内部高度只有文字行高，远低于条目高度，视觉上很矮

### 涉及文件

* `frontend/src/css/components/ai-chat.css` — 第 2180-2185 行，`.ai-session-item-title:focus` 样式

## 变更方案

### 修改文件

**`frontend/src/css/components/ai-chat.css`** 第 2180-2185 行

将当前的 focus 样式：

```css
.ai-session-item-title:focus {
    outline: none;
    background: var(--input-bg);
    border-radius: 3px;
    padding: 0 4px;
}
```

修改为：

```css
.ai-session-item-title:focus {
    outline: none;
    background: var(--input-bg);
    border-radius: 3px;
    padding: 5px 4px;
}
```

**说明**：

* 将 focus 时的上下内边距从 `0` 增加到 `5px`

* 这会使输入框高度接近 `.ai-session-item` 的整体高度（`padding: 9px 4px`）

* 输入框文字的垂直居中由父容器 `.ai-session-item` 的 `align-items: center` 保证

## 验证步骤

1. 启动前端开发服务器
2. 打开 AI 助手页面，确保有至少一个会话
3. 双击会话标题或右键选择"重命名"
4. 验证输入框高度与会话条目高度一致
5. 验证输入框内文字仍然垂直居中
6. 验证编辑完成后恢复正常状态

