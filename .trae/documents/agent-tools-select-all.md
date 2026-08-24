# Agent工具管理 - 批量全选功能实现计划

## Summary

为设置页「对话与搜索」面板中的 Agent 工具管理功能添加批量全选/全不选能力，用户可以一键启用或禁用所有工具，无需逐个勾选。

## Current State Analysis

### 现有实现

* **界面**：点击"Agent工具"按钮展开管理面板，列出所有工具（13个内置工具 + MCP工具）

* **交互**：每个工具有独立的 checkbox，勾选/取消立即生效并保存

* **数据**：`agentToolsDisabled` 数组存储禁用工具名，`agentToolsMeta` 存储工具元信息

* **Header结构**：标题"Agent 工具" + "关闭"按钮（flex布局，两端对齐）

### 问题

* 批量操作需要逐个勾选，效率低下

* 无"全选"机制

## Proposed Changes

### 文件1: `frontend/src/main.js`

#### 修改1: 新增变量 `selectAllCheckbox`

**位置**: 第9121行附近，在 `agentToolsMgrContainer` 变量定义后
**内容**: 新增 `let agentToolsSelectAllCheckbox = null;`

#### 修改2: 新增函数 `updateSelectAllCheckboxState()`

**位置**: 第9188行后（`updateAgentToolsButtonText` 函数之后）
**功能**: 同步全选checkbox的状态（checked/unchecked/indeterminate）
**逻辑**:

```javascript
function updateSelectAllCheckboxState() {
    if (!agentToolsSelectAllCheckbox) return;
    const enabledCount = agentToolsMeta.filter(tool =>
        agentToolsDisabled.indexOf(tool.Name) === -1
    ).length;
    if (enabledCount === agentToolsMeta.length) {
        agentToolsSelectAllCheckbox.checked = true;
        agentToolsSelectAllCheckbox.indeterminate = false;
    } else if (enabledCount === 0) {
        agentToolsSelectAllCheckbox.checked = false;
        agentToolsSelectAllCheckbox.indeterminate = false;
    } else {
        agentToolsSelectAllCheckbox.indeterminate = true;
        agentToolsSelectAllCheckbox.checked = false;
    }
}
```

#### 修改3: 新增函数 `toggleSelectAllTools()`

**位置**: 紧接 `updateSelectAllCheckboxState()` 函数后
**功能**: 批量启用/禁用所有工具
**逻辑**:

```javascript
function toggleSelectAllTools() {
    // 根据当前状态决定是启用还是禁用
    // indeterminate或checked → 禁用全部；unchecked → 启用全部
    const shouldEnable = agentToolsSelectAllCheckbox.indeterminate || !agentToolsSelectAllCheckbox.checked;

    agentToolsMeta.forEach(tool => {
        const isEnabled = agentToolsDisabled.indexOf(tool.Name) === -1;
        if (isEnabled === shouldEnable) return; // 状态未变，跳过

        if (shouldEnable) {
            // 启用：从禁用列表移除
            const idx = agentToolsDisabled.indexOf(tool.Name);
            if (idx !== -1) agentToolsDisabled.splice(idx, 1);
            // 记录变更
            if (agentToolsChanges.enabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.enabled.push(tool.Name);
            }
            // 清除相反方向的变更记录
            const deIdx = agentToolsChanges.disabled.indexOf(tool.Name);
            if (deIdx !== -1) agentToolsChanges.disabled.splice(deIdx, 1);
        } else {
            // 禁用：加入禁用列表
            if (agentToolsDisabled.indexOf(tool.Name) === -1) {
                agentToolsDisabled.push(tool.Name);
            }
            // 记录变更
            if (agentToolsChanges.disabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.disabled.push(tool.Name);
            }
            // 清除相反方向的变更记录
            const enIdx = agentToolsChanges.enabled.indexOf(tool.Name);
            if (enIdx !== -1) agentToolsChanges.enabled.splice(enIdx, 1);
        }
    });

    // 更新所有子checkbox的UI状态
    document.querySelectorAll('.ai-agent-tools-item input[type="checkbox"]').forEach(cb => {
        cb.checked = shouldEnable;
    });

    updateAgentToolsButtonText();
    updateSelectAllCheckboxState();
    saveSettings();
}
```

#### 修改4: 修改 `renderAgentToolsMgrList()` 函数中的header创建部分

**位置**: 第9220-9232行
**修改内容**: 在标题前添加全选checkbox

