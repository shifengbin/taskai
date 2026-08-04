# TaskAI Agent Guide

## Repository Layout

- `main.go` starts Wails; `app.go` owns exported Wails bindings and application orchestration.
- `internal/lifecycle/` manages task transitions and lifecycle command chains. `internal/storage/` persists tasks and settings; `internal/settings/` defines and validates configuration.
- `internal/terminal/` owns PTY sessions and platform process integration. `internal/workspace/` owns task workspace paths and filesystem operations.
- `frontend/src/` contains the React 18, Vite, and MUI user interface. Generated Wails frontend bindings live in `frontend/wailsjs/`.
- `openspec/specs/` contains executable behavior specifications. `docs/plans/` contains approved design and implementation records.

## Development and Verification

Run the smallest relevant test first (an affected Go package or frontend test file). For cross-layer changes, or before handing off a broad change, run the full verification set:

```sh
go test -race ./...
cd frontend && npm test && npm run build
```

When changing exported Wails application methods or bound Go types, regenerate bindings and review the generated frontend surface:

```sh
wails generate module
git diff -- frontend/wailsjs/
```

## Change Boundaries

- Keep changes in the owning layer; preserve existing validation and error handling at API boundaries.
- Built-in workspace and Git commands must retain path validation, workspace working-directory boundaries, and non-destructive behavior. Do not bypass their checks or add unreviewed destructive filesystem or Git operations.
- Treat persisted task data as the source of truth. Do not repurpose temporary runtime state as persisted task state.

## Lifecycle Command Rules

- Preserve the five hooks and their semantics: `beforeStart`, `postStart`, `beforeEnd`, `postEnd`, and `updateTask`.
- Preserve ordering and commit points: `beforeStart` runs before the start state is committed, then `postStart`; `beforeEnd` runs before completion is committed, then `postEnd`. `updateTask` runs on its dedicated update flow after the task update is persisted.
- Do not change failure, status-commit, or retry behavior incidentally. A failed pre-transition hook must not advance its task transition; retries must remain valid only for the hook and task status recorded by the execution.
- Keep durable task status and lifecycle-chain configuration distinct from transient lifecycle execution state, including its run ID, revision, progress, failure, and retry information.

## Cross-Platform Terminal Rules

- Keep Windows and Unix PTY/process implementations in their platform-specific files and build constraints.
- When changing a shared terminal or process contract, inspect and test the corresponding Windows and Unix implementations. Do not introduce an OS-specific path, shell, signal, or process assumption into shared code.

## OpenSpec Workflow

- Update the matching specification under `openspec/specs/` whenever user-visible behavior changes; code and specifications must not conflict.
- For substantial features or behavior changes, follow the existing OpenSpec proposal, design, and tasks workflow before and during implementation.
- Keep approved implementation records in `docs/plans/` aligned with the resulting change when the workflow calls for them.
