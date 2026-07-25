# NSIS 安装程序增强计划

## 概要

对 Wails 生成的 NSIS 安装脚本 `project.nsi` 进行 5 项增强：中文语言包、可选桌面快捷方式、卸载保留数据确认、压缩优化、安装后可选运行。

**涉及文件**（仅修改 `project.nsi`，`wails_tools.nsh` 为自动生成文件不动）：

| 文件                                    | 操作       |
| ------------------------------------- | -------- |
| `build\windows\installer\project.nsi` | 修改：添加新功能 |

***

## 当前状态分析

### 1. 项目.nsi 当前结构

* **语言**：仅 `!insertmacro MUI_LANGUAGE "English"`

* **快捷方式**：无条件创建开始菜单和桌面快捷方式

* **卸载**：无确认框，直接删除 `$INSTDIR`、`$AppData\jot.exe\`、注册表项和快捷方式；不删除 `~/.jot/`（数据库目录）

* **压缩**：未设置 `SetCompressor`，使用 NSIS 默认值（zlib）

* **安装完成**：使用 `MUI_PAGE_FINISH`（无运行选项）

### 2. 用户数据目录

数据库路径：`~/.jot/data/jot.db`（通过 `os.UserHomeDir()` + `.jot` 在 Go 中解析）
NSIS 中通过 `ReadEnvStr $0 "USERPROFILE"` 可获取用户主目录，拼接得到 `$0\.jot`

### 3. 注意点

* `wails_tools.nsh` 开头有 `# DO NOT EDIT - Generated automatically by wails build`，**不应修改**

* 所有自定义逻辑必须写在 `project.nsi` 中

* 卸载时运行级别为 `admin`（通过 `wails_tools.nsh` 定义 `RequestExecutionLevel "admin"`）

***

## 具体变更

### 变更 1：中文语言包

**文件**：`project.nsi`
**原因**：让安装界面显示中文，提升国内用户使用体验

* 将 `!insertmacro MUI_LANGUAGE "English"` 替换为：

  * `!insertmacro MUI_LANGUAGE "SimpChinese"`

  * `!insertmacro MUI_LANGUAGE "English"`（保留英文作为回退）

* MUI 会自动根据系统区域选择语言

### 变更 2：可选桌面快捷方式

**文件**：`project.nsi`
**原因**：用户可能不希望安装时自动创建桌面快捷方式

实现步骤：

1. 在页码宏之前添加 `!insertmacro MUI_PAGE_COMPONENTS`，插入组件选择页
2. 定义 `MUI_COMPONENTSPAGE_SMALLDESC` 启用小描述样式
3. 新增一个具名 Section `"桌面快捷方式(&D)"` 空出桌面快捷方式创建逻辑
4. 从主 Section 中移除 `CreateShortCut "$DESKTOP\..."` 行
5. 开始菜单快捷方式保留在主 Section 中（始终创建）
6. 在 `Section "uninstall"` 后添加组件描述函数

### 变更 3：卸载保留数据确认

**文件**：`project.nsi`
**原因**：用户卸载时可能想保留数据库和设置文件，以便重新安装后继续使用

实现步骤：

1. 在 `Section "uninstall"` 开头添加 `MessageBox` 两步确认：

   * 第一步：`确定要卸载 xxx 吗？`（MB\_YESNO）

   * 第二步（确认后）：`是否保留用户数据？选择「是」保留数据库和设置文件，选择「否」彻底删除`（MB\_YESNO）
2. 用户选择"不保留"时，通过 `ReadEnvStr $0 "USERPROFILE"` 获取用户主目录，拼接 `.jot` 路径，执行 `RMDir /r "$0\.jot"`
3. 当前卸载本身不删除 `~/.jot/`，所以"保留"即维持现状

### 变更 4：压缩优化

**文件**：`project.nsi`
**原因**：减小安装包体积，提升下载和安装体验

* 在文件开头添加 `SetCompressor /SOLID lzma`

* `/SOLID lzma` 提供最佳压缩比，适合桌面应用安装包

### 变更 5：安装后可选运行

**文件**：`project.nsi`
**原因**：用户安装完成后可选择立即运行程序，提升体验

* 在 `MUI_PAGE_FINISH` 前添加：

  * `!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"`

  * `!define MUI_FINISHPAGE_RUN_TEXT "运行 ${INFO_PRODUCTNAME}"`

  * `!define MUI_FINISHPAGE_RUN_NOTCHECKED`（默认不勾选，由用户主动选择）

***

## 决策和假设

| 决策/假设                | 说明                                                     |
| -------------------- | ------------------------------------------------------ |
| 快捷方式组件默认勾选           | 桌面快捷方式在组件页默认勾选（`SectionIn RO` 不行，需用 checked 标志）        |
| 数据目录硬编码 `~/.jot`     | 通过 `USERPROFILE` 环境变量拼接，与 Go 后端 `DefaultDBPath()` 逻辑一致 |
| WebView2 数据          | `$AppData\${PRODUCT_EXECUTABLE}` 始终被删除（非用户数据，可重建）      |
| `wails_tools.nsh` 不动 | 该文件由 `wails build` 自动重新生成，修改会被覆盖                       |
| 不需要图标变更              | 保持原有 `..\icon.ico` 不变                                  |

## 验证步骤

1. 运行 `wails build --target windows/amd64 --nsis` 生成安装包
2. 安装验证：

   * 安装界面显示中文

   * 组件选择页出现"桌面快捷方式"选项（默认勾选）

   * 安装完成后出现"运行 jot"复选框（默认不勾选）
3. 卸载验证：

   * 弹出卸载确认框

   * 选择"保留数据"→ 卸载后 `~/.jot/` 目录保留

   * 再次安装选择"不保留数据"→ `~/.jot/` 被删除
4. 安装包体积对比优化前

