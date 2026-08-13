# Menu Command Terminal Mouse Clipboard Implementation Plan

> 状态：已由 `2026-08-13-unify-terminal-mouse-selection-implementation.md` 的统一终端鼠标策略取代，旧字段仅作为历史实现记录保留。

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to disable TaskAI's automatic selection copy and right-click clipboard paste for individual display-terminal menu commands, returning mouse handling to the terminal program while preserving existing behavior everywhere else.

**Architecture:** Persist a terminal-only policy on each custom task-menu command, copy it into the terminal record when the command starts, and treat that record as an immutable per-session behavior snapshot. The frontend settings editor exposes the policy only for commands configured to show a terminal; the terminal view and terminal session registry consume the snapshot independently for context-menu and selection handling.

**Tech Stack:** Go, Wails v2 bindings, React, TypeScript, Vitest, xterm.js, OpenSpec.

---

## Scope and Behavior

- The setting belongs to a user-created task-menu command and defaults to `false`.
- It is available only when the command uses `showTerminal`.
- When enabled, TaskAI does not write selected text to the system clipboard and does not intercept right-click to inject clipboard text.
- Keyboard paste and all terminal-program mouse behavior remain untouched.
- Existing and newly created terminals without the setting retain TaskAI's current selection-copy and right-click-paste behavior.
- A terminal receives its policy when it starts. Editing the menu command later affects only future terminals.

## Task 1: Persist the Per-Command Policy

**Files:**
- Modify: `internal/settings/settings.go`
- Modify: `internal/settings/settings_test.go`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/types.test.ts`

### Step 1: Add failing settings tests

Add table-driven coverage for custom menu commands that verifies:

- a terminal-visible command preserves `disableTaskAIMouseClipboard: true` through validation and save/load;
- an omitted field from an existing configuration evaluates to `false`;
- a command without `showTerminal` cannot retain an enabled value after normalization;
- fixed menu items remain unaffected.

Run:

```bash
go test ./internal/settings
```

Expected: FAIL because the setting model does not yet contain the policy field.

### Step 2: Add the settings field and normalize its applicability

Extend `settings.TaskMenuItem` with the JSON field `disableTaskAIMouseClipboard`. During task-menu normalization, retain it only for valid custom commands with `showTerminal: true`; normalize it to `false` for non-terminal commands. Do not add a migration version: Go's zero value makes previously saved configurations default safely to the current behavior.

Mirror the field in the frontend `TaskMenuItem` type. Keep it optional at the frontend boundary so previously generated payloads and test fixtures remain compatible; UI and terminal behavior must enable it only for the literal value `true`.

### Step 3: Verify the persistence contract

Run:

```bash
go test ./internal/settings
cd frontend && npm test -- --run src/types.test.ts
```

Expected: PASS.

### Step 4: Commit the settings model

```bash
git add internal/settings/settings.go internal/settings/settings_test.go frontend/src/types.ts frontend/src/types.test.ts
git commit -m "feat: persist terminal mouse clipboard policy"
```

## Task 2: Carry the Policy Into Each Terminal Session

**Files:**
- Modify: `internal/terminal/types.go`
- Modify: `internal/terminal/manager.go`
- Modify: `internal/terminal/manager_test.go`
- Modify: `app.go`
- Modify: `app_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`

### Step 1: Add failing terminal creation tests

Add manager/app tests using a captured terminal start request. Cover these cases:

- executing a display-terminal menu command with the option enabled starts a terminal and returns `terminal.Info` with the policy enabled;
- a display-terminal command without the option, direct command terminals, and background menu commands have the policy disabled;
- the policy is passed when a terminal is created, rather than looked up from mutable settings afterward.

Run:

```bash
go test . ./internal/terminal
```

Expected: FAIL because terminal start requests and terminal info do not yet expose the policy.

### Step 2: Extend terminal creation data deliberately

Add `DisableTaskAIMouseClipboard` to the terminal request/record data returned to the frontend. Extend the command-terminal creation path with a small typed option or explicit parameter so `app.ExecuteTaskMenuCommand` passes the saved menu-command value at creation time. Leave all non-menu creation paths on the zero-value policy.

Do not have frontend terminal behavior query settings by task ID: that would let editing a menu item mutate a terminal that is already running.

### Step 3: Regenerate Wails type bindings and prove backend propagation

Run:

```bash
wails generate module
go test . ./internal/terminal
```

Expected: PASS. Review the generated terminal model to confirm it includes the policy field with the expected JSON name.

### Step 4: Commit terminal policy propagation

```bash
git add internal/terminal/types.go internal/terminal/manager.go internal/terminal/manager_test.go app.go app_test.go frontend/wailsjs/go/models.ts
git commit -m "feat: pass menu clipboard policy to terminals"
```

## Task 3: Expose the User Setting in Menu Management

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

### Step 1: Add failing menu-editor tests

In the menu management settings tests, add coverage that:

- the mouse-clipboard switch appears for a custom command only after `showTerminal` is enabled;
- enabling the switch and saving sends `disableTaskAIMouseClipboard: true` in `SaveSettings`;
- turning off terminal display clears or leaves inaccessible the setting so a non-terminal command cannot be configured with it.

Run:

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

Expected: FAIL because the menu editor does not render or save the option.

### Step 2: Add a conditional switch beside the terminal-display option

Update the custom command draft editor in `App.tsx` with a `SnapSwitch` bound to `disableTaskAIMouseClipboard`. Use a concise label, `禁用 TaskAI 鼠标复制与右键粘贴`. Render it only for terminal-visible commands and set it to `false` when the user turns off `showTerminal`.

New custom commands must initialize the field to `false`; existing draft cloning and save paths must preserve an enabled value. Do not show this control for fixed menu entries or non-terminal commands.

### Step 3: Verify settings UI persistence

Run:

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

Expected: PASS.

### Step 4: Commit the menu-editor control

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: configure terminal mouse clipboard handling"
```

