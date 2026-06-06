# 笔记单条导出为 Markdown

## 概述

在右键菜单中添加「导出」选项，将单条笔记导出为 `.md` 文件。文件名由笔记标题自动生成（特殊符号/空白替换为下划线），通过系统保存对话框选择保存路径。

## 当前状态分析

- **右键菜单**：HTML 结构在 `index.html` 中，菜单项使用 `data-action` 标识；`main.js` 中 `handleContextAction(action)` 用 `switch/case` 分发操作
- **导出模式**：已有 `app.go:ExportDataWithDialog()` 调用 `runtime.SaveFileDialog` 的完整模式可供参考
- **Note 模型**：`id`, `title`, `content` 字段可直接使用

## 修改方案

### 1. `index.html` — 右键菜单新增「导出」项

**位置**: `d:\峡谷\Dev\本地项目\jot\frontend\index.html` 右键菜单区域

在「复制」和「查看」之间插入一个「导出」菜单项：

```html
<div class="context-menu-item" data-action="export">导出</div>
```

菜单完整顺序：复制 → 导出 → 分隔线 → 查看 → 编辑 → 分隔线 → 置顶 → 分隔线 → 删除

### 2. `app.go` — 新增后端绑定方法 `ExportNoteAsMarkdown`

**位置**: `d:\峡谷\Dev\本地项目\jot\app.go`

新增方法，流程：
1. 通过 ID 查找笔记
2. 用笔记标题生成安全的默认文件名（`sanitizeFilename(title) + ".md"`）
3. 弹出系统保存对话框
4. 写入 `# 标题\n\n内容` 格式的 Markdown 文件
5. 返回导出结果字符串

**文件名清理规则**（内置函数 `sanitizeFilename`）：
- 空白和特殊符号（`\ / : * ? " < > |`）替换为下划线
- 连续多个替换符合并为单个下划线
- 首尾下划线去掉
- 标题为空或清理后为空 → fallback 为 `untitled`

```go
func (a *App) ExportNoteAsMarkdown(id uint) (string, error) {
    note, err := a.noteService.GetByID(id)
    if err != nil {
        return "", fmt.Errorf("笔记不存在: %w", err)
    }

    defaultName := sanitizeFilename(note.Title) + ".md"
    filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
        Title:           "导出笔记为 Markdown",
        DefaultFilename: defaultName,
        Filters: []runtime.FileFilter{
            {DisplayName: "Markdown (*.md)", Pattern: "*.md"},
        },
    })
    if err != nil {
        return "", err
    }
    if filePath == "" {
        return "已取消", nil
    }

    content := fmt.Sprintf("# %s\n\n%s", note.Title, note.Content)
    if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
        return "", fmt.Errorf("写入文件失败: %w", err)
    }

    return "导出成功：" + filePath, nil
}
```

### 3. `main.js` — 添加导出处理逻辑

**位置**: `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`

#### 3a. `handleContextAction` 添加 `export` 分支

```javascript
case 'export':
    window.exportNote(id);
    break;
```

#### 3b. 新增 `exportNote` 函数

```javascript
window.exportNote = async function (id) {
    try {
        const result = await window.go.main.App.ExportNoteAsMarkdown(id);
        if (result && result !== '已取消') {
            nm.show(result, 'success');
        }
    } catch (err) {
        nm.show('导出失败：' + err, 'error');
    }
};
```

### 4. 构建验证

- `npx vite build` — 前端构建通过
- `golangci-lint run ./...` — 后端 lint 通过

## 假设与决策

| 决策 | 原因 |
|------|------|
| 用 Go 端调 SaveFileDialog | Wails v2 的 runtime 函数只能从 Go 端调用 |
| 前端只用 `window.go.main.App.ExportNoteAsMarkdown(id)` | 保持与现有导出模式一致 |
| 文件名由标题生成，不加时间戳 | 因为保存对话框已经允许用户手动修改文件名 |
| 文件头加 `# 标题` 行 | 导出的 `.md` 文件打开时直接显示标题，内容完整 |
| 不添加 YAML frontmatter | 保持文件纯净，任何编辑器都能直接阅读 |

## 验证步骤

1. 右键任意笔记 → 菜单显示「导出」
2. 点击「导出」→ 弹出系统保存对话框，默认文件名如 `笔记标题.md`
3. 选择路径确认 → 文件写入成功 → 右上角提示「导出成功」
4. 打开 `.md` 文件确认内容格式正确
5. 标题含特殊符号的笔记 → 文件名中特殊符号被替换为下划线
6. 空标题笔记 → 默认文件名 `untitled.md`
