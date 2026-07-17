# 锁屏密码状态显示精简方案

## 当前问题

设置页锁屏密码行当前同时显示状态标签（"未设置"/"已启用 ✓"）和按钮（"设置密码"/"修改密码"），信息冗余。用户要求：
1. **去掉** `pwdStatusLabel` 状态标签
2. 按钮文本单独承担状态指示：无密码时显示"设置密码"，有密码时显示"修改密码"
3. 模态框内部逻辑按此状态正确显示/隐藏旧密码输入框

## 改动文件及内容

### 1. HTML — `frontend/index.html`（第 732 行）

移除 `<span class="pwd-status-label" id="pwdStatusLabel">未设置</span>`

```diff
 <div class="ai-setting-control" style="flex:none;margin-left:auto;">
-    <span class="pwd-status-label" id="pwdStatusLabel">未设置</span>
     <button class="btn btn-sm btn-save" id="pwdChangeBtn" type="button">设置密码</button>
 </div>
```

### 2. CSS — `settings-panel.css`（第 1193-1200 行）

移除 `.pwd-status-label` 样式规则（约 7 行）

### 3. JS — `main.js`（4 处改动）

#### 3a. 模态框打开（第 2284 行）
`isPasswordSet` 的判断来源从 `pwdStatusLabel.textContent` 改为 `pwdChangeBtn.textContent`：
```diff
- const isPasswordSet = document.getElementById('pwdStatusLabel').textContent.includes('已启用');
+ const isPasswordSet = pwdChangeBtn.textContent.includes('修改密码');
```

#### 3b. 表单验证（第 2309 行）
同样修改：
```diff
- const isPasswordSet = document.getElementById('pwdStatusLabel').textContent.includes('已启用');
+ const isPasswordSet = pwdChangeBtn.textContent.includes('修改密码');
```

#### 3c. 保存成功回调（第 2368 行）
移除 `pwdStatusLabel.textContent` 赋值：
```diff
- document.getElementById('pwdStatusLabel').textContent = '已启用 ✓';
  document.getElementById('pwdChangeBtn').textContent = '修改密码';
```

#### 3d. 设置加载（第 8041-8051 行）
移除 `statusLabel` 引用，保留 `changeBtn` 更新：
```diff
- const statusLabel = document.getElementById('pwdStatusLabel');
  const changeBtn = document.getElementById('pwdChangeBtn');
  ...
- if (statusLabel && changeBtn) {
+ if (changeBtn) {
      const hasPassword = cfg.screen_lock_password && cfg.screen_lock_password !== '';
-     statusLabel.textContent = hasPassword ? '已启用 ✓' : '未设置';
      changeBtn.textContent = hasPassword ? '修改密码' : '设置密码';
  }
```

## 验证

- Vite 构建无错误
- 开启开关 → 显示"设置密码"按钮，无状态标签
- 设置密码后 → 按钮变为"修改密码"
- 点击"设置密码"/"修改密码" → 模态框正确显示/隐藏旧密码输入框
- 重新加载设置页 → 按钮文本与密码存在状态一致