## Task 4: Respect the Policy in Terminal Mouse Handling

**Files:**
- Modify: `frontend/src/components/TerminalView.tsx`
- Modify: `frontend/src/components/TerminalView.test.tsx`
- Modify: `frontend/src/terminal-session.ts`
- Modify: `frontend/src/terminal-session.test.ts`

### Step 1: Add failing terminal interaction tests

Add cases for a terminal record with `disableTaskAIMouseClipboard: true` that verify:

- an xterm selection does not call `ClipboardSetText`;
- right-click does not call `preventDefault`, does not read the clipboard, and does not write input to the terminal;
- the same interactions for a terminal without the flag preserve the existing copy/paste behavior.

Run:

```bash
cd frontend && npm test -- --run src/components/TerminalView.test.tsx src/terminal-session.test.ts
```

Expected: FAIL because both handlers currently run unconditionally.

### Step 2: Gate the two TaskAI-owned handlers on the terminal snapshot

Pass the terminal's policy into the terminal session initialization and have the selection-change callback skip automatic clipboard writes when it is enabled. In `TerminalView`, omit the TaskAI `onContextMenu` handler when the policy is enabled so the event reaches xterm/the terminal program without a forced clipboard read or input write.

Do not change keyboard paste, drag-and-drop, terminal output, or session reconnect behavior. Use `terminal.disableTaskAIMouseClipboard === true` rather than truthiness so absent fields keep current behavior.

### Step 3: Run focused frontend tests

Run:

```bash
cd frontend && npm test -- --run src/components/TerminalView.test.tsx src/terminal-session.test.ts
```

Expected: PASS.

### Step 4: Commit terminal interaction behavior

```bash
git add frontend/src/components/TerminalView.tsx frontend/src/components/TerminalView.test.tsx frontend/src/terminal-session.ts frontend/src/terminal-session.test.ts
git commit -m "feat: allow terminals to own mouse clipboard actions"
```

## Task 5: Document the Contract and Run Full Verification

**Files:**
- Create: `openspec/specs/terminal-mouse-clipboard-policy/spec.md`
- Modify as generated: `frontend/wailsjs/go/models.ts`

### Step 1: Add the OpenSpec requirement

Document the following scenarios in Chinese:

- terminal-visible custom commands can persist the opt-out, while omitted configuration defaults to enabled TaskAI behavior;
- each newly created terminal snapshots the command policy and later setting changes do not alter it;
- opted-out terminals do not auto-copy selections or intercept right-click, while normal terminals retain both behaviors;
- non-terminal and fixed commands cannot enable the terminal-only behavior.

### Step 2: Run the complete test and build suite

Run:

```bash
go test -race ./...
cd frontend && npm test
cd frontend && npm run build
./scripts/build-linux.sh
git diff --check
git status --short
```

Expected: all commands exit successfully, generated bindings are current, and the only remaining changes are the OpenSpec specification (plus any intentionally regenerated binding that has not already been committed).

### Step 3: Commit documentation and generated artifacts

```bash
git add openspec/specs/terminal-mouse-clipboard-policy/spec.md frontend/wailsjs/go/models.ts
git commit -m "docs: specify terminal mouse clipboard policy"
```

## Final Review Checklist

- Check the setting is unchecked by default on a newly created terminal menu command.
- Start one terminal with the setting on and another with it off; confirm each retains its own behavior after editing the menu command.
- Confirm a right-click on the opted-out Claude terminal is no longer consumed by TaskAI.
- Confirm keyboard paste continues to work in both terminals.
- Verify no unrelated clipboard path was changed.
