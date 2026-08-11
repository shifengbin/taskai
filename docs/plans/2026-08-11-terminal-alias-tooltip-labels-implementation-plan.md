# Terminal Alias Tooltip Labels Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为别名终端的两行悬浮提示添加明确的标题和命令标签。

**Architecture:** 保持 `TerminalName` 作为任务树和右侧标题栏的共享提示入口，仅在渲染时为既有实际标题和启动命令值添加固定前缀。别名、终端标题和状态管理数据流不变。

**Tech Stack:** React、TypeScript、Vitest、Testing Library、Wails。

---

### Task 1: 更新变更约定

**Files:**
- Modify: `openspec/changes/add-terminal-alias/design.md`
- Modify: `openspec/changes/add-terminal-alias/specs/terminal-aliases/spec.md`
- Modify: `openspec/changes/add-terminal-alias/tasks.md`

**Step 1: 明确提示文本**

将两行提示约定更新为 `标题: <实际标题>` 与 `命令: <启动命令>`，并新增一个未完成任务记录该显示调整。

**Step 2: 核对状态隔离边界**

确认文档继续声明提示只读展示数据，不能参与 `title-change` 判定。

### Task 2: 先写提示标签的失败测试

**Files:**
- Modify: `frontend/src/components/TerminalName.test.tsx`

**Step 1: 更新断言**

在已有别名悬浮测试中断言提示包含 `标题: npm run dev` 与 `命令: zsh`。

**Step 2: 验证失败**

Run: `cd frontend && npm test -- --run src/components/TerminalName.test.tsx`

Expected: FAIL，因为当前组件仅渲染未带标签的值。

### Task 3: 最小化实现共享提示标签

**Files:**
- Modify: `frontend/src/components/TerminalName.tsx`
- Test: `frontend/src/components/TerminalName.test.tsx`

**Step 1: 渲染标签**

将两行文本分别改为 `标题: {details.actualName}` 和 `命令: {details.command}`，保留原有测试标识、两行结构和仅有别名时显示的条件。

**Step 2: 验证通过**

Run: `cd frontend && npm test -- --run src/components/TerminalName.test.tsx`

Expected: PASS。

### Task 4: 回归验证与应用启动

**Files:**
- Verify: `frontend/src/components/TaskTree.tsx`
- Verify: `frontend/src/components/TerminalView.tsx`

**Step 1: 运行前端全量验证**

Run: `cd frontend && npm test -- --run && npm run build`

Expected: PASS，任务树和右侧标题栏因共用 `TerminalName` 同时获得标签提示。

**Step 2: 构建并启动桌面程序**

Run: `./scripts/build-linux.sh`，恢复构建产生的无关锁文件/绑定副产物后，以独立会话启动 `build/bin/taskai`。

Expected: 应用保持运行，悬浮提示显示两行带标签文本。
