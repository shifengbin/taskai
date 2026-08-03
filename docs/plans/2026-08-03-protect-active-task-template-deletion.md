# Active Task Template Deletion Protection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent deletion of task templates referenced by pending or running tasks.

**Architecture:** Persist the selected task-template ID with each task so the repository can validate deletion against an exact relationship. Reject unsafe settings snapshots atomically on the backend, then expose the same protection in the template-management UI.

**Tech Stack:** Go, Wails, React, TypeScript, Vitest.

---

### Task 1: Persist Task Template References

**Files:**
- Modify: `internal/task/model.go`
- Modify: `internal/task/model_test.go`
- Modify: `internal/lifecycle/service.go`
- Test: `internal/lifecycle/service_test.go`

**Step 1: Write failing tests**

Add tests proving a task created or updated with a selected template persists its template ID together with its field values.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lifecycle ./internal/task`

Expected: failure because tasks do not yet retain a template ID.

**Step 3: Implement the minimal model and service changes**

Add `TaskTemplateID` to `task.Task`, set it when template fields are saved with a current template, and preserve it when no template fields are changed.

**Step 4: Run focused tests**

Run: `go test ./internal/lifecycle ./internal/task`

Expected: PASS.

### Task 2: Reject Unsafe Template Removal

**Files:**
- Modify: `internal/storage/repository.go`
- Test: `internal/storage/repository_test.go`

**Step 1: Write failing tests**

Cover removal of a template used by pending and running tasks, a completed-task reference that remains removable, legacy active tasks with unassociated template fields, and no data mutation after a rejected save.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/storage -run 'TaskTemplate.*Removal|Template.*Deletion'`

Expected: unsafe template removal is accepted.

**Step 3: Implement minimal validation**

Compare previous and proposed template IDs during `SaveSettings`. Reject any removal referenced by pending or running tasks; conservatively reject removals while legacy active tasks carry unassociated template fields.

**Step 4: Run focused tests**

Run: `go test ./internal/storage`

Expected: PASS.

### Task 3: Surface Protection in Settings

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write failing tests**

Cover disabled template deletion for pending/running references, enabled deletion for completed-only references, and the legacy active-task safety message.

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test -- --run src/App.test.tsx -t '任务模板删除保护'`

Expected: delete buttons remain enabled for protected templates.

**Step 3: Implement the minimal UI wiring**

Expose `taskTemplateId` in TypeScript, derive protection state from active tasks, and pass it to `TaskTemplateManagement` to disable unsafe delete controls with an explanatory tooltip.

**Step 4: Run focused tests**

Run: `cd frontend && npm test -- --run src/App.test.tsx -t '任务模板删除保护'`

Expected: PASS.

### Task 4: Verify and Integrate

**Files:**
- Modify: generated Wails bindings only if the added Go field requires regeneration

**Step 1: Run complete verification**

Run: `go test ./...`, `cd frontend && npm test -- --run`, `cd frontend && npm run build`, and `./scripts/build-linux.sh`.

**Step 2: Review and commit**

Run `git diff --check` and `git status --short`; commit implementation separately from plan documentation.
