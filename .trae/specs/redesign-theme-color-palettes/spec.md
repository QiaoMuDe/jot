# 主题配色方案重构 Spec

## Why

当前 14 套系统主题的配色存在以下问题：
- 亮色主题的页面背景色（`--bg`）普遍偏平、缺乏层次感和情绪氛围
- 部分主题的卡片背景（`--card-bg`）与页面背景差异过小（如 gruvbox-light 仅 ∆=2），视觉层次不足
- 护眼主题（eye-protection）豆沙绿饱和度太高（~45%），整个页面被染成绿色，视觉上反而疲劳
- 暗色主题（dark）使用纯黑 `#0D0D0D`，在非 OLED 屏幕上发灰，阅读舒适度差
- 代码块背景色（`--bg-secondary`）与代码内容的对比关系不够精致
- light 主题过于平庸（`#FAFAFA` + `#FFFFFF`），没有个性
- 部分主题的 accent 色与其他颜色搭配不协调

需要一次全面的配色方案重构，让每套主题都有清晰的情绪方向和视觉品质。

## What Changes

- **所有 14 套主题的 `--bg` / `--card-bg` / `--bg-secondary` 重新设计**
- **护眼主题彻底重做**：从高饱和豆沙绿改为低饱和度柔和米绿
- **dark 主题底色调整**：从纯黑改为深灰 `#1A1A1E` 提升阅读舒适度
- **增加各主题之间的层次差异**：card-bg 与 bg 保持有意义的分离度（∆ ≥ 10-15）
- **优化代码块背景**：通过独立的色值让代码块区域在页面中有"嵌入感"
- **light 主题注入个性**：从纯白改为极浅蓝灰调
- **其他亮色主题微调色相/饱和度**：每个主题的底色调向更有辨识度的方向偏移
- **所有主题的 `--border` / `--divider` / `--hover-bg` 同步调整**：使其与新的底色协调

### 不涉及更改
- 不新增/删除主题（保持 14 套）
- 不改动主题名称和 ID
- 不改动 CSS 变量名（仅改变值）
- 不改动语义色（success/warning/error/info）的基础结构（仅微调部分值）
- 不改动 Go 后端代码
- 不改动 JS 逻辑

## Impact

- Affected specs: 主题系统
- Affected code:
  - `frontend/src/css/variables.css` — 修改所有 `[data-theme]` 块的配色值

## 各主题配色重构方案

### 1. default（温暖极简 → 陈年纸本）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#F7F5F0` | `#F2EDE3` | 更暖更深的米白底，模拟陈年纸本质感 |
| `--card-bg` | `#FFFFFF` | `#FCF9F2` | 暖白卡片，与背景拉开差距 |
| `--bg-secondary` | `#EDE9E0` | `#EBE3D3` | 略深暖灰，带黄色倾向，代码块区域有"嵌入感" |
| `--border` | `#E5E0D8` | `#E3DBCB` | 匹配新底色 |
| `--divider` | `#EBE6DE` | `#E8E0D2` | 匹配新底色 |
| `--hover-bg` | `#F3F0EB` | `#F0E9DC` | 匹配新底色 |
| `--input-bg` | `#F3F1ED` | `#F0EBE0` | 匹配新底色 |
| `--topbar-bg` | `#FFFFFF` | `#FCF9F2` | 与 card-bg 统一 |
| `--text-tertiary` | `#C5C0B8` | `#C8C0B0` | 略暖 |
| `--scrollbar-thumb` | `rgba(0,0,0,0.35)` | `rgba(60,50,40,0.25)` | 暖调的滚动条 |
| `--scrollbar-thumb-hover` | `rgba(0,0,0,0.55)` | `rgba(60,50,40,0.40)` | 暖调的滚动条 |

### 2. light（纯净明亮 → 雪光）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#FAFAFA` | `#F4F4F7` | 极浅蓝灰底，有"雪光白"的冷感，告别纯灰 |
| `--card-bg` | `#FFFFFF` | `#FFFFFF` | 保持纯白 |
| `--bg-secondary` | `#F0F0F0` | `#EBEBF0` | 浅蓝灰代码块，与纯白卡片区分 |
| `--border` | `#EEEEEE` | `#E6E6EC` | 略冷的分隔线 |
| `--divider` | `#F0F0F0` | `#EBEBF0` | 匹配新底色 |
| `--hover-bg` | `#F5F5F5` | `#EEEFF3` | 匹配新底色 |
| `--input-bg` | `#F5F5F5` | `#EEEFF3` | 匹配新底色 |
| `--text-secondary` | `#8C8C8C` | `#909098` | 微调 |
| `--text-tertiary` | `#D0D0D0` | `#C8C8D0` | 微调 |
| `--scrollbar-thumb` | `rgba(0,0,0,0.32)` | `rgba(0,0,0,0.22)` | 更柔和 |

