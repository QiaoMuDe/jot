# 优化浅色主题对比度

## Summary
提高 7 个浅色主题的文字与背景对比度，使界面更清晰易读。默认主题（default）保持不变。

## 需要修改的文件
- `frontend/src/css/variables.css`

## 优化原则
1. `text-secondary` 加深到 `#5A-6A` 范围（当前大多在 `#7C-9C`，太淡）
2. `text-muted` 适当加深
3. `border` 加深到 `#C8-D5` 范围，增加可见性
4. 保持各主题的色调风格不变

## 各主题调整详情

### 1. light（浅色）
- `--text-secondary`: `#909098` → `#5A5A62`
- `--text-muted`: `#B0B0B0` → `#787880`
- `--border`: `#E6E6EC` → `#D0D0D8`

### 2. nord（北极）
- `--text-secondary`: `#6D7590` → `#4A5270`
- `--text-muted`: `#9199B0` → `#6A7290`
- `--border`: `#D0D7E4` → `#B8C0D4`

### 3. eye-protection（护眼）
- `--text-secondary`: `#658570` → `#3A5A48`
- `--text-muted`: `#8EA890` → `#5A7A68`
- `--border`: `#D5DEC4` → `#B8C8A8`

### 4. catppuccin-latte（暖咖）
- `--text-secondary`: `#908B9E` → `#5A5568`
- `--text-muted`: `#BEB8C4` → `#8A849A`
- `--border`: `#DBD1D0` → `#C0B6B5`

### 5. gruvbox-light（旧纸）
- `--text-secondary`: `#7C6F64` → `#4A4038`
- `--text-muted`: `#A09480` → `#6A5E50`
- `--border`: `#D0C4A5` → `#B8AC90`

### 6. quiet-light（静谧）
- `--text-secondary`: `#8A7E8A` → `#5A4E5A`
- `--text-muted`: `#B8AEB8` → `#8A7E8A`
- `--border`: `#DCD2DB` → `#C0B6BF`

### 7. ysgrifennwr（暖笺）
- `--text-secondary`: `#9C8B7A` → `#5A4A3A`
- `--text-muted`: `#C0B09E` → `#8A7A68`
- `--border`: `#E0D0BB` → `#C8B8A0`

## Verification
- 逐个切换主题，检查文字是否清晰可读
- 检查边框是否可见
- 确认各主题色调风格保持一致
