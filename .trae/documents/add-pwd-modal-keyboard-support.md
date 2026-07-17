# 密码弹窗键盘快捷键支持方案

## 概述

为锁屏密码修改弹窗（`#pwdModal`）添加 ESC 键关闭和 Enter 键提交的支持，与其他模态对话框保持一致。

## 当前状态

* 密码弹窗 `#pwdModal` 目前仅支持点击关闭按钮、取消按钮、遮罩背景三种关闭方式

* 三个密码输入框无任何键盘事件监听

* 全局 ESC 处理器（`handleKeyboardNavigation`）未检查密码弹窗状态

## 变更计划

### 1. ESC 键支持 — 修改全局 `handleKeyboardNavigation`

**文件**: `frontend/src/main.js`

**位置**: 第 5505-5506 行（预设弹窗关闭之后、关于页面检查之前）

**变更**: 插入 `#pwdModal` 可见性检查，如果弹窗打开则关闭并 return

**原因**: `#pwdModal` 是模态遮罩层，与 `#presetModalOverlay` 同类，应放在同一优先级区域。ESC 关闭弹窗后不应继续执行后续的视图导航逻辑。

**具体代码**:

```javascript
// 密码弹窗打开时关闭它
const pwdModal = document.getElementById('pwdModal');
if (pwdModal && pwdModal.style.display === 'flex') {
    pwdModal.classList.remove('visible');
    pwdModal.style.display = 'none';
    return;
}
```

### 2. Enter 键支持 — 在密码弹窗 JS 块中增加键盘监听

**文件**: `frontend/src/main.js`

**位置**: 第 2350-2352 行（密码可见切换逻辑之后、保存按钮点击事件之前）

**变更**: 为三个密码输入框统一注册 keydown 事件，按下 Enter 且保存按钮可用时触发点击

**原因**: 符合用户预期——输入完成后按回车直接提交，与搜索弹窗、文件扩展名弹窗等保持一致。

**具体代码**:

```javascript
// Enter 键提交
const pwdInputs = [document.getElementById('pwdOldInput'), document.getElementById('pwdNewInput'), document.getElementById('pwdConfirmInput')];
pwdInputs.forEach(input => {
    if (!input) return;
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            const saveBtn = document.getElementById('pwdModalSaveBtn');
            if (!saveBtn.disabled) saveBtn.click();
        }
    });
});
```

## 两个变更各自独立，没有先后依赖关系，可同时实施。

## 验证方式

1. 打开密码弹窗 → 按 ESC → 弹窗关闭
2. 打开密码弹窗（修改密码模式）→ 填写完三个输入框 → 按 Enter → 保存提交
3. 打开密码弹窗 → 不填内容 → 按 Enter → 不触发保存（按钮为 disabled 状态）