原代码:

```javascript
const header = document.createElement('div');
header.className = 'agent-tools-mgr-header';
const title = document.createElement('span');
title.className = 'agent-tools-mgr-title';
title.textContent = 'Agent 工具';
const closeBtn = document.createElement('button');
closeBtn.className = 'btn btn-sm btn-secondary';
closeBtn.textContent = '关闭';
closeBtn.addEventListener('click', closeAgentToolsMgrList);
header.appendChild(title);
header.appendChild(closeBtn);
```

修改为:

```javascript
const header = document.createElement('div');
header.className = 'agent-tools-mgr-header';

// 左侧容器：全选checkbox + 标题
const headerLeft = document.createElement('div');
headerLeft.className = 'agent-tools-mgr-header-left';

// 全选checkbox
agentToolsSelectAllCheckbox = document.createElement('input');
agentToolsSelectAllCheckbox.type = 'checkbox';
agentToolsSelectAllCheckbox.className = 'agent-tools-mgr-select-all';
agentToolsSelectAllCheckbox.addEventListener('change', toggleSelectAllTools);
updateSelectAllCheckboxState(); // 初始化状态

const title = document.createElement('span');
title.className = 'agent-tools-mgr-title';
title.textContent = 'Agent 工具';

headerLeft.appendChild(agentToolsSelectAllCheckbox);
headerLeft.appendChild(title);

const closeBtn = document.createElement('button');
closeBtn.className = 'btn btn-sm btn-secondary';
closeBtn.textContent = '关闭';
closeBtn.addEventListener('click', closeAgentToolsMgrList);
header.appendChild(headerLeft);
header.appendChild(closeBtn);
```

#### 修改5: 在 `createAgentToolRow()` 函数的checkbox change事件中调用状态同步

**位置**: 第9279行，在 `updateAgentToolsButtonText()` 调用后
**新增**: `updateSelectAllCheckboxState();`

***

### 文件2: `frontend/src/css/components/settings-panel.css`

#### 修改1: 调整 `.agent-tools-mgr-header` 样式

**位置**: 第1631-1636行
**修改为**:

```css
.agent-tools-mgr-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  gap: 8px;
}
```

#### 修改2: 新增 `.agent-tools-mgr-header-left` 样式

**位置**: 第1636行后
**内容**:

```css
.agent-tools-mgr-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
```

#### 修改3: 新增 `.agent-tools-mgr-select-all` 样式

**位置**: 第1642行后（`.agent-tools-mgr-title` 样式后）
**内容**:

```css
.agent-tools-mgr-select-all {
  flex-shrink: 0;
  accent-color: var(--accent);
  cursor: pointer;
}
```

***

## Assumptions & Decisions

1. **全选checkbox位置**: 放在标题左侧，与标题在同一行，符合常见UI模式（如文件选择器）
2. **indeterminate状态**: 当部分工具启用时显示半选状态，提供清晰的状态反馈
3. **点击行为**: indeterminate或checked状态点击→禁用全部；unchecked状态点击→启用全部
4. **变更记录**: 批量操作时正确记录每个工具的变更，保持与单个操作一致的行为
5. **无需动画**: 批量操作时直接切换checkbox状态，无需逐个动画

## Verification Steps

1. **功能验证**:

   * 打开设置页 → 对话与搜索 → 点击"Agent工具"按钮

   * 验证全选checkbox显示在标题左侧

   * 验证初始状态：所有工具启用时全选checkbox为checked

   * 点击全选checkbox → 验证所有工具被禁用

   * 再次点击全选checkbox → 验证所有工具被启用

   * 手动取消部分工具勾选 → 验证全选checkbox变为indeterminate状态

   * 关闭面板 → 验证变更摘要正确显示

2. **状态同步验证**:

   * 批量禁用后，逐个启用一个工具 → 验证全选checkbox变为indeterminate

   * 批量启用后，逐个禁用最后一个工具 → 验证全选checkbox变为unchecked

3. **边界情况验证**:

   * 工具列表为空时 → 全选checkbox状态正常

   * 只有一个工具时 → 全选checkbox行为正常

## Implementation Order

1. 修改CSS文件（添加新样式）
2. 修改JS文件（添加变量和函数）
3. 修改JS文件（修改renderAgentToolsMgrList函数）
4. 修改JS文件（修改createAgentToolRow函数）
5. 测试验证

