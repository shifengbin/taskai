## Context

应用存在两条"后台进程"启动路径，二者对 Windows 控制台窗口的处理不一致：

- **生命周期命令链**（自定义 Shell 命令、内置 Git 命令）：经 `internal/lifecycle/shell.go`、`internal/lifecycle/git.go` 调用 `configureBackgroundProcess(process)`，在 Windows 上设 `HideWindow: true` 与 `CREATE_NO_WINDOW`，无可见控制台。
- **任务菜单后台命令 / 前后置脚本**（`ShowTerminal == false`）：经 `app.go` 的 `startTaskCommand`、`startTaskScript` → `configureCommandProcess`，后者仅设置 `Dir` 与 `Env`，**从不设置 `SysProcAttr`**。

TaskAI 是 Wails GUI 程序（`/SUBSYSTEM:WINDOWS`），自身无控制台。当它直接启动控制台子系统的 `cmd.exe`、PowerShell（`commandProcessForPlatform` 的 Windows 分支）或控制台目标程序（`shellPath == ""` 直执）时，因未设 `CREATE_NO_WINDOW`，Windows 内核为该子进程分配一个**可见**控制台；该 shell 进程执行完（`/C` 跑完即退）后控制台随进程关闭。用户看到的就是"黑框一闪、目标程序打开、黑框消失"——与"后台启动（不显示终端）"的显式选择相矛盾。

核实要点：

- `configureCommandProcess` 仅被 `startTaskCommand`（`app.go:1474`）与 `startTaskScript`（`app.go:1494`）调用；内嵌终端经终端管理器 `terminal.StartRequest` 后端（ConPTY/host）创建，**不**经过该函数——故在此设置窗口标志不会影响显示终端。
- 既有测试 `internal/lifecycle/process_windows_test.go` 已示范在 Windows 构建标签下断言 `SysProcAttr.HideWindow` 与 `CreationFlags & CREATE_NO_WINDOW`，可作为新测试的同构先例。

## Goals / Non-Goals

**Goals:**

- 消除 Windows 上后台任务菜单命令（及前后置脚本）启动时的可见控制台黑框。
- 与生命周期链路径对齐，使"后台进程窗口策略"只有唯一定义来源。
- 零数据迁移、零前端/Wails 绑定变更；非 Windows 平台行为不变。

**Non-Goals:**

- 不改变显示终端（`ShowTerminal == true`）的内嵌终端交互与可见性。
- 不隐藏后台命令自身另行显式创建的窗口（如命令内部再 `start` 一个 cmd）。
- 不改变 Shell 包裹方式（仍按 `commandProcessForPlatform` 走 `cmd /C`、PowerShell `-Command` 或 `-ic exec`），不绕过已配置 Shell。

## Decisions

### 决策 1：复用 `configureBackgroundProcess`，不引入新标志或新启动器

复用 `internal/lifecycle/process_windows.go` 已有的 `configureBackgroundProcess`（非 Windows 上 `process_other.go` 为 no-op），消除两条路径的不对称。

**备选与取舍：**

- *新增独立窗口标志字段*：会形成两处"后台窗口策略"定义，违背单一来源。**否决**。
- *VBS / `wscript //B` 无窗口启动器包一层*：引入外部文件与启动延迟，而 Win32 层 `CREATE_NO_WINDOW` 已彻底解决根因。**否决**。
- *后台命令绕过 Shell 直接 `exec.Command(command)`*：丢失 Shell 的环境、引号与别名一致性，语义改动大且超出范围。**否决**。

### 决策 2：在共享入口 `configureCommandProcess` 应用，覆盖命令与脚本两条路径

`configureCommandProcess` 是 `startTaskCommand`、`startTaskScript` 的公共配置点（已核实仅这两处调用）。在其中调用 `configureBackgroundProcess(process)`，即一次覆盖 `RunTaskCommand`、`ExecuteTaskMenuCommand` 的 `ShowTerminal == false` 分支以及前后置脚本 `startTaskScript`。

`configureBackgroundProcess` 写 `process.SysProcAttr`，`configureCommandProcess` 现仅写 `Dir`/`Env`，二者字段互不冲突，合并安全。附带收益：标志设置发生在 configure 阶段（早于 `Start`），可在 Windows 构建标签下对 `configureCommandProcess` 直接做单元断言，无需真正启动进程。

**备选与取舍：**

- *在 `startTaskCommand`/`startTaskScript` 各自显式调用*：两处重复、易漏。**否决**。

### 决策 3：内嵌终端结构性排除，无需运行时分支

显示终端（`ShowTerminal == true`）经终端管理器 `terminal.StartRequest` 后端创建，不经过 `configureCommandProcess`，因此无窗口策略天然不作用于它，无需增加 `if ShowTerminal` 运行时判断。

**备选与取舍：**

- *显式按 `ShowTerminal` 分支判断是否隐藏*：多余且易错；结构性排除更稳健。**否决**。

### 决策 4：测试沿用 `process_windows_test.go` 的 Windows 构建标签断言

新增 `app_windows_test.go`：构造 `commandProcess(...)` 后调用 `configureCommandProcess(...)`，断言 `SysProcAttr.HideWindow == true` 且 `CreationFlags & CREATE_NO_WINDOW != 0`，覆盖 `cmd`、PowerShell、直执（`shellPath == ""`）三种包裹形态。

**备选与取舍：**

- *通过注入 `commandStarter` 断言*：注入会替换真实 starter，反而测不到真实标志设置。**否决**。

## Risks / Trade-offs

- **[后台命令自身另开窗口不可控]** → `CREATE_NO_WINDOW` 只作用于 TaskAI 直接启动的子进程；命令若自行 `CREATE_NEW_CONSOLE` 仍可见。与既有 spec "命令自身显式打开窗口" 取舍一致，文档说明此边界。
- **[跨平台测试覆盖]** → 标志仅在 Windows 构建下生效，CI 若在非 Windows 运行则跳过该测试；缓解：`_windows_test.go` 构建标签 + 文档注明，与 lifecycle 既有覆盖策略一致。
- **[行为变化仅限 Windows 后台命令]** → 用户若曾以"看到黑框"确认命令在跑，将不再可见；但"后台 / 不显示终端"本就是用户显式选择无可见终端，故属预期修正而非回归。

## Migration Plan

- 无持久化或数据迁移，纯进程创建属性变更。
- 非 Windows 平台 `configureBackgroundProcess` 为 no-op，行为不变。
- 回滚：移除 `configureCommandProcess` 中的 `configureBackgroundProcess` 调用即恢复原状。

## Open Questions

- 是否需要为后台命令的失败/退出提供某种（无窗口的）反馈渠道？当前已有退出回调（后置脚本），超出本次范围，留作后续增强。
