# Agent Guidance Design

## Goal

Add a repository-root `AGENTS.md` that gives coding agents the project context and protects the domain rules most likely to regress during routine changes.

## Scope

The guidance applies to the entire `taskai` repository. It documents the Go/Wails backend, the React/Vite frontend, test commands, and the existing OpenSpec documentation workflow. It does not change application behavior, replace specifications, or add nested instruction files.

## Structure

The root file will contain five concise sections:

1. Repository layout and primary commands.
2. General change rules, including generated Wails bindings and focused verification.
3. Lifecycle command-chain invariants: hook ordering, commit and failure semantics, retry behavior, and the separation of persisted task state from runtime execution state.
4. Cross-platform terminal and process rules: preserve Unix/Windows build-tag separation and validate the relevant platform implementations when changing shared contracts.
5. OpenSpec synchronization: behavior changes update the matching `openspec/specs` document; substantial changes use the repository's proposal, design, and tasks workflow.

## Rationale

A single root-level guide matches the repository's current size and avoids duplicated instructions. The lifecycle and PTY areas have platform and state-machine behavior that ordinary build instructions do not communicate, while OpenSpec rules keep code and product contracts aligned.

## Verification

Review the file for correct paths and commands, confirm it has no conflicting nested `AGENTS.md`, and use it as the source of instruction discovery for subsequent work. No application tests are required because this change only adds contributor guidance.
