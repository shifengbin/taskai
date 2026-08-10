# 终端快捷键生效程序范围设计

## 背景

终端快捷键在所有活动终端上都会拦截。某些组合键与全屏 TUI 程序自身的快捷键冲突：例如把 `Shift+Enter` 配置成写入 `\`+`Enter`，会让 codex 这类需要用 `Shift+Enter` 输入换行的 TUI 拿不到原始按键。

需要一种机制把快捷键的生效范围缩小到“由特定程序创建的终端”，让其他终端（包括各种 TUI）自动透传原始按键。

## 生效程序语义（include 包含）

每条快捷键带一个可选的 `includePrograms` 字符串数组：

- **留空或缺失**：在所有终端生效（保留既有行为，向后兼容）。
- **非空**：仅在该终端的启动命令匹配列表中任一程序时拦截，其余终端透传原始按键。

采用包含（allowlist）而非排除（denylist）：Shell 种类有限可枚举，而需要透传的 TUI 是开放集合，用白名单把快捷键收缩到 Shell 更可靠。

## 判定依据：启动命令

是否生效依据 **TaskAI 创建该终端时使用的启动命令**，而不是终端内当前运行的程序：

- 命中来源：任务菜单命令项、生命周期命令链、显式命令终端等创建路径记录的启动命令。
- **不追踪**用户在 Shell 内手动执行的程序——例如在普通 Shell 里运行 `codex` 不改变该终端的归属。

这样既避免依赖运行时前台进程检测（跨平台、跨架构脆弱），也匹配用户的直觉：终端的“类型”在创建时就确定了。

## 归一化与匹配

为兼容 `codex`、`codex.exe`、`C:\tools\codex.exe`、`/usr/local/bin/codex`、`CODEX.EXE` 等写法，两端在比较前对程序名做同一归一化：

1. 去首尾空白；
2. 按路径分隔符 `/`、`\` 切分取最后一段（basename）；
3. 去掉 Windows 可执行扩展名 `.exe`、`.com`；
4. 转小写。

归一化后做**精确相等**匹配，列表中任一项相等即视为生效。不使用子串匹配——子串会误伤（如 `sh` 命中 `powershell`）。

后端 `NormalizeTerminalShortcuts` 在保存时对 `IncludePrograms` 做去空白、丢空项、按小写去重（保留首次出现的写法）。

## 实现要点

- 后端 `terminal.Info` 增加 `Command` 字段，在创建终端时由已计算的启动命令（`request.Command`，缺失时为 `request.ShellPath`）带出，随创建结果返回前端。
- 后端 `settings.TerminalShortcut` 增加 `IncludePrograms []string`（`omitempty`），并在归一化阶段清洗。
- 前端 `TerminalRecord` 增加 `command`，经 Wails 绑定直传；待处理事件合并（`mergePendingTerminalEvents`）以展开原对象的方式保留该字段。
- 前端 `terminalShortcutApplies(shortcut, command)` 在 `TerminalView` 的 `customKeyEventHandler` 中、命中快捷键后、写入前调用：范围外 `return true` 透传，范围内照旧 `preventDefault` + 写入 + `return false`。
- 设置 UI 在每条快捷键卡片下增加“生效程序”编辑器，逐项增删，并标注“留空=全部终端生效”。为避免手填程序名出错，编辑器不提供自由文本输入，而是从“可显示终端的程序”下拉中选择：候选项由当前 Shell 路径与标记为显示终端（`showTerminal`）的任务菜单命令项的启动命令经归一化（basename + 去 `.exe`/`.com` + 小写）去重后组成；后台启动（不显示终端）的命令项不出现在候选中。

## 测试

- Go：`Info.Command` 快照命令终端与纯 Shell 终端的取值；`IncludePrograms` 归一化（去空白、去重）与缺失时保持 `nil`。
- 前端工具：归一化命中各类写法、空列表=全部生效、不相关程序与缺失命令不匹配。
- 终端视图：范围外透传不写入、范围内仍写入已配置动作。
- 设置界面：从可显示终端的程序下拉中选择、删除生效程序并随快捷键保存；后台命令不进入候选。
