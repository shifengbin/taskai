# Proposal: fix-windows-installer-launch

## Why

Windows 上自动更新在最后一步失败：用户点击「立即安装」后应用弹出「安装更新」失败对话框，错误文本为：

```
启动安装程序: fork/exec C:\...\updates\v0.0.1\taskai-amd64-installer.exe: The requested operation requires elevation.
```

（该错误出现在用户无法复制文本的提示框中，由临时探针复现完整链路后捕获：检查 ✓ → 下载 ✓ → 校验 ✓ → 启动安装程序 ✗。）

根因：NSIS 安装程序内嵌 UAC 清单为 `requestedExecutionLevel=requireAdministrator`（由 `build/windows/installer/wails_tools.nsh` 的 `REQUEST_EXECUTION_LEVEL admin` 决定，并已在 v0.0.1 发布产物二进制中验证）。而 `internal/updater/launcher.go` 的 `startDetached` 通过 `exec.Command`（Win32 `CreateProcess`）启动安装程序。非提升进程调用 `CreateProcess` 启动要求管理员的可执行文件时，不会出现 UAC 提示，而是立即返回 `ERROR_ELEVATION_REQUIRED (740)`。结果就是：下载、校验全部成功，唯独永远无法启动安装程序，自动更新在 Windows 上完全不可用。

## What Changes

- Windows 平台的安装程序启动改用 `ShellExecuteW` 的 `open` 动词：这是 Windows 上启动需要 UAC 提升的可执行文件的标准机制，会正常弹出 UAC 确认并启动 NSIS 安装向导，且不会额外创建控制台窗口。
- macOS / Linux 的安装启动路径与 Release 页面打开路径保持不变。
- `LauncherInvocation`（校验扩展名、生成命令）保持不变；仅替换 Windows 上的"如何启动"。
- 单元测试证明 Windows 安装启动走 `ShellExecute` 且动词为 `open`、目标为安装包路径；既有测试继续覆盖 Release 页面启动仍隐藏控制台。

## Impact

- 代码：`internal/updater/launcher.go`（Windows 安装启动分发）、新增 `internal/updater/launcher_windows.go` / `launcher_other.go`、对应测试。
- 规格：`application-auto-update` 中「安装遵守平台启动和现有退出语义」需求的 Windows 场景措辞（由 CreateProcess 后台进程改为 ShellExecute open 动词）。
- 不涉及更新清单、缓存、下载校验、前端交互的任何变更。