### 3. nord（北极 → 极光黎明）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#ECEFF4` | `#E4E9F0` | 更深的极地蓝灰，加强"北欧"辨识度 |
| `--card-bg` | `#FFFFFF` | `#FAFBFD` | 冷白卡片 |
| `--bg-secondary` | `#E2E7EE` | `#DCE1EA` | 冷灰蓝辅助背景 |
| `--border` | `#D8DEE9` | `#D0D7E4` | 更深边框 |
| `--divider` | `#E2E6EF` | `#D8DEE9` | 匹配 |
| `--hover-bg` | `#E8ECF3` | `#DEE3EC` | 略冷 |
| `--input-bg` | `#E8ECF3` | `#DEE3EC` | 匹配 |
| `--topbar-bg` | `#FFFFFF` | `#FAFBFD` | 与 card-bg 统一 |
| `--text-secondary` | `#6B7394` | `#6D7590` | 微调 |
| `--text-tertiary` | `#B0B7C8` | `#A8B0C4` | 微调 |

### 4. gruvbox-light（旧纸 → 泛黄报纸）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#FBF1C7` | `#F0E8C9` | 降低饱和度~30%，更像泛黄旧报纸而非亮黄 |
| `--card-bg` | `#F9F5D7` | `#F8F2D8` | 比背景略亮，拉开 ∆≈12 |
| `--bg-secondary` | `#EBDBB2` | `#E6DDB8` | 黄褐色代码块 |
| `--border` | `#D5C4A1` | `#D0C4A5` | 匹配新底色 |
| `--divider` | `#E0CFB0` | `#DBCFB2` | 匹配 |
| `--hover-bg` | `#EBDBB2` | `#E4D9B8` | 匹配 |
| `--input-bg` | `#EBDBB2` | `#E4D9B8` | 匹配 |
| `--topbar-bg` | `#F9F5D7` | `#F8F2D8` | 与 card-bg 统一 |
| `--text-muted` | `#A89984` | `#A09480` | 微调 |
| `--text-tertiary` | `#C4B89E` | `#C0B498` | 微调 |

### 5. one-dark-pro（暗夜）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#282C34` | 保持 | 经典配色不变 |
| `--card-bg` | `#2C313A` | `#2C313A` | 保持 |
| `--bg-secondary` | `#2C313A` | `#21252B` | 使用 Atom 编辑器原生底色，与其他面板区分 |
| `--border` | `#3E4452` | `#383D4A` | 微调 |
| `--divider` | `#3E4452` | `#383D4A` | 微调 |
| `--hover-bg` | `#3E4452` | `#383D4A` | 微调 |
| `--input-bg` | `#3E4452` | `#383D4A` | 微调 |

### 6. quiet-light（静谧 → 灰玫瑰）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#F5F0E8` | `#F0EAEC` | 灰玫瑰米色，比当前更有"静谧"气质 |
| `--card-bg` | `#FCF8F2` | `#F9F4F4` | 与其搭配的暖粉白 |
| `--bg-secondary` | `#F0EAE2` | `#EBE2E6` | 带紫调的浅灰粉 |
| `--border` | `#E0D8E5` | `#DCD2DB` | 匹配新底色 |
| `--divider` | `#EAE2EC` | `#E4DAE2` | 匹配 |
| `--hover-bg` | `#E8E2F4` | `#E5DBEC` | 紫调 hover |
| `--input-bg` | `#FFFFFF` | `#F6F0F0` | 暖粉输入框 |
| `--topbar-bg` | `#FCF8F2` | `#F9F4F4` | 与 card-bg 统一 |
| `--text-tertiary` | `#C8BEC8` | `#C8BDCB` | 微调 |

### 7. ysgrifennwr（暖笺 → 羊皮纸）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#F5EDDA` | `#F0E5D0` | 略深，更有信纸/羊皮纸质感 |
| `--card-bg` | `#FAF4E3` | `#F8F0DB` | 暖白偏米 |
| `--bg-secondary` | `#F0E4D0` | `#EADCC4` | 暖褐色辅助背景 |
| `--border` | `#E5D6C2` | `#E0D0BB` | 匹配 |
| `--divider` | `#EDE0D0` | `#E8D8C6` | 匹配 |
| `--hover-bg` | `#F0E4D0` | `#EADCC4` | 匹配 |
| `--input-bg` | `#FAF4E3` | `#F0E5D0` | 匹配 |
| `--topbar-bg` | `#FAF4E3` | `#F8F0DB` | 与 card-bg 统一 |

