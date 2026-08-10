# 终端快捷键按键动作 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将终端快捷键动作扩展为可录入的文本或按键，并按标准终端序列执行。

**Architecture:** Go 设置模型保存 `text` 与 `key` 两种动作，读取时兼容旧 `enter`；前端用一次 `keydown` 捕获按键和修饰键，终端会话层把规范化按键映射为直接 PTY 输入。设置草稿继续由 App 管理，快捷输入选择器保留优先级。

**Tech Stack:** Go、React 18、TypeScript、xterm.js、Vitest、Wails bindings。

---

### Task 1: 扩展动作模型并保留旧数据

**Files:**
- Modify: `internal/settings/settings.go`
- Test: `internal/settings/terminal_shortcut_test.go`
- Modify: `internal/storage/repository.go`

**Steps:**
1. Write failing tests for `key` actions, modifiers, legacy `enter` migration, and invalid key data.
2. Run `go test ./internal/settings ./internal/storage` and verify the new tests fail.
3. Add the `key` step fields and normalize legacy `enter` into `key: Enter`.
4. Run the targeted Go tests and verify they pass.

### Task 2: Map captured keys to PTY input

**Files:**
- Modify: `frontend/src/terminal-shortcuts.ts`
- Modify: `frontend/src/terminal-session.ts`
- Modify: `frontend/src/components/TerminalView.tsx`
- Test: `frontend/src/terminal-shortcuts.test.ts`
- Test: `frontend/src/components/TerminalView.test.tsx`

**Steps:**
1. Write failing tests for key normalization and common xterm input sequences.
2. Implement printable, control, navigation, function-key, and Alt mappings.
3. Send each configured step directly and stop/report when the session closes.
4. Run the focused frontend tests and verify they pass.

### Task 3: Update the settings editor and bindings

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/components/TerminalShortcutSettings.tsx`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`
- Modify: `frontend/wailsjs/go/models.ts`

**Steps:**
1. Write failing UI tests for text/key action switching and key capture.
2. Replace the fixed Enter action selector with a text/key editor and capture control.
3. Regenerate Wails bindings and update frontend type conversions.
4. Run App and component tests and verify they pass.

### Task 4: Verify and update documentation

**Files:**
- Modify: `openspec/changes/configurable-terminal-shortcuts-and-window-state/specs/terminal-shortcut-inputs/spec.md`
- Modify: `openspec/changes/configurable-terminal-shortcuts-and-window-state/design.md`
- Modify: `openspec/changes/configurable-terminal-shortcuts-and-window-state/tasks.md`

**Steps:**
1. Update the OpenSpec requirements and migration notes for key actions.
2. Mark the implementation task complete after tests pass.
3. Run strict OpenSpec validation.
4. Run `go test -race ./...`, full frontend tests, and frontend build.
