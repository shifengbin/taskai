# 多平台构建脚本实施计划

> **供 Codex 使用：** 按 `superpowers:executing-plans` 技能逐项执行此计划。

**目标：** 为 Linux、Windows 和 macOS 提供可重复执行的原生 Wails 构建入口。

**架构：** 构建脚本均从项目根目录执行 Wails CLI。Linux 根据系统可用的 WebKitGTK 4.0/4.1 自动选择对应构建标签；Windows 和 macOS 分别使用 PowerShell 与 Bash 脚本，并允许选择受支持的目标架构。

**技术栈：** Bash、PowerShell、Wails CLI、pkg-config、WebKitGTK。

---

### 任务 1：创建平台构建脚本

**文件：**
- 新建：`scripts/build-linux.sh`
- 新建：`scripts/build-windows.ps1`
- 新建：`scripts/build-macos.sh`

**步骤：**
1. 在每个脚本中定位项目根目录并校验 Wails CLI。
2. Linux 脚本校验 GTK/WebKitGTK，选择 `webkit2_41` 或默认 WebKitGTK 4.0 标签。
3. Windows 和 macOS 脚本校验本机系统与对应开发工具链，并调用原生 Wails 打包命令。
4. 输出构建产物所在的 `build/bin` 目录。

### 任务 2：记录使用方法

**文件：**
- 修改：`README.md`

**步骤：**
1. 增加各平台的调用命令、可选架构参数和产物路径。
2. 说明 Linux WebKitGTK 4.0/4.1 的前置依赖，以及 Windows 和 macOS 的本机构建要求。

### 任务 3：验证

**文件：**
- 测试：`scripts/build-linux.sh`
- 测试：`scripts/build-macos.sh`
- 测试：`scripts/build-windows.ps1`

**步骤：**
1. 使用 `bash -n` 校验两个 Bash 脚本语法。
2. 使用 PowerShell 解析器校验 Windows 脚本（环境提供时）。
3. 在 Linux 环境实际运行 `scripts/build-linux.sh`，确认生成 `build/bin/taskai`。
