# 固定两行任务卡片实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将顶层任务卡片固定为 60px 高的两行排版，并精确采用 Snap 参考的标题与描述字体、字号和间距。

**Architecture:** 在 `TaskTree` 中始终渲染描述行，空值时显示“暂无描述”；通过稳定的高度和两行文本样式保持操作区位置不变。只修改任务卡片，终端子项及状态逻辑不变。

**Tech Stack:** React 18、TypeScript、Tailwind CSS、Vitest、Testing Library。

---

### Task 1: 任务卡片两行回归测试

**Files:**
- Modify: `frontend/src/components/TaskTree.test.tsx`

**Step 1: Write the failing test**

新增有描述与无描述任务的渲染断言：任务行具有 `h-[60px]`，名称和描述均为单行文本，无描述显示“暂无描述”，且样式包含 Hanken 13.5px/800、Plus Jakarta 11.5px/500 和 1px 间距。

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- src/components/TaskTree.test.tsx`

Expected: FAIL，因为当前描述为空时不会渲染且任务行只有最小高度。

### Task 2: 固定两行任务正文

**Files:**
- Modify: `frontend/src/components/TaskTree.tsx:397-475`

**Step 1: Implement the minimal code**

任务卡片改为 `h-[60px]`；标题使用 `font-display`、`text-[13.5px]`、`font-extrabold`，描述使用 `font-sans`、`text-[11.5px]`、`font-medium` 与 `mt-px`。无描述时渲染“暂无描述”，提示文案使用同一缺省值。

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm test -- src/components/TaskTree.test.tsx`

Expected: PASS。

### Task 3: 完整验证

**Files:**
- Modify: `openspec/changes/restore-snap-task-list-cards/tasks.md`

**Step 1: Run validation**

Run: `cd frontend && npm test && npm run build`

Run: `openspec validate restore-snap-task-list-cards --strict`

Run: `scripts/build-linux.sh`

**Step 2: Record completion**

将 OpenSpec 第 5 节的任务标记为完成。
