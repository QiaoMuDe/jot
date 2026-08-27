# 密码管理对话框交互打磨：按钮反馈统一 + 详情→编辑丝滑切换

## 摘要

两项纯前端优化：

1. **按钮反馈统一**：为添加/编辑/查看详情三个对话框内的全部按钮补齐「hover 反馈 + :active 点击回弹」，以现有 `.pm-add-btn` 为参考标准。
2. **详情→编辑切换丝滑化**：复用详情对话框已在内存中的 `detailRecord` 数据，同帧完成「关详情→开编辑」，消除因二次异步拉取造成的空窗闪烁。

仅涉及前端两个文件，不改 Go 后端。

## 现状分析

### 问题 1：按钮反馈不统一

文件：`frontend/src/css/components/password-manager.css`

对话框内按钮现状（全部已有 `transition: all 0.15s ease` 与 hover 效果）：

| 按钮                | 类名                | hover            | :active |
| ----------------- | ----------------- | ---------------- | ------- |
| 对话框右上角关闭          | `.pm-close-btn`   | 变色+背景圆           | ❌ 无     |
| 密码输入框 显示/隐藏       | `.pm-pwd-toggle`  | 变色+背景            | ❌ 无     |
| 底栏 取消             | `.pm-btn`         | 变色+边框高亮          | ❌ 无     |
| 底栏 保存             | `.pm-btn.primary` | brightness(1.08) | ❌ 无     |
| 底栏 删除             | `.pm-btn.danger`  | 背景变红底            | ❌ 无     |
| 详情行内小按钮（复制/打开链接等） | `.pm-mini-btn`    | 变色+边框            | ❌ 无     |

参考标准（密码页工具栏新增按钮，L117-132）：

```css
.pm-add-btn:hover {
    filter: brightness(1.08);
    transform: translateY(-1px);
    box-shadow: var(--shadow-dropdown);
}
.pm-add-btn:active {
    transform: scale(0.96);
}
```

即：hover 有位移/亮度反馈，按下有回弹缩放。对话框内按钮恰恰缺这一档体验。

### 问题 2：详情→编辑切换闪烁

文件：`frontend/src/js/password-manager.js`

当前流程（L553-558）：

```js
function editFromDetail() {
    const id = detailRecord?.id;
    if (id == null) return;
    closePmDetailDialog();          // 同步：pmDetailOverlay.style.display='none'，瞬时消失，无退出动画
    openPmEditDialog(Number(id));   // 异步：需 await GetPasswordRecord(id) 网络往返后才 display='flex'
}
```

根因：

* `closePmDetailDialog()` 把详情遮罩**同步置 none**（CSS 只有入场动画，无退出过渡）；

* `openPmEditDialog()` 是 async，要先 `await GetPasswordRecord(id)` 才显示编辑遮罩；

* 于是出现「详情消失 → 短暂露出底层列表 → 新遮罩透明度从 0 淡入」的闪跳。

关键事实：打开详情时已经调过 `GetPasswordRecord` 并把完整记录存进了 `detailRecord`（含明文密码）。模态期间该记录不可能被并发修改（单用户本地应用）。**第二次拉取完全多余**。

## 提出的修改

### 文件 1：`frontend/src/css/components/password-manager.css`（按钮反馈）

在每个类对应 hover 规则之后追加 `:active` 回弹；唯一涉及改动的既有规则是给 `.pm-btn.primary` 的 hover 补上悬浮位移，与其余保持同一手感：

```css
/* 各处追加 */
.pm-close-btn:active { transform: scale(0.90); }      /* 圆形小图标钮，缩放幅度稍大更明显 */
.pm-pwd-toggle:active { transform: scale(0.94); }
.pm-btn:active { transform: scale(0.96); }
.pm-mini-btn:active { transform: scale(0.94); }

/* .pm-btn.primary:hover 改为（对齐 .pm-add-btn 手感） */
.pm-btn.primary:hover {
    filter: brightness(1.08);
    color: #fff;
    transform: translateY(-1px);
}
```

补充一致性（虽非本次明确点名，但属同一视图、代价极低，保证「全页统一」）：批量模式两个按钮也补上回弹——

