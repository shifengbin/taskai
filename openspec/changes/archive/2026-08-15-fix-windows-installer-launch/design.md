# Design: fix-windows-installer-launch

## Context

- 现状：`SystemLauncher.LaunchInstaller` 对所有平台统一走 `startDetached`（`exec.Command` + `backgroundprocess.Configure`，即 `HideWindow` + `CREATE_NO_WINDOW`）。
- 事实约束：Windows NSIS 安装程序要求管理员权限。`CreateProcess` 无法启动要求提升的 exe（直接返回 740），只有 `ShellExecute` 家族（资源管理器同款打开机制）能触发 UAC 提升后启动。
- 项目里已有同类先例：`ReleasePageInvocation` 在 Windows 上用 `rundll32.exe url.dll,FileProtocolHandler` 打开 URL，本质就是借道 ShellExecute。

## Goals / Non-Goals

- Goal：Windows 上成功启动要求管理员权限的 NSIS 安装程序；不弹控制台黑框；启动失败时仍保持应用运行（既有语义）。
- Non-Goal：静默安装（`/S`）、权限降级安装器、改变下载/校验/缓存逻辑、改变前端交互。

## Decisions

### D1. Windows 安装启动使用 ShellExecuteW "open" 动词

新增 `launcher_windows.go`（`//go:build windows`）：

```go
var shellExecute = windows.ShellExecute

func startInstallerDetached(invocation Invocation) error {
    verb "open"; file = invocation.Command; cwd = filepath.Dir(file); SW_SHOWNORMAL
    return shellExecute(0, verb, file, nil, cwd, windows.SW_SHOWNORMAL)
}
```

- `launcher_other.go`（`!windows`）中 `startInstallerDetached = startDetached`，macOS/Linux 行为完全不变。
- `SystemLauncher` 增加 `startInstaller` 函数字段，`NewSystemLauncher` 统一注入；`LaunchInstaller` 改用该字段，`OpenReleasePage` 继续用原 `start` 字段。
- `shellExecute` 声明为包级变量是唯一为可测试性引入的接缝：单元测试替换它以断言动词/路径/参数，不真正启动进程。
- 错误包装保持 `启动安装程序: %w`，用户取消 UAC（`ERROR_CANCELLED`）等失败会走既有「安装程序启动失败」语义（不退出、可重试）。
- 不额外弹黑框：ShellExecute 不为 GUI 子系统程序创建控制台；NSIS 安装向导窗口本身是预期 UI。

### D2. Alternatives considered

- `rundll32 url.dll,FileProtocolHandler <exe>`：同样有效，但多一层间接且错误码语义被 rundll32 吞掉；直接 `ShellExecuteW` 更简单、错误更可读。
- 要求 NSIS 改为 `RequestExecutionLevel user`（per-user 安装）：改变安装位置与升级/卸载语义，超出本修复范围。
- PowerShell `Start-Process -Verb RunAs`：引入脚本进程与转义问题，无收益。

## Risks / Trade-offs

- `ShellExecuteW` 成功仅代表"已交给系统启动"，UAC 取消发生在其后：用户取消 UAC 时本进程可能已调用退出。缓解：`LaunchInstaller` 对 ERROR_CANCELLED 返回错误（x/sys 将 `SE_ERR_ACCESSDENIED/5` 映射为错误），前端停留在安装确认并可重试。实测以单元测试 + 本地真机验证为准。
- ShellExecute 需要COM 已初始化的场景仅限某些动词（如 "runas" 在极旧系统）；桌面应用主线程场景下 "open" 无此要求。

## Migration Plan

纯内部实现替换，无数据/接口迁移。发布 v0.0.2 后，v0.0.1 安装的用户即可通过自动更新完成升级（本修复正是让这条链路可用）。

## Open Questions

无。
