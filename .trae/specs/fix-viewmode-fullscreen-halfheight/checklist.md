# Checklist

- [x] **根因分析确认** — `spec.md` 中的根因分析（查看模式未设置 `data-mode="edit"` 导致 CSS `!important` 不生效）与实际代码路径一致。

- [x] **修复代码实现** — `openEditor()` 中 view mode + text note 分支添加了 `els.editorOverlay.dataset.mode = 'edit'`。

- [ ] **场景 1：新建纯文本笔记 → 全屏** — 编辑器区域撑满全高，无回归。（需手动测试）

- [ ] **场景 2：编辑已有纯文本笔记 → 全屏** — 编辑器区域撑满全高，无回归。（需手动测试）

- [ ] **场景 3：查看纯文本笔记（只读）→ 全屏** — 编辑器区域撑满全高（核心修复场景）。（需手动测试）

- [ ] **场景 4：Markdown 笔记查看模式 → 全屏** — 预览区域撑满全高，无回归。（需手动测试）

- [x] **编辑器关闭后重新打开** — `closeEditor()` 已在 line 2958 执行 `delete els.editorOverlay.dataset.mode`，`data-mode` 状态正确重置。
