# Windows ConPTY 关闭读取消除误报设计

## 目标

关闭 Windows 嵌入式终端或退出应用时，不再把 ConPTY 返回的 `ERROR_OPERATION_ABORTED`（995）显示为终端 I/O 异常；其他读取错误仍须保留并上报。

## 根因

终端管理器的读取协程会在 `Session.Close()` 关闭 ConPTY 后收到错误 995。该错误表示应用主动中止了挂起 I/O，是关闭过程的正常结果。`read_error_windows.go` 当前没有将其列入预期读取结束，因而 `Manager.watch` 会发布 `error` 事件。

## 方案

- 在不受构建标签限制的测试辅助函数中，通过 `errors.Is` 识别被包装的 Windows 错误 995。
- Windows 平台的 `isExpectedTerminalReadError` 调用该辅助函数。
- Unix 平台继续仅将 PTY 的 `EIO` 视为预期结束，不改变现有行为。
- `Manager.watch` 保持既有分支，只因 Windows 分类结果修正而不再发布误报事件。

## 验证

- 单元测试验证错误 995（包括被包装的错误）会被识别。
- Windows 构建标签下的管理器测试验证 995 只产生退出事件；其他错误仍会产生错误事件，避免掩盖真实 I/O 故障。
- 运行 Go 全量测试、前端测试和项目 Linux 构建脚本；Windows 专用测试在 Windows 主机上执行。