### 8. tokyo-night（夜幕）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#1A1B26` | 保持 | 经典配色 |
| `--card-bg` | `#24283B` | `#1F2335` | 更接近 Tokyo Night 原版配色 |
| `--bg-secondary` | `#15161F` | `#13141F` | 更深代码块背景 |
| `--border` | `#2F354A` | `#2A2F44` | 微调 |
| `--divider` | `#2F354A` | `#2A2F44` | 微调 |
| `--hover-bg` | `#292E42` | `#262B3E` | 微调 |
| `--input-bg` | `#292E42` | `#262B3E` | 微调 |

### 9. eye-protection（护眼 → 柔和米绿）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#C7EDCC` | `#E4ECD9` | 极低饱和度米绿，护眼且美观（饱和度从~45% 降至~15%） |
| `--card-bg` | `#D8F5DD` | `#EFF3E6` | 更亮的近白色卡片 |
| `--bg-secondary` | `#BFE5C8` | `#DEE6CD` | 浅灰绿代码块 |
| `--border` | `#B8DCC0` | `#D5DEC4` | 匹配新底色 |
| `--divider` | `#C4E4CB` | `#DCE3CE` | 匹配 |
| `--hover-bg` | `#D0ECD5` | `#E3EBD6` | 匹配 |
| `--input-bg` | `#D0ECD5` | `#E3EBD6` | 匹配 |
| `--topbar-bg` | `#D8F5DD` | `#EFF3E6` | 与 card-bg 统一 |
| `--accent` | `#2E7D32` | `#4A8C5C` | 柔和橄榄绿，替换原偏深的绿色 |
| `--accent-dark` | `#1B5E20` | `#3A704A` | 匹配新 accent |
| `--accent-light` | `#C8E6C9` | `#D8E8D0` | 匹配 |
| `--accent-lighter` | `#E8F5E9` | `#EAF0E0` | 匹配 |
| `--text-primary` | `#2C4A3E` | `#384A3E` | 深绿灰文字，不刺眼 |
| `--text-secondary` | `#5B8A6E` | `#658570` | 柔和绿灰 |
| `--text-muted` | `#8AB89A` | `#8EA890` | 柔和绿灰 |
| `--text-tertiary` | `#A8D0B2` | `#A8BCA5` | 匹配 |
| `--success` | `#2E7D32` | `#4A8C5C` | 同 accent |
| `--overlay-bg` | `rgba(44,74,62,0.35)` | `rgba(56,74,62,0.30)` | 匹配新底色 |
| `--toast-bg` | `#2C4A3E` | `#384A3E` | 匹配 text-primary |
| `--toast-text` | `#C7EDCC` | `#E4ECD9` | 匹配 bg |

### 10. dark（深色 → 柔黑）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#0D0D0D` | `#18181C` | 深灰底色，避免纯黑的晕影效应，阅读更舒适 |
| `--card-bg` | `#1A1A1A` | `#222226` | 略亮，与背景形成微妙分层 |
| `--bg-secondary` | `#181818` | `#1C1C22` | 代码块深背景 |
| `--border` | `#2A2A2A` | `#2D2D33` | 微调 |
| `--divider` | `#2A2A2A` | `#2D2D33` | 微调 |
| `--hover-bg` | `#252525` | `#2A2A30` | 微调 |
| `--input-bg` | `#252525` | `#2A2A30` | 微调 |
| `--topbar-bg` | `#141414` | `#1C1C20` | 匹配新 card-bg |
| `--text-primary` | `#E8E8E8` | `#E4E4E8` | 柔白文字，减轻眩光 |
| `--text-secondary` | `#9E9E9E` | `#9C9CA8` | 微调 |
| `--text-muted` | `#6B6B6B` | `#6E6E7A` | 微调 |
| `--text-tertiary` | `#505050` | `#555560` | 微调 |

### 11. dracula（德古拉 → 哥特紫）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#282A36` | `#232530` | 略深，接近 Dracula Pro 风格 |
| `--bg-secondary` | `#21222C` | `#1E1F2B` | 更深代码块背景 |
| `--accent-light` | `#44475A` | `#3D3E52` | 紫色调更深 |
| `--accent-lighter` | `#3A3C4E` | `#35364A` | 匹配 |
| `--hover-bg` | `#44475A` | `#3E4054` | 匹配 |
| `--input-bg` | `#44475A` | `#3E4054` | 匹配 |
| `--text-tertiary` | `#44475A` | `#404254` | 微调 |

