# NSIS 安装包记住安装路径

## 当前状态

`InstallDir` 硬编码为 `"$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"`（[project.nsi#L75](file:///d:/峡谷/Dev/本地项目/jot/build/windows/installer/project.nsi#L75)），每次安装都显示默认路径，不记忆用户上次选择。

## 改动目标

仅修改 `project.nsi` 一个文件，`wails_tools.nsh` 不动（自动生成）。

## 三处修改

| # | 位置 | 改动 | 用途 |
|---|------|------|------|
| 1 | `.onInit` 函数末尾 | 添加 `ReadRegStr` 读取上次路径 | 安装/静默安装前恢复 |
| 2 | `Section` 安装段末尾 | 添加 `WriteRegStr` 保存当前路径 | 安装完成后存储 |
| 3 | `Section "uninstall"` 卸载段 | 添加 `DeleteRegValue` 清理注册表 | 卸载时清理 |

注册表位置：`HKCU\Software\jot\jot` → `InstallPath`

## 边界情况

- 全新安装 → 无注册表值，显示默认路径
- 升级/重装 → 自动恢复上次路径
- 用户手动改路径 → 安装后保存新路径
- 静默安装 `/S` → `.onInit` 照样执行恢复
- 卸载 → 清理注册表项（下次安装显示默认路径）

## 验证

1. 运行 `wails build -clean -platform windows/amd64 -nsis` 构建安装包
2. 安装时确认初始路径为默认 `C:\Program Files\jot\jot`
3. 修改路径后再安装，确认恢复为上次路径
4. 卸载后重装，确认恢复为默认路径
