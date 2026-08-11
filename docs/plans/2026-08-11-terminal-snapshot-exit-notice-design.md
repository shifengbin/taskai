# 终端快照退出提示 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** 在异常终端的冻结 xterm 快照中追加独立的“终端已退出”提示行，并隐藏终端光标。

**Architecture:** `TerminalSessionRegistry` 已负责冻结异常会话并保留解析后的 xterm 状态。冻结时创建或复用会话、写入一次退出提示及 ANSI 私有模式隐藏光标序列，然后标记为只读；提示进入同一滚屏缓冲，不需要新增 React 状态或覆盖层。正常和受控退出继续直接释放会话。

**Tech Stack:** TypeScript、React、xterm.js、Vitest。

---

### Task 1: 覆盖异常快照退出提示

**Files:**
- Modify: `frontend/src/terminal-session.test.ts`
- Modify: `frontend/src/components/TerminalView.test.tsx`

**Step 1: Write the failing test**

在会话注册表测试中，针对异常退出（包括此前没有输出的终端）断言：

- 创建或复用 xterm 会话；
- 追加独立的 `终端已退出` 行；
- 写入 `\x1b[?25l` 隐藏光标序列；
- 重新挂载后会话未被释放，且不会触发后端尺寸更新。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run src/terminal-session.test.ts`

Expected: FAIL，因为当前冻结分支不会添加提示或隐藏光标，且无输出会话无法成为快照。

**Step 3: Write minimal implementation**

在 `TerminalSessionRegistry.handleTerminalEvent` 的保留快照分支中，在设置冻结标记前创建或取得会话；每个终端仅首次退出时向 xterm 写入退出提示和隐藏光标序列。保持现有正常/受控退出释放路径不变。

**Step 4: Run tests to verify it passes**

Run: `cd frontend && npm test -- --run src/terminal-session.test.ts src/components/TerminalView.test.tsx`

Expected: PASS。

### Task 2: 回归验证与运行态复验

**Files:**
- Modify: `openspec/changes/classify-terminal-exit-status/tasks.md`

**Step 1: Run frontend regression suite**

Run: `cd frontend && npm test && npm run build`

Expected: PASS，且生产构建完成。

**Step 2: Build and start Linux application**

Run: `scripts/build-linux.sh && ./build/bin/taskai`

Expected: 构建成功并启动应用；不得设置 `NO_COLOR` 或其他终端颜色禁用变量。

**Step 3: Manual verification**

执行一个有输出的失败命令并查看异常终端：输出尾部显示单独一行 `终端已退出`，没有可见光标；`exit 0` 仍自动移除终端标签。

**Step 4: Commit**

按工作树验收流程，在用户确认后与其余变更一并提交。
