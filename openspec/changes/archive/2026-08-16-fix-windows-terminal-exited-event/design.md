# 设计：Windows ConPTY 两阶段拆卸

## 上下文

管理层（`internal/terminal/manager.go` 的 `watch()`）在会话读循环退出后才调用 `session.Wait()` 并发布 `exited` 事件。因此"输出流何时结束"直接决定事件何时到达。Windows 26200 构建上 conhost 不主动结束输出管道，读循环永不退出。证据与排除项见 verification-baseline.md。

## 目标

1. shell 自然退出后，`exited` 事件与真实退出码及时到达（亚秒级）。
2. 退出前 conhost 冲刷的最终输出一条不丢。
3. 用户主动关闭（关终端/结束任务/应用退出）不悬挂、不残留句柄。
4. 不改变 Backend 接口、PTY 事件协议与 Unix 平台实现。

## 方案

`backend_windows.go` 直接用 `golang.org/x/sys/windows` 管理 ConPTY 会话（`windowsSession`）：

### 启动（`startWindowsSession`）

1. `CreatePipe` ×2：得到主进程侧 `inWrite`/`outRead` 与 conhost 侧两端。
2. `CreatePseudoConsole(coord, conhostInRead, conhostOutWrite, 0, &hpc)`；conhost 侧两端随即关闭（与微软示例一致，伪控制台已持有引用）。
3. `NewProcThreadAttributeList(1)` + `Update(PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, 句柄值本身, sizeof(Handle))`。注意属性值是句柄值而不是句柄变量地址（见 verification-baseline 的排错记录）；为过 vet 用存储重解释转换。
4. `CreateProcess`：`StartupInfoEx`（`Cb`、`STARTF_USESTDHANDLES`、属性表）+ `EXTENDED_STARTUPINFO_PRESENT | CREATE_UNICODE_ENVIRONMENT`；环境块为手工拼接的 `[]uint16`（条目 NUL 分隔、双 NUL 结束、同名键取首个、`StringToUTF16` 自带条目结尾 NUL 不可再补）。conhost 侧管道句柄不进入子进程（`bInheritHandles=false`）。
5. 关闭主进程持有的线程句柄，保留进程句柄；启动 `watchProcess` 等待线程。

### 等待线程（`watchProcess`）

`WaitForSingleObject(process, INFINITE)` → `GetExitCodeProcess` → 缓存 `ExitResult` → **阶段一：`closePseudoConsole()`（仅关伪控制台）** → `close(waitDone)`。`Wait()` 读缓存结果，不重复等待。

### 读循环结束（`Read`）

`ReadFile(outRead)` 出错时视为读循环终止：标记 `readClosed`，执行**阶段二：`releaseResources()`**（删属性表 + 关 `inWrite`/`outRead`/进程句柄，`sync.Once` 保证只执行一次）。错误 109（管道已结束）与 995（操作已中止）由 `isExpectedWindowsTerminalReadError` 识别为预期结束，管理层照常发布 `exited`。

### 主动关闭（`Close`，`sync.Once`）

关伪控制台（解除读阻塞）→ 进程未退出则 `TerminateProcess(1)`（容忍刚退出的竞态）→ 等 `readClosed`（2 秒兜底超时后强制释放）→ `releaseResources()`。用户关闭场景下 `waitDone` 可能尚未关闭，等待线程随后自然收尾（`closePseudoConsole` 幂等）。

## 取舍

- **不升级库而移除库**：v0.1.0 与 v0.2.0 的 `Close()` 都是 `sync.Once` 捆绑拆卸，无法表达"先关伪控制台、读完后释放其余"的顺序；升级解决不了问题。
- **为什么不轮询进程退出**：轮询只能更早发现退出，结束不了输出管道；且会引入时延与抖动。关闭伪控制台是让 conhost 结束管道的正路。
- **2 秒兜底超时**：只影响"读循环已死循环卡住"的异常场景，正常路径读循环毫秒级排空。超时后释放句柄与既有行为（关闭即释放）一致。
- **退出码语义**：自然退出返回真实退出码；主动关闭由管理层按受控原因（`closed`/`task-ended`）覆盖分类，本层不做特殊处理，符合 terminal-exit-status-classification 既有要求。

## 风险与对策

- 竞态：`ptyClosed`/`resourceClosed`/`readSignaled`/`closeOnce` 各自 `sync.Once`；`waitResult`/`waitErr` 在 `close(waitDone)` 前写入、`Wait()` 在其后读取（happens-before 成立）。
- 资源泄漏：启动路径上任何失败分支都先做阶段二释放（含属性表未建的场景，`attrList` 为 nil 时跳过）。
- 行为回归：既有两个 ConPTY 集成测试保留并加强（目录、参数、退出码、新增 sentinel 不丢断言），20 次循环稳定。
