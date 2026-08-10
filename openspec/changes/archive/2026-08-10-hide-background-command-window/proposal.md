## Why

在 Windows 上，任务菜单中标记为"后台启动（不显示终端）"的命令项被触发时，会先弹出一个可见的控制台黑框，目标程序随后才打开、黑框随即消失。这与用户显式选择"不显示终端"的意图相矛盾：用户要的是无可见终端的后台执行，实际却看到了一闪而过的控制台窗口。根因是后台命令的启动路径（`startTaskCommand`/`startTaskScript`）未像生命周期命令链那样应用"无控制台窗口"进程属性，二者存在不对称。

## What Changes

- 将"无控制台窗口"进程属性（`CREATE_NO_WINDOW` + `HideWindow`）应用到任务菜单后台命令的启动路径（`ShowTerminal == false` 分支），与生命周期命令链后台进程的既有行为对齐。
- 同步覆盖后台前置/后置脚本（`startTaskScript`）在同一路径上的执行，确保经 `cmd.exe`/PowerShell 包裹或直接执行的后台命令都不再弹出可见控制台。
- 收紧现有 `windows-lifecycle-command-execution` 中"无窗口仅限生命周期链"的边界：无窗口属性同样适用于后台（不显示终端）的任务菜单命令；**不**改变显示终端（`ShowTerminal == true`）的内嵌终端既有行为。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `windows-lifecycle-command-execution`: 将"无控制台窗口"要求从仅生命周期命令链后台进程，扩展到同样由任务菜单发起、且标记为后台（`ShowTerminal == false`）的命令与脚本启动路径；保留内嵌终端（`ShowTerminal == true`）不在该约束内。

## Impact

- Go：`app.go` 的 `startTaskCommand`、`startTaskScript`（及共享的 `configureCommandProcess`）需对生成的 `*exec.Cmd` 应用现有的 `configureBackgroundProcess`（`internal/lifecycle/process_windows.go` 已提供，非 Windows 上为 no-op）。
- 复用已有平台分发函数，无新依赖、无前端/Wails 绑定变更、无数据迁移。
- 行为变化仅限 Windows 平台的后台命令；macOS/Linux 维持现状。
- 现有测试以可注入的 `commandStarter` 为先例，可对进程窗口标志做断言验证。
