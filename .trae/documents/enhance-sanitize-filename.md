# 增强导出文件名清洗逻辑

## Summary

采用**白名单（保留）策略**重构 `sanitizeFilename` 函数：仅保留英文、中文、数字和安全符号（`- _ . ( ) [ ] { } , ; ! ? + = ~ @ # &`），其余字符（emoji、特殊符号、不可见字符等）全部移除，生成更干净、跨平台兼容的导出文件名。

## Current State Analysis

当前 `sanitizeFilename`（[app.go#L559-L576](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L559-L576)）：

```go
func sanitizeFilename(title string) string {
    name := strings.TrimSpace(title)
    if name == "" {
        return "untitled"
    }
    // 仅替换 Windows 非法字符和空白
    re := regexp.MustCompile(`[\\/:*?"<>|\s]+`)
    name = re.ReplaceAllString(name, "_")
    // 合并连续下划线
    re2 := regexp.MustCompile(`_+`)
    name = re2.ReplaceAllString(name, "_")
    // 去掉首尾下划线
    name = strings.Trim(name, "_")
    if name == "" {
        return "untitled"
    }
    return name
}
```

**问题**：emoji（📝🔥🎉）、特殊符号（©™®）、零宽字符、控制字符等不会被处理，原样保留在文件名中。

## Proposed Changes

### 1. 重构 `sanitizeFilename`（[app.go#L559-L576](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L559-L576)）

**白名单策略**：先筛选保留字符，再执行原有清洗流程。

保留的字符范围：

| 类别 | Unicode 范围/字符 |
|------|------------------|
| 英文字母 | `a-z A-Z` |
| 数字 | `0-9` |
| 中文字符 | `\u4e00-\u9fff`（CJK 统一表意文字） |
| 中文标点 | `\u3000-\u303F`（含全角空格、顿号、书名号等） |
| 安全符号 | `- _ . ( ) [ ] { } , ; ! ? + = ~ @ # & 空格` |

流程：

```
原始标题
  → step1: 移除所有不在白名单的字符（emoji、特殊符号、控制字符等全部干掉）
  → step2: 走现有逻辑（空白/Win非法字符 → _，合并_，trim首尾_，空值回退）
```

```go
func sanitizeFilename(title string) string {
    // step1: 白名单过滤 — 只保留英文、中文、数字、安全符号
    // 保留: a-z A-Z 0-9 中文 中文标点 以及 -_.()[]{};,!?+=~@#&空格
    re := regexp.MustCompile(`[^a-zA-Z0-9\u4e00-\u9fff\u3000-\u303F().\[\]{},;!?+=~@#&\-_\s]`)
    name = re.ReplaceAllString(title, "")

    // step2: 原有清洗（TrimSpace + Win非法替换 + 合并_ + trim首尾_ + 空值回退）
    name = strings.TrimSpace(name)
    if name == "" {
        return "untitled"
    }
    re2 := regexp.MustCompile(`[\\/:*?"<>|\s]+`)
    name = re2.ReplaceAllString(name, "_")
    re3 := regexp.MustCompile(`_+`)
    name = re3.ReplaceAllString(name, "_")
    name = strings.Trim(name, "_")
    if name == "" {
        return "untitled"
    }
    return name
}
```

### 2. 修改 `ExportNoteAsMarkdown` 的弹窗标题（可选）

`"导出笔记为 Markdown"` → `"导出笔记"`（更通用，因为不限于 Markdown）

### 3. 更新注释

更新函数注释，说明白名单策略和保留字符范围。

## Assumptions & Decisions

- `sanitizeFilename` 只在 `ExportNoteAsMarkdown` 一处使用，改动不影响其他逻辑
- 白名单策略天然处理 emoji、特殊符号、控制字符、零宽字符等一切非白名单字符
- 考虑到本应用用户主要使用中英文标题，白名单范围足够覆盖实际场景
- `\u3000-\u303F` 包含中文标点（顿号、引号、书名号等），`\u4e00-\u9fff` 覆盖大部分常用汉字
- 短横线 `-` 在字符类中需要转义或放末尾，下划线 `_` 和点 `.` 不需要转义

## Verification

- `go build` 编译通过（Go 的 `regexp` 支持 `\uXXXX` 语法）
- 手动测试含 emoji 标题导出：
  - `"📝 学习笔记 🔥"` → `"学习笔记"`
  - `"我的笔记（重要）©2024"` → `"我的笔记（重要）"`
  - `"a-b_c.d+e~f@g#h&i!j?k;l,m"` → `"a-b_c.d+e~f@g#h&i!j?k;l,m"`（安全符号保留）
