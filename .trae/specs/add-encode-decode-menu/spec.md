# 新增编码解码菜单 Spec

## Why
编辑器操作菜单目前有"格式化"、"文本转换"、"文本清理"三个分组，缺少编码解码类操作。在笔记中经常需要处理 Base64、URL 编码、HTML 实体编码等场景，增加该分组可以方便用户快速进行编码解码操作。

## What Changes
- 在 `EDITOR_ACTIONS` 数组中新增「编码解码」分组，含 3 个子分组共 6 个操作项
  - **Base64**：编码 / 解码（`btoa`/`atob`，浏览器内置 API）
  - **URL**：编码 / 解码（`encodeURIComponent`/`decodeURIComponent`，浏览器内置 API）
  - **HTML**：编码 / 解码（手工替换 + DOMParser，浏览器内置 API）
- 全部零依赖，无需安装任何 npm 包

## Impact
- Affected code: `frontend/src/js/editor-actions.js` — 在 `EDITOR_ACTIONS` 数组末尾追加 6 个新操作条目
- 无需新增文件、无需安装依赖、无需修改 CSS

## ADDED Requirements

### Requirement: 编码解码分组
系统 SHALL 在操作菜单中提供"编码解码"分组，含 3 个子分组（Base64、URL、HTML），每个子分组含编码/解码两个操作项。

#### Scenario: Base64 编码
- **WHEN** 用户对文本 `Hello World` 执行「Base64 编码」
- **THEN** 结果输出 `SGVsbG8gV29ybGQ=`

#### Scenario: Base64 解码
- **WHEN** 用户对文本 `SGVsbG8gV29ybGQ=` 执行「Base64 解码」
- **THEN** 结果输出 `Hello World`

#### Scenario: Base64 解码失败
- **WHEN** 用户对非法 Base64 字符串 `!!!` 执行「Base64 解码」
- **THEN** 提示错误"不是合法的 Base64"

#### Scenario: URL 编码
- **WHEN** 用户对文本 `hello world` 执行「URL 编码」
- **THEN** 结果输出 `hello%20world`

#### Scenario: URL 解码
- **WHEN** 用户对文本 `hello%20world` 执行「URL 解码」
- **THEN** 结果输出 `hello world`

#### Scenario: URL 解码失败
- **WHEN** 用户对非法 URL 编码字符串执行「URL 解码」
- **THEN** 提示错误"不是合法的 URL 编码"

#### Scenario: HTML 编码
- **WHEN** 用户对文本 `<script>alert("xss")</script>` 执行「HTML 编码」
- **THEN** 结果输出 `&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;`

#### Scenario: HTML 解码
- **WHEN** 用户对文本 `&lt;b&gt;bold&lt;/b&gt;` 执行「HTML 解码」
- **THEN** 结果输出 `<b>bold</b>`

## MODIFIED Requirements
无。

## REMOVED Requirements
无。