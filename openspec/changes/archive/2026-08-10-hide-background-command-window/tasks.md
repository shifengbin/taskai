## 1. 导出并复用后台进程窗口策略

- [x] 1.1 将 `internal/lifecycle` 的 `configureBackgroundProcess` 导出为 `ConfigureBackgroundProcess`（`process_windows.go` 与 `process_other.go` 同步改名），并更新同包内调用方 `shell.go`、`git.go` 以及 `process_windows_test.go` 的调用名
- [x] 1.2 在 `app.go` 的 `configureCommandProcess` 中调用 `lifecycle.ConfigureBackgroundProcess(process)`，使 `startTaskCommand`、`startTaskScript` 两条后台路径在 Windows 上获得 `HideWindow` + `CREATE_NO_WINDOW`（非 Windows 为 no-op）

## 2. 测试

- [x] 2.1 新增 `app_windows_test.go`：对 `cmd`、PowerShell、直执（`shellPath == ""`）三种包裹形态，调用 `commandProcess(...)` + `configureCommandProcess(...)`，断言 `SysProcAttr.HideWindow == true` 且 `CreationFlags & CREATE_NO_WINDOW != 0`
- [x] 2.2 确认非 Windows 下 `configureCommandProcess` 行为不变（`ConfigureBackgroundProcess` 为 no-op），既有 `app_test.go` 全部通过

## 3. 验证

- [x] 3.1 运行 `go build ./...` 与 `go test ./...`；额外以 `GOOS=windows` 构建并运行 Windows 标签测试
- [x] 3.2 在 Windows 上手动验证：标记为后台的任务菜单命令不再弹出控制台黑框、目标程序正常打开；标记为显示终端的命令内嵌终端行为不变；命令失败仍触发后置脚本
