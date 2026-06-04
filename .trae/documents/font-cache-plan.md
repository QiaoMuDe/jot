# 字体族列表缓存到数据库 + 刷新按钮 实施计划

## Summary

将系统字体列表缓存到 SQLite 数据库的 `settings` 表中，避免每次打开设置页时调用 Windows GDI API 枚举字体。同时在字体族下拉菜单旁边添加「刷新缓存」按钮，允许用户手动更新缓存（如安装新字体后）。

---

## Current State

- `GetSystemFonts()` 每次调用都通过 `fontutil.GetFonts()` 重新枚举系统字体（Windows GDI EnumFontFamiliesW）
- 前端 `fontFamilyList` 变量在 JS 内存中缓存，仅首次打开下拉时调用一次后端
- 应用运行期间安装新字体不会反映到下拉列表，除非重启应用
- Setting 模型 `Value` 字段 `size:500`，不足以存储完整字体列表（~500 字体 × 20 字节 ≈ 10KB）
- 字体族下拉菜单 trigger 右侧有充足空间放置刷新按钮

---

## Proposed Changes

### 1. 后端: `internal/models/setting.go` — 扩大 Value 字段容量

**What**: 将 `Value` 字段的 GORM size 从 `500` 改为 `10000`（或移除限制）

**Why**: JSON 序列化的字体列表可能超过 500 字符

**How**:
```go
// 修改前
Value string `gorm:"size:500" json:"value"`
// 修改后
Value string `gorm:"size:10000" json:"value"`
```

GORM AutoMigrate 会自动调整列类型，无需手动迁移。

---

### 2. 后端: `app.go` — 修改 `GetSystemFonts` + 新增 `RefreshFontCache`

**What**:
- `GetSystemFonts()`: 改为先读缓存（Setting key=`cached_fonts`），缓存未命中时调用 `fontutil.GetFonts()` 并写入缓存再返回
- `RefreshFontCache()`: 强制调用 `fontutil.GetFonts()`，更新缓存，返回新列表

**Why**: 减少不必要的系统调用，提升设置页打开速度

**How**:

```go
// 修改 GetSystemFonts — 支持缓存
func (a *App) GetSystemFonts() []string {
    // 尝试从缓存读取
    cached := a.settingService.Get("cached_fonts")
    if cached != "" {
        var fonts []string
        if err := json.Unmarshal([]byte(cached), &fonts); err == nil && len(fonts) > 0 {
            return fonts
        }
    }
    // 缓存不存在或解析失败，重新枚举并缓存
    fonts := fontutil.GetFonts()
    if data, err := json.Marshal(fonts); err == nil {
        _ = a.settingService.Set("cached_fonts", string(data))
    }
    return fonts
}

// 新增 RefreshFontCache — 强制刷新缓存
func (a *App) RefreshFontCache() []string {
    fonts := fontutil.GetFonts()
    if data, err := json.Marshal(fonts); err == nil {
        _ = a.settingService.Set("cached_fonts", string(data))
    }
    return fonts
}
```

需要在 `app.go` 的 import 中添加 `"encoding/json"`。

---

### 3. 前端: `frontend/index.html` — 添加刷新按钮

**What**: 在字体族下拉菜单 trigger 右侧添加一个刷新按钮 `🔄`

**Why**: 允许用户手动刷新字体缓存

**How**: 在 `font-family-select` 容器内，trigger 旁边添加：

```html
<div class="font-family-select" id="fontFamilySelect">
    <div class="font-family-trigger" id="fontFamilyTrigger">
        <span id="fontFamilyDisplay">DM Sans</span>
        <span class="font-family-arrow">▼</span>
    </div>
    <button class="font-refresh-btn" id="fontRefreshBtn" title="刷新字体缓存">🔄</button>
    <!-- dropdown 内容... -->
</div>
```

---

### 4. 前端: `frontend/src/style.css` — 刷新按钮样式

**What**: 添加 `.font-refresh-btn` 样式

**Why**: 让刷新按钮视觉上对齐 trigger

**How**:
```css
.font-refresh-btn {
    width: 32px;
    height: 32px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--input-bg);
    color: var(--text-secondary);
    font-size: 0.875rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: var(--transition);
    flex-shrink: 0;
    font-family: inherit;
}
.font-refresh-btn:hover {
    background: var(--hover-bg);
    color: var(--accent);
    border-color: var(--accent-light);
}
.font-refresh-btn:active {
    transform: rotate(360deg);
    transition: transform 0.3s ease;
}
```

另外在 `.font-family-select` 上加 `display: flex; gap: 8px;` 让 trigger 和按钮并排。

---

### 5. 前端: `frontend/src/main.js` — 刷新按钮事件处理

**What**: 
- 绑定 `fontRefreshBtn` 点击事件
- 点击时调用 `RefreshFontCache()`，清空前端 `fontFamilyList` 缓存，重新加载下拉选项

**Why**: 联动后端刷新，更新前端显示

**How**:

在 `initFontSettings()` 函数中添加：
```javascript
// 刷新字体缓存
els.fontRefreshBtn.addEventListener('click', async (e) => {
    e.stopPropagation(); // 防止触发下拉菜单
    const btn = els.fontRefreshBtn;
    btn.classList.add('spinning');
    
    if (window.go.main.App.RefreshFontCache) {
        const fonts = await window.go.main.App.RefreshFontCache();
        fontFamilyList = fonts || [];
    } else {
        // 降级：重新枚举
        fontFamilyList = ['DM Sans', 'Arial', ...];
    }
    
    // 重新渲染下拉选项（保持当前选中字体）
    const currentFont = getCurrentFontFamily();
    renderFontFamilyOptions(currentFont, els.fontFamilySearch.value);
    
    setTimeout(() => btn.classList.remove('spinning'), 500);
});
```

在 `els` 对象中添加：`fontRefreshBtn: $('fontRefreshBtn')`

---

## Assumptions & Decisions

1. **Setting Value 容量**: 扩到 10000 字符足够存储 Windows 系统全部字体的 JSON 序列化（实测约 500 字体 × 平均 15 字符 ≈ 7.5KB，JSON 序列化后约 10-12KB）
2. **缓存键名**: 使用 `cached_fonts`，与现有 `font_family` / `font_size` 区分
3. **JSON 序列化**: `encoding/json` 标准库，Go 原生支持，无额外依赖
4. **首次启动**: 首次设置页打开时自动枚举并缓存（`GetSystemFonts` 的 cache-miss 路径）
5. **不涉及 migration**: GORM AutoMigrate 自动处理字段 size 变更

---

## Verification

1. `go build ./...` — 编译通过
2. 打开设置页 → 字体下拉菜单 → 显示字体列表（不再调用 GDI）
3. 点击 🔄 按钮 → 下拉菜单刷新 → 新安装的字体出现
4. 重启应用 → 字体列表仍然存在（从缓存读取）
5. `go vet ./...` — 无问题
