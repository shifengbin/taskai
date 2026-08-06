# 移除模态弹窗动画 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除共享模态弹窗的进入和退出动画，避免关闭时出现跳位闪框。

**Architecture:** 所有模态弹窗通过 `DialogContent` 和 `DialogOverlay` 共享样式。删除这两个组件上的动画 class 即可覆盖任务、信息管理和设置，无需调整各弹窗的状态管理。

**Tech Stack:** React 18、Radix Dialog、Tailwind CSS、Vitest、Testing Library。

---

### Task 1: 为共享 Dialog 增加无动画回归测试

**Files:**
- Create: `frontend/src/components/ui/dialog.test.tsx`

**Step 1: Write the failing test**

渲染打开状态的 `Dialog`、`DialogContent` 和标题，断言 dialog 内容及遮罩不包含 `animate-in`、`animate-out`、`fade-in-0`、`fade-out-0`、`zoom-in-95`、`zoom-out-95`。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- src/components/ui/dialog.test.tsx`

Expected: FAIL，因为当前共享组件仍含有动画 class。

### Task 2: 移除共享 Dialog 的动画 class

**Files:**
- Modify: `frontend/src/components/ui/dialog.tsx:14-48`
- Test: `frontend/src/components/ui/dialog.test.tsx`

**Step 1: Write minimal implementation**

删除 `DialogOverlay` 和 `DialogContent` 中所有 `data-[state=...]` 动画 class，保留遮罩、居中布局和交互 class。

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm test -- src/components/ui/dialog.test.tsx`

Expected: PASS。

### Task 3: 完整验证

**Files:**
- Verify: `frontend/src/components/ui/dialog.tsx`
- Verify: `frontend/src/components/ui/dialog.test.tsx`

**Step 1: Run frontend checks**

Run: `cd frontend && npm test && npm run build`

Expected: PASS。

**Step 2: Build the application**

Run: `./scripts/build-linux.sh`

Expected: exit code 0 and the Linux application artifact is produced.
