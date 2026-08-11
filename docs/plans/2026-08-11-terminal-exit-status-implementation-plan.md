# 终端退出状态分类实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 正确区分正常与异常终端结束，自动清理正常终端并保留异常终端的只读输出快照。

**Architecture:** 后端把平台进程状态转换为共享的退出结果，终端管理器在受控关闭原因缺失时依据退出码分类。应用层按退出原因投影实时状态，前端只消费该结果：正常结束释放会话，异常结束冻结现有 xterm 会话直到用户关闭。

**Tech Stack:** Go、`os/exec`、ConPTY、React、TypeScript、xterm、Vitest。

---

### Task 1: 终端退出结果与管理器分类

**Files:**
- Modify: `internal/terminal/types.go`
- Modify: `internal/terminal/manager.go`
- Modify: `internal/terminal/manager_test.go`
- Modify: `internal/terminal/backend_unix.go`
- Modify: `internal/terminal/backend_windows.go`
- Modify: `internal/terminal/backend_unix_test.go`
- Modify: `internal/terminal/backend_windows_test.go`

**Step 1: 写失败测试**

让测试会话分别报告退出码 `0`、`1` 和无有效退出码；断言管理器发布 `normal`、`unexpected` 和无伪造退出码。再断言显式关闭即使获得非零结果仍发布 `closed`。

**Step 2: 确认测试失败**

运行：`go test ./internal/terminal -run 'TestManagerPublishes.*Exit' -count=1`

预期：现有实现把所有自然退出归为 `unexpected`，且退出事件没有退出码。

**Step 3: 最小实现**

新增共享退出结果类型，让两个平台后端从 `ProcessState` 提取可用退出码；管理器以受控原因优先、自然零退出码为 `normal` 的规则发布事件。

**Step 4: 确认测试通过**

运行：`go test ./internal/terminal -count=1`

预期：终端包所有测试通过。

### Task 2: 应用层与前端终端条目

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/state.ts`
- Modify: `frontend/src/state.test.ts`

**Step 1: 写失败测试**

为正常退出后的实时状态移除、关闭异常快照后的状态移除，以及按退出原因移除或保留前端条目编写测试。

**Step 2: 确认测试失败**

运行：`go test . -run TestAppMapsTerminalExitReasonsToRealtimeStatus -count=1` 与 `npm test -- --run src/state.test.ts`

预期：正常自然退出仍是异常，前端对全部退出仅标记为 `exited`。

**Step 3: 最小实现**

让应用层在终端管理器成功关闭或已关闭后移除实时投影；扩展前端事件类型并只保留 `unexpected` 条目。

**Step 4: 确认测试通过**

运行：`go test . -run TestAppMapsTerminalExitReasonsToRealtimeStatus -count=1` 与 `npm test -- --run src/state.test.ts`

预期：两组目标测试通过。

### Task 3: 异常输出只读快照

**Files:**
- Modify: `frontend/src/terminal-session.ts`
- Modify: `frontend/src/terminal-session.test.ts`
- Modify: `frontend/src/components/TerminalView.tsx`
- Modify: `frontend/src/components/TerminalView.test.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: 写失败测试**

验证异常退出后已写入的 xterm 会话仍可附着显示，但拒绝输入、粘贴、快捷键、文件拖放和尺寸更新；验证正常退出和手动关闭仍释放会话。

**Step 2: 确认测试失败**

运行：`npm test -- --run src/terminal-session.test.ts src/components/TerminalView.test.tsx`

预期：当前实现会销毁所有退出终端会话。

**Step 3: 最小实现**

将异常退出会话冻结为只读，保留现有 1000 行滚屏；由终端状态限制视图交互，并在手动关闭、结束任务或卸载时统一释放。

**Step 4: 确认测试通过**

运行：`npm test -- --run src/terminal-session.test.ts src/components/TerminalView.test.tsx`

预期：目标前端测试通过。

### Task 4: 规格同步与交付验证

**Files:**
- Modify: `openspec/changes/classify-terminal-exit-status/tasks.md`
- Modify: `openspec/specs/terminal-output-retention/spec.md`
- Create: `openspec/specs/terminal-exit-status-classification/spec.md`

**Step 1: 更新完成状态**

按实施进度标记 OpenSpec 任务，并在归档前将 delta 规格同步到正式规格目录。

**Step 2: 运行完整验证**

运行：`go test -race ./...`、`cd frontend && npm test -- --run`、`cd frontend && npm run build` 和 `openspec validate classify-terminal-exit-status --strict --no-interactive`。

**Step 3: 构建和手动检查**

运行 `scripts` 中的 Linux 构建脚本并启动生成的程序，不设置 `NO_COLOR` 等禁用颜色的环境变量；手动确认 `exit 0` 自动移除且失败命令保留只读输出。
