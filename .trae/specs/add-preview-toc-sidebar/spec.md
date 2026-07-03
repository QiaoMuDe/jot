# Markdown 预览大纲侧栏 (TOC Sidebar) Spec

## Why

在笔记的查看/编辑/新建页面中，Markdown 预览模式下缺少标题导航功能。用户阅读长文档时无法快速定位到某个章节，需要手动滚动查找，体验不佳。

## What Changes

- 在编辑器预览模式下，于预览区左侧增加一个可折叠的大纲侧栏
- 自动提取 Markdown 内容中的标题（H1~H6），生成层级目录
- 点击目录项平滑滚动到对应标题位置
- 滚动预览内容时高亮当前所在章节
- 仅对 `.md` 后缀的笔记生效
- 侧栏支持展开/折叠切换

## Impact

- Affected specs: 编辑器模块、Markdown 预览模块
- Affected code:
  - `frontend/index.html` — 新增 TOC 侧栏 DOM 结构
  - `frontend/src/main.js` — 标题提取、TOC 渲染、滚动同步逻辑
  - `frontend/src/js/preview-worker.js` — 标题信息提取
  - `frontend/src/css/components/editor.css` — TOC 侧栏样式

## ADDED Requirements

### Requirement: 标题提取

The system SHALL 在 Markdown 预览渲染时，自动提取文档中的标题信息。

#### Scenario: 空文档
- **WHEN** 文档内容为空
- **THEN** TOC 侧栏隐藏，不显示目录

#### Scenario: 有标题的文档
- **WHEN** 文档包含 H1~H6 标题
- **THEN** 提取所有标题及对应的层级（depth）和文本内容

#### Scenario: 无标题的文档
- **WHEN** 文档不包含任何标题
- **THEN** TOC 侧栏隐藏或显示"无标题"提示

### Requirement: TOC 侧栏展示

The system SHALL 在预览模式下展示大纲侧栏。

#### Scenario: 进入预览模式
- **WHEN** 用户切换到预览模式（编辑模式中点击"预览" 或 从卡片网格打开 Markdown 笔记进入查看模式）
- **THEN** 左侧显示 TOC 侧栏，右侧显示渲染后的 Markdown 内容

#### Scenario: 侧栏折叠
- **WHEN** 用户点击侧栏的折叠按钮
- **THEN** 侧栏收起，仅保留一个窄条触发按钮

#### Scenario: 侧栏展开
- **WHEN** 用户点击窄条展开按钮
- **THEN** 侧栏重新展开

#### Scenario: 非 Markdown 笔记
- **WHEN** 笔记后缀不是 `.md`
- **THEN** TOC 侧栏不显示

### Requirement: 标题层级缩进

The system SHALL 在目录中按标题层级展示缩进关系。

- H1 → 无缩进，加粗
- H2 → 缩进 1 级
- H3 → 缩进 2 级
- H4+ → 统一缩进 3 级（避免过度缩进）
- 每个目录项显示标题文本，点击可滚动到对应位置

### Requirement: 点击跳转

The system SHALL 支持点击目录项跳转到对应标题位置。

#### Scenario: 点击目录项
- **WHEN** 用户点击 TOC 中的某个标题项
- **THEN** 预览区平滑滚动到该标题位置，标题在视口中处于靠上位置

#### Scenario: 再次点击
- **WHEN** 用户点击同一个标题项
- **THEN** 再次触发滚动到该位置

### Requirement: 滚动高亮

The system SHALL 在用户滚动预览内容时，自动高亮当前对应的目录项。

#### Scenario: 滚动内容
- **WHEN** 用户在预览区滚动
- **THEN** 自动识别当前视口顶部的最近标题，在 TOC 中高亮对应的目录项
- **THEN** 高亮项自动滚动到 TOC 可见区域内（若 TOC 可滚动）

#### Scenario: 快速滚动
- **WHEN** 用户快速滚动后停止
- **THEN** 高亮在滚动停止后更新（防抖 100ms）

### Requirement: 标题锚点 ID

The system SHALL 为渲染后的 HTML 标题元素生成唯一 ID，以便锚点跳转。

- ID 生成规则：取标题文本，转为小写，非单词字符替换为连字符 `-`
- 遇到重复 ID 时，自动追加数字后缀（`-1`, `-2`）

### Requirement: 持久化折叠状态

The system SHALL 记住用户的侧栏折叠/展开偏好。

- 使用 `localStorage` 存储折叠状态
- 下次打开预览时恢复上次状态

## MODIFIED Requirements

暂无

## REMOVED Requirements

暂无
