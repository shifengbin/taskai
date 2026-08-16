## Why

在 Windows 10.0.26200 构建上，嵌入终端里 shell（如 cmd）自然退出后，`exited` 事件迟迟不到达或永不到达：前端终端条目一直停在"运行中"，退出提示不出现，异常退出快照也无法生成。用独立探针程序实测定位到根因（见 verification-baseline.md）：该构建的 conhost 在客户端进程退出后**不会关闭 ConPTY 输出管道**，读循环里的阻塞读取永远不会返回；而管理层的 `exited` 事件是在读循环退出后才发布的，于是事件被无限期卡住。进程本身 9ms 内就已退出（`WaitForSingleObject` 正常），问题只出在输出管道的结束方式上。另证实：此前使用的 charmbracelet/x/conpty 库把"关伪控制台"和"关管道/属性表"捆绑在一次 Close 里，在 conhost 尚未冲刷完最终输出时就关掉了读端管道，会丢失退出前的最后一段输出（20 次循环复现），因此不能靠在进程退出后调用库的 Close 来解决。

## What Changes

- Windows 终端后端不再依赖 charmbracelet/x/conpty 库，改为直接管理 ConPTY 生命周期，实现两阶段拆卸：
  - 阶段一（进程退出后）：等待线程取得真实退出码后，仅关闭伪控制台句柄。conhost 随即冲刷剩余输出并结束管道，读循环排空数据后收到"管道已结束"（错误 109）或"操作已中止"（错误 995），这两个错误被识别为预期的读取结束而不是故障。
  - 阶段二（读循环退出后）：才释放属性表、输入/输出管道和进程句柄。保证 conhost 尚未冲刷的最终输出（如退出前最后一条 echo）不丢失。
- 主动关闭（用户关终端/结束任务/应用退出）路径同样按两阶段执行：关伪控制台解除读阻塞，进程尚未退出时终止进程，等读循环收尾（有 2 秒兜底超时）后释放句柄。
- 组装 `CreateProcess` 所需的 UTF-16 环境块（条目以 NUL 分隔、双 NUL 结束、同名键取首个），以 `CREATE_UNICODE_ENVIRONMENT` 传入。
- 移除 go.mod 中的 charmbracelet/x/conpty 依赖。
- Unix 平台、PTY 事件协议、前端会话层与既有退出状态分类逻辑均不变。

## Capabilities

### New Capabilities

- `windows-conpty-lifecycle`: Windows 平台会话的 ConPTY 两阶段拆卸契约——进程退出后关闭伪控制台以结束输出流，读循环排空并退出后才释放句柄资源；自然退出不丢最终输出，主动关闭不悬挂。

### Modified Capabilities

（无。`terminal-exit-status-classification` 的需求语义不变，本变更只是让 Windows 平台会话能按时满足"输出流结束后取得进程真实退出结果"的既有要求。）

## Impact

- `internal/terminal/backend_windows.go`：重写为直接管理 ConPTY 的会话实现（管道创建、属性表、CreateProcess、等待线程、两阶段拆卸）。
- `internal/terminal/backend_windows_test.go`：既有两个 ConPTY 集成测试增加"退出前最终输出不丢失"断言（`TERMINAL_EXIT_SENTINEL`），读取结束错误改用统一的预期错误判定。
- `internal/terminal/read_error_windows_common.go`：预期读取结束错误补充错误 109（ERROR_BROKEN_PIPE）。
- `go.mod` / `go.sum`：移除 charmbracelet/x/conpty。
- 不涉及 Wails 绑定、前端代码或存储格式变更；Unix 平台文件不受影响。
- 集成验证：`wails dev` + chrome-devtools，在真实应用里让终端自然退出并确认 `exited` 事件及时到达（细节见 tasks.md 与 verification-baseline.md）。
