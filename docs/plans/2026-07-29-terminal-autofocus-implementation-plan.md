# Terminal Autofocus Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新建或选中终端后，让右侧 xterm 自动获得键盘焦点，用户无需再次点击即可输入。

**Architecture:** 焦点属于 xterm 实例自身，由 `TerminalView` 在实例挂载到 DOM 后的下一帧调用 `focus()`。`App` 继续只负责终端的选中状态；该状态变化使终端视图切换，从而统一覆盖普通终端、自定义终端与左侧终端项选择。

**Tech Stack:** React、TypeScript、Vitest、Testing Library、xterm.js、Wails。

---

### Task 1: 为终端视图增加自动聚焦回归测试

**Files:**
- Modify: `frontend/src/components/TerminalView.test.tsx`
- Reference: `frontend/src/components/TerminalView.tsx:20-76`

**Step 1: Write the failing test**

在 xterm mock 中暴露 `focus` spy，渲染一个活动终端后执行已排队的动画帧，并断言该实例的 `focus()` 被调用一次。

```tsx
it('挂载活动终端后自动聚焦 xterm 输入区', () => {
  renderTerminalView()
  runAnimationFrame()

  expect(mockTerminal.focus).toHaveBeenCalledOnce()
})
```

**Step 2: Run test to verify it fails**

Run: `npm test -- --run src/components/TerminalView.test.tsx`

Expected: FAIL，提示 `focus` 未被调用。

**Step 3: Commit the failing test only if the repository workflow requires an intermediate commit**

本任务较小，默认不创建红灯提交；保留失败结果后立即进入实现。

### Task 2: 在 xterm 挂载后请求键盘焦点

**Files:**
- Modify: `frontend/src/components/TerminalView.tsx:31-73`
- Test: `frontend/src/components/TerminalView.test.tsx`

**Step 1: Write minimal implementation**

在现有 `requestAnimationFrame` 回调中、尺寸适配之后调用 xterm 的 `focus()`，并确保清理时取消尚未执行的动画帧。

```tsx
const animationFrame = requestAnimationFrame(() => {
  fit()
  instance.focus()
})

return () => {
  cancelAnimationFrame(animationFrame)
  // 现有资源清理
}
```

**Step 2: Run focused test to verify it passes**

Run: `npm test -- --run src/components/TerminalView.test.tsx`

Expected: PASS，包含新增自动聚焦断言。

**Step 3: Run relevant integration tests**

Run: `npm test -- --run src/App.test.tsx`

Expected: PASS，右键创建和自定义终端既有行为未回归。

**Step 4: Commit implementation**

```bash
git add frontend/src/components/TerminalView.tsx frontend/src/components/TerminalView.test.tsx
git commit -m "fix: 自动聚焦新建和选中的终端"
```

### Task 3: 完整验证

**Files:**
- Verify only

**Step 1: Run frontend test suite**

Run: `npm test -- --run`

Expected: PASS。

**Step 2: Run project compilation**

Run: `./scripts/build-linux.sh`

Expected: 构建成功并输出 `build/bin/taskai`。

**Step 3: Check the worktree**

Run: `git diff --check && git status --short`

Expected: 无空白错误，且仅包含本计划预期的修改或提交后为空。

