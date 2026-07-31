## Why

Windows 桌面版执行生命周期命令链时，应用启动的 `cmd.exe`、PowerShell 或 Git 子进程可能创建并短暂显示命令行窗口，打断用户当前工作。生命周期链的输入、输出和错误均由应用接管，因此不需要向用户暴露独立控制台。

## What Changes

- 在 Windows 上以无控制台窗口方式执行自定义生命周期 Shell 命令。
- 在 Windows 上以相同方式执行命令链内置的 Git 命令。
- 保持命令链既有工作目录、标准输入输出传递、错误处理、环境变量和执行顺序不变。
- 不改变应用内嵌终端及用户主动启动的任务菜单命令的窗口行为。

## Capabilities

### New Capabilities
- `windows-lifecycle-command-execution`: 定义 Windows 生命周期命令链子进程不显示控制台窗口且保留既有执行语义的行为。

### Modified Capabilities

- 无。

## Impact

- 受影响代码：`internal/lifecycle/shell.go`、`internal/lifecycle/git.go`，以及用于隔离 Windows 进程属性的平台专用辅助代码和测试。
- 受影响系统：Windows 桌面版的自定义生命周期 Shell 命令与内置 Git 克隆命令。
- 不涉及前端、持久化格式、公开 API 或新增运行时依赖。