```css
.pm-batch-toggle-btn:active { transform: scale(0.97); }
.pm-delete-btn:active { transform: scale(0.96); }
```

说明：`:active` 的 transform 会自然覆盖 hover 态 transform（按下瞬间松开浮起效果被压扁是预期行为）；disabled 按钮不触发 `:active`（保存中、批量删除不可用时无需处理）。

### 文件 2：`frontend/src/js/password-manager.js`（丝滑切换）

1. 抽出表单填充函数，去除重复：

```js
/** 将记录填充到编辑表单 */
function fillPmEditForm(rec) {
    pmFieldName.value = rec.name || '';
    pmFieldUsername.value = rec.username || '';
    pmFieldPassword.value = rec.password || '';
    pmFieldUrl.value = rec.url || '';
    pmFieldNote.value = rec.note || '';
}
```

1. `openPmEditDialog(id)` 增加可选参数 `presetRecord`（有则直接填表、跳过网络）：

```js
async function openPmEditDialog(id, presetRecord) {
    /* 重置逻辑不变 */

    if (presetRecord) {
        fillPmEditForm(presetRecord);           // 零延迟路径
    } else if (id != null) {
        try {
            const rec = await window.go.main.App.GetPasswordRecord(id);
            if (!rec) throw new Error('记录不存在');
            fillPmEditForm(rec);                // 原 try 块内的 5 行赋值替换为这一句
        } catch (e) { /* 原错误处理不变 */ }
    }

    pmEditOverlay.style.display = 'flex';       // 不变
    setTimeout(() => pmFieldName.focus(), 50);  // 不变
}
```

1. `editFromDetail()` 改为同帧切换（先捕获 `rec` 再关详情，因为 `closePmDetailDialog` 会把 `detailRecord` 置 null）：

```js
function editFromDetail() {
    const rec = detailRecord;
    if (!rec || rec.id == null) return;
    // 复用详情已在内存的数据，同一渲染帧内完成“关详情→开编辑”，消除空窗闪烁
    closePmDetailDialog();
    openPmEditDialog(Number(rec.id), rec);
}
```

效果：JS 单次执行内先后设置两个 display 值，浏览器合并为一帧绘制——旧遮罩消失与新遮罩出现零间隔，编辑对话框按原有 0.22s 入场动画打开，观感为一个连续过渡，无任何底层内容闪现。

其它调用点（右键菜单「编辑」等仍传裸 id）行为不变，自动走异步拉取旧路径。

## 决策与假设

* **按钮风格基准**：用户授权自选（“你来给统计配置上”），采用仓库内已有的 `.pm-add-btn` 范式做基准，保持视觉语言一致；不同尺寸按钮使用不同缩放幅度（小按钮 0.90/0.94，常规 0.96，大按钮 0.97），避免回弹力度失真。

* **丝滑化方案选型**：候选有二——① 缓存数据同帧切换（选中）；② 给 overlay 加退出过渡动画做交叉淡出。② 存在 DOM 层叠顺序问题（`pmDetailOverlay` 在文档流中位于 `pmEditOverlay` 之后，淡出过程会盖住新对话框，需要临时调 z-index 补丁），且增加总时长；① 零成本根治主要闪烁源。故选 ①。

* **范围决策**：`.pm-batch-toggle-btn` / `.pm-delete-btn` 虽不在对话框内，但一并补齐以保证整个密码管理视图的手感统一（两行 CSS，无风险）。

* 用户此前约束仍然有效：不启动 vite dev server，仅通过构建验证。

## 验证步骤

1. `npm run build`（frontend 目录）通过，无编译错误。
2. 静态核查 diff：

   * 六类对话框按钮 + 批量栏两按钮均具备 hover 与 `:active` 规则；

   * `editFromDetail` 中先捕获 `rec` 再调用 `closePmDetailDialog()`；

   * `openPmEditDialog` 其它调用点签名兼容（第二参数可选）。
3. （用户侧运行时复核清单）编辑/添加/详情三对话框逐个按钮按压有回弹；详情→编辑无黑屏/空窗闪现；右键菜单直接进入编辑仍正常加载表单。

