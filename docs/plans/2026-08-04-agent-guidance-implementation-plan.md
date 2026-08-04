# Agent Guidance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a repository-root `AGENTS.md` that gives coding agents accurate development guidance and preserves TaskAI's lifecycle, cross-platform terminal, and OpenSpec constraints.

**Architecture:** A single Markdown file at the repository root applies to all code. It points to the existing source ownership boundaries instead of duplicating product specifications, while elevating the behavior that must remain stable during implementation work.

**Tech Stack:** Markdown, Go, Wails v2, React, TypeScript, Vitest, OpenSpec.

---

### Task 1: Add repository agent guidance

**Files:**
- Create: `AGENTS.md`
- Reference: `README.md`
- Reference: `go.mod`
- Reference: `frontend/package.json`
- Reference: `openspec/specs/`

**Step 1: Draft the guidance structure**

Create a concise root-level Markdown document with these sections:

```markdown
# TaskAI Agent Guide

## Repository Layout
## Development and Verification
## Change Boundaries
## Lifecycle Command Rules
## Cross-Platform Terminal Rules
## OpenSpec Workflow
```

The document must identify the backend, frontend, storage, lifecycle, terminal, and specification directories.

**Step 2: Add project-specific invariants**

Document these rules in `AGENTS.md`:

- Lifecycle hook ordering, failure semantics, retry semantics, and persisted-versus-runtime state must remain consistent with the existing specifications.
- Built-in workspace and Git commands retain their path validation and non-destructive behavior.
- Shared terminal and process contracts must preserve Unix/Windows platform-file separation.
- Regenerate Wails bindings after changing exported application methods or types.
- Product behavior changes synchronize the applicable OpenSpec specification; substantial work follows the proposal, design, and tasks workflow.

**Step 3: Add exact verification commands**

Include the repository's supported commands without inventing new tooling:

```bash
go test -race ./...
cd frontend && npm test && npm run build
```

State that focused tests should run first and cross-layer changes require the complete command set.

**Step 4: Validate the document**

Run:

```bash
test -f AGENTS.md
rg -n '^## |go test -race ./\.\.\.|npm test && npm run build|openspec/specs|wails generate module' AGENTS.md
git diff --check
```

Expected: The file exists, the required sections and commands are present, and `git diff --check` produces no output.

**Step 5: Commit the guidance**

```bash
git add AGENTS.md docs/plans/2026-08-04-agent-guidance-implementation-plan.md
git commit -m "docs: add agent guidance"
```
