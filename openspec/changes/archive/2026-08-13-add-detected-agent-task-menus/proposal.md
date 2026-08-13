## Why

用户安装 Codex 或 Claude CLI 后，仍需手动在 TaskAI 中重复配置常用启动命令。应用启动时自动识别本机已有的代理命令并补充任务菜单设置，可以让所有支持平台获得一致的开箱即用入口。

## What Changes

- TaskAI 每次启动后在当前进程可用的命令环境中检测 `codex` 与 `claude`。
- 检测到 `codex` 时，向现有任务菜单设置补充名称为 `codex`、参数为 `--yolo`、显示终端的命令项。
- 检测到 `claude` 时，向现有任务菜单设置补充名称为 `claude`、参数为 `--dangerously-skip-permissions`、显示终端的命令项。
- Linux、macOS 与 Windows 使用相同的检测和补充规则；重复启动不得生成重复菜单项，也不得覆盖其他菜单设置。
- 自动补充的命令继续复用现有任务菜单执行、任务操作目录和内嵌终端行为。

## Capabilities

### New Capabilities

- `detected-agent-task-menus`: 定义应用启动时检测 Codex、Claude 命令并幂等补充任务菜单设置的跨平台行为。

### Modified Capabilities

无。

## Impact

- 后端应用启动编排、命令查找与设置持久化逻辑。
- 任务菜单设置的规范化和回归测试；现有 Wails API 与前端数据结构预计无需变更。
- Linux、macOS 与 Windows 的命令查找兼容性，以及 Wails 开发模式下的菜单设置与命令终端集成测试。
