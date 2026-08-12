# Rnx.toml release 任务跨平台拆分计划

## Summary

将 [Rnx.toml](file:///d:/峡谷/Dev/本地项目/jot/Rnx.toml) 中硬编码 Windows 的 `release` 任务拆分为按平台分发的三个任务（`release_windows` / `release_linux` / `release_darwin`），保留 `release` 作为统一入口。利用 rnx 的 `platform` 字段自动跳过不匹配平台的任务，实现 `rnx --run release` 在任意平台一键发布。

## Current State Analysis

现有 `release` 任务（[Rnx.toml L116-138](file:///d:/峡谷/Dev/本地项目/jot/Rnx.toml#L116-L138)）有 3 处 Windows 硬编码，非 Windows 平台直接失败：

1. `wails build ... -nsis`：NSIS 安装器仅 Windows 支持（L121）
2. `fck mv ... jot-amd64-installer.exe ...`：installer.exe 仅 `-nsis` 时生成（L124）
3. `release_name = '{{app_name}}_windows_amd64_{{git_version}}.zip'`：打包名硬编码 windows（[L32](file:///d:/峡谷/Dev/本地项目/jot/Rnx.toml#L32)）

已确认的 rnx 行为（[rnx spec.md](file:///c:/Users/QIAOMU/.trae-cn/skills/rnx-task/references/spec.md)）：

* `platform = ["windows"/"linux"/"darwin"]` 按当前平台自动跳过不匹配任务（L302-325）

* 官方推荐"平台任务 + depends\_on 组合"模式（L319-323 示例 `build_all`）

* 命令由 mvdan.cc/sh/v3 纯 Go 解析，跨平台一致，现有 `if [ -d ... ]`（L100）无需改动

* 变量支持全局/任务级、动态变量（`@` 命令）

fck pack 语法：`fck pack -o <输出> <源>`，扩展名决定格式（`.zip`/`.tar.gz`），支持目录打包。

额外发现：UPX 不支持 macOS ARM64（Mach-O），`release_darwin` 必须去掉 `-upx`。

## Proposed Changes

仅修改 [Rnx.toml](file:///d:/峡谷/Dev/本地项目/jot/Rnx.toml) 一个文件。

### 1. 全局变量：`release_name` → 三个平台变量（L32）

替换为：

```toml
release_name_windows = '{{app_name}}_windows_amd64_{{git_version}}.zip'
release_name_linux = '{{app_name}}_linux_amd64_{{git_version}}.tar.gz'
release_name_darwin = '{{app_name}}_darwin_amd64_{{git_version}}.tar.gz'
```

### 2. 重写 `[task.release]` 并新增三个平台任务（L116-138）

```toml
[task.release]
# 任务描述
desc = '发布程序（按当前平台自动执行对应发布任务）'
# 命令列表
cmds = [
    'echo "release 已按当前平台分发，请查看上方各平台任务输出"'
]
# 依赖任务列表
depends_on = [
    'release_windows',
    'release_linux',
    'release_darwin'
]
# 支持运行的平台, 为空时表示支持所有平台(支持配置为: windows, linux, darwin)
platform = []

[task.release_windows]
# 任务描述
desc = '发布 Windows 程序（含 NSIS 安装器）'
# 命令列表
cmds = [
    'wails build -ldflags "{{ldflags}}" -o {{exe_name}} -trimpath -upx -upxflags="-2" -clean -nsis',
    'fck pack -o {{output_dir}}/{{release_name_windows}} {{exe_path}}',
    'fck rm -f {{exe_path}}',
    'fck mv -f {{output_dir}}/jot-amd64-installer.exe {{output_dir}}/jot_windows_amd64_installer_{{git_version}}.exe'
]
# 依赖任务列表
depends_on = [
    'frontend',
    'fc'
]
# 支持运行的平台
platform = ['windows']

[task.release_linux]
# 任务描述
desc = '发布 Linux 程序（tar.gz 包）'
# 命令列表
cmds = [
    'wails build -ldflags "{{ldflags}}" -o {{exe_name}} -trimpath -upx -upxflags="-2" -clean',
    'fck pack -o {{output_dir}}/{{release_name_linux}} {{exe_path}}',
    'fck rm -f {{exe_path}}'
]
# 依赖任务列表
depends_on = [
    'frontend',
    'fc'
]
# 支持运行的平台
platform = ['linux']

[task.release_darwin]
# 任务描述
desc = '发布 macOS 程序（tar.gz 包，无 UPX）'
# 命令列表
cmds = [
    'wails build -ldflags "{{ldflags}}" -o {{exe_name}} -trimpath -clean',
    'fck pack -o {{output_dir}}/{{release_name_darwin}} {{output_dir}}/{{app_name}}.app',
    'fck rm -f -r {{output_dir}}/{{app_name}}.app'
]
# 依赖任务列表
depends_on = [
    'frontend',
    'fc'
]
# 支持运行的平台
platform = ['darwin']
```

### 改动说明

| 变更                                            | 原因                                    |
| --------------------------------------------- | ------------------------------------- |
| `release_name` → 三个平台变量                       | 打包名按平台区分（zip/tar.gz）                  |
| `-nsis` + installer 重命名 → 仅 `release_windows` | NSIS 仅 Windows 支持                     |
| `release_darwin` 去掉 `-upx`                    | UPX 不支持 macOS ARM64                   |
| `release_darwin` 打包 `{{app_name}}.app` 目录     | Wails 在 macOS 产物为 .app bundle，非裸可执行文件 |
| `release` 入口 `depends_on` 三平台任务               | rnx 自动跳过不匹配平台任务，任意平台一键发布              |
| 其他任务（build/run/frontend/fc/fclint）不动          | 已跨平台，无需改动                             |

## Assumptions & Decisions

* **决策**：macOS 与 Linux 均打包 `tar.gz`（与现有 fck pack 工作流一致，避免引入 dmg/deb 等外部工具）；安装器需求后续单独扩展。

* **决策**：保留 `release` 统一入口，用户无需记忆平台任务名（官方推荐模式）。

* **假设**：Linux/macOS 构建环境装有 `wails`、`fck`、`upx`（linux 分支）、npm 等依赖；与 Windows 现状一致。

* **假设**：`wails build -o jot` 在 macOS 生成 `build/bin/jot.app`（Wails v2 默认行为）；若实际命名不同，执行时按实际产物调整。

* **不改动**：`build/darwin/Info.plist`（默认模板可用）、`build/linux/`（缺目录不影响构建，Wails 用根 `appicon.png` 派生图标）。

## Verification steps

1. `rnx check`：配置校验通过（TOML 语法、命令 shell 语法、变量引用、无循环依赖）。
2. `rnx -l`：任务列表出现 `release`、`release_windows`、`release_linux`、`release_darwin`。
3. 本机（Windows）执行 `rnx --run release` 前，先确认第 1、2 步通过；完整发布耗时较长（wails build + npm），由用户按需触发；预期 Windows 分支正常执行、linux/darwin 分支被自动跳过。
4. 可选冒烟：`rnx --run release_windows` 单独验证 Windows 分支（与现状行为一致）。