### 12. catppuccin-latte（暖咖 → 热拿铁）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#EFF1F5` | `#EDE7E5` | 暖粉米白，还原 Catppuccin Latte 真正的暖粉气质 |
| `--card-bg` | `#FFFFFF` | `#F8F2EF` | 暖粉白卡片 |
| `--bg-secondary` | `#E6E9EF` | `#E6DEDC` | 暖粉色辅助背景 |
| `--border` | `#DCE0E8` | `#DBD1D0` | 匹配新底色 |
| `--divider` | `#E6E9EF` | `#E4DCDA` | 匹配 |
| `--hover-bg` | `#E6E9EF` | `#E4DCDA` | 匹配 |
| `--input-bg` | `#E6E9EF` | `#E4DCDA` | 匹配 |
| `--topbar-bg` | `#FFFFFF` | `#F8F2EF` | 与 card-bg 统一 |
| `--text-primary` | `#4C4F69` | `#504C60` | 微调 |
| `--text-secondary` | `#8C8FA7` | `#908B9E` | 微调 |
| `--text-muted` | `#BCC0CC` | `#BEB8C4` | 微调 |
| `--text-tertiary` | `#CCD0DA` | `#D0CAD4` | 微调 |

### 13. alice（爱丽丝 → 暖米蓝）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#F9F5E8` | `#F3EDDE` | 更有深度的暖米色 |
| `--card-bg` | `#FFFCF5` | `#FBF5E8` | 暖白偏米 |
| `--bg-secondary` | `#F0E8DC` | `#EAE0D2` | 暖米辅助背景 |
| `--border` | `#E8DDD0` | `#E2D6C8` | 匹配 |
| `--divider` | `#F0E8DC` | `#EAE0D2` | 匹配 |
| `--hover-bg` | `#F5EDE0` | `#EDE4D4` | 匹配 |
| `--input-bg` | `#F5EDE0` | `#EDE4D4` | 匹配 |
| `--topbar-bg` | `#FFFCF5` | `#FBF5E8` | 与 card-bg 统一 |
| `--text-tertiary` | `#D5C8B8` | `#D0C4B2` | 微调 |

### 14. lightmind（山林 → 竹纸）

| 变量 | 当前值 | 新值 | 理由 |
|------|--------|------|------|
| `--bg` | `#F2EFE6` | `#EAECE0` | 更清雅的灰绿米 |
| `--card-bg` | `#FAF7EF` | `#F5F4E8` | 灰绿白卡片 |
| `--bg-secondary` | `#EDE8DC` | `#E4E6D4` | 灰绿辅助背景 |
| `--accent` | `#4A7C59` | `#3D8C55` | 更鲜明的森林绿 |
| `--accent-dark` | `#2F5A40` | `#2D6E42` | 匹配新 accent |
| `--border` | `#DED8CC` | `#D8D4C4` | 匹配 |
| `--divider` | `#E8E2D6` | `#E2DECC` | 匹配 |
| `--hover-bg` | `#EDE8DC` | `#E4E6D4` | 匹配 |
| `--input-bg` | `#EDE8DC` | `#E4E6D4` | 匹配 |
| `--topbar-bg` | `#FAF7EF` | `#F5F4E8` | 与 card-bg 统一 |
| `--text-tertiary` | `#C8C0B4` | `#C4C0B0` | 微调 |

## ADDED Requirements

无（不新增功能）

## MODIFIED Requirements

### Requirement: 主题配色重新设计

系统 SHALL 更新所有 14 套系统主题的配色值。

#### Scenario: 主题切换后可视效果
- **GIVEN** 用户切换任意主题
- **WHEN** `[data-theme]` 属性变更
- **THEN** 页面的背景色、卡片背景色、代码块背景色、边框色等均为新的精心设计的配色
- **AND** 所有 CSS 变量完整覆盖，无缺失导致视觉断裂

#### Scenario: 护眼主题效果
- **GIVEN** 用户选择护眼主题（eye-protection）
- **WHEN** 主题切换完成
- **THEN** 页面背景为低饱和度米绿色 `#E4ECD9`，文字为深绿灰 `#384A3E`
- **AND** 整体视觉温和不刺目，饱和度明显低于原方案

#### Scenario: 暗色主题效果
- **GIVEN** 用户选择深色主题（dark）
- **WHEN** 主题切换完成
- **THEN** 页面背景为深灰 `#18181C` 而非纯黑
- **AND** 文字为柔白 `#E4E4E8`

#### Scenario: 亮色主题效果
- **GIVEN** 用户选择亮色主题（light）
- **WHEN** 主题切换完成
- **THEN** 页面背景为极浅蓝灰 `#F4F4F7`，卡片为纯白 `#FFFFFF`
- **AND** 整体色调偏冷，与 "default" 的温暖形成对比

## REMOVED Requirements

无
