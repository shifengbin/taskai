# 终端备注操作 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让每个终端会话能选择在复制或发送备注汇总后是否清空，并支持复制完整汇总文本到系统剪贴板。

**Architecture:** 在 `terminal-notes.ts` 维护按会话索引、仅运行期存在的清空偏好，并由 `App` 与备注相同的生命周期路径管理。`TerminalView` 展示三个底部操作组件，使用同一格式化结果进行复制或发送，并依据偏好决定是否调用现有清空回调。

**Tech Stack:** React 18、TypeScript、Radix Checkbox、Wails Runtime、Vitest。

---

### Task 1: 清空偏好的会话状态辅助函数

**Files:**
- Modify: `frontend/src/terminal-notes.ts`
- Test: `frontend/src/terminal-notes.test.ts`

**Step 1: Write the failing test**

测试未设置会话偏好时返回 `true`，更新一个会话不影响另一个会话，清除单个终端和整个任务时删除对应偏好。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run src/terminal-notes.test.ts`

Expected: FAIL，因为清空偏好辅助函数尚不存在。

**Step 3: Write minimal implementation**

定义 `TerminalNoteClearPreferencesBySession` 映射及获取、更新、清除会话和清除任务的纯函数。读取缺失键时返回 `true`，不添加持久化逻辑。

**Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run src/terminal-notes.test.ts`

Expected: PASS。

### Task 2: App 维护偏好生命周期

**Files:**
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing test**

扩展 `TerminalView` mock，验证当前终端接收默认勾选状态和变更回调；终端关闭、退出、任务完成、删除及卸载时复用会话清理路径释放偏好。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: FAIL，因为 App 尚未传递或清理清空偏好。

**Step 3: Write minimal implementation**

增加仅运行期的会话偏好状态，在现有备注清理路径同步清除，并将当前会话的值和更新回调传给 `TerminalView`。

**Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: PASS。

### Task 3: 备注面板的复制和清空控制

**Files:**
- Modify: `frontend/src/components/TerminalView.tsx`
- Test: `frontend/src/components/TerminalView.test.tsx`

**Step 1: Write the failing test**

覆盖默认勾选时复制完整汇总文本并清空；取消勾选时复制和发送都保留备注；同一终端重新打开面板仍显示传入的勾选状态。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run src/components/TerminalView.test.tsx`

Expected: FAIL，因为没有复制按钮、复选框和偏好 props。

**Step 3: Write minimal implementation**

在面板底部添加 `Checkbox`、复制按钮和发送按钮。复制调用 `ClipboardSetText`；两种操作均使用同一 `formatTerminalNotes` 结果，并仅在勾选时调用 `onClearNotes`。失败也遵循勾选状态并报告错误。

**Step 4: Run focused tests and build**

Run: `cd frontend && npm test -- --run src/terminal-notes.test.ts src/App.test.tsx src/components/TerminalView.test.tsx && npm run build`

Expected: PASS。

### Task 4: 同步规格和交付验证

**Files:**
- Modify: `openspec/changes/terminal-notes/design.md`
- Modify: `openspec/changes/terminal-notes/specs/terminal-notes/spec.md`
- Modify: `openspec/changes/terminal-notes/tasks.md`

**Step 1: Update specification**

记录复制完整汇总文本、默认勾选、按会话保存偏好、未勾选保留备注及运行期清理规则。

**Step 2: Run complete verification**

Run: `cd frontend && npm test && npm run build && cd .. && go test -race ./... && openspec validate terminal-notes --strict`

Expected: PASS。

**Step 3: Run and build application**

使用持续运行的 `wails dev -tags webkit2_41` 进行界面检查，再运行 `scripts/build-linux.sh` 并启动生成程序，等待用户确认后再整合分支。
