## 1. Settings model and migration

- [x] 1.1 Add lifecycle preset settings types, default seed data, deep-copy/default resolution helpers, and normalization validation.
- [x] 1.2 Migrate legacy `lifecycleDefaultChains` to a default preset exactly once and use the resolved preset when normalizing old tasks.
- [x] 1.3 Add focused settings and repository migration tests for defaults, validation, explicit empty maps, and status-aware legacy tasks.

## 2. Repository preset management and referential integrity

- [x] 2.1 Add dedicated lifecycle preset list, save, copy, delete, and default-selection repository operations.
- [x] 2.2 Protect command-chain deletion and applicability changes when a lifecycle preset references the chain.
- [x] 2.3 Add repository tests for CRUD isolation, default clearing, validation, task independence, and reference protection.

## 3. Task creation and application bindings

- [x] 3.1 Resolve default lifecycle presets for implicit task creation while preserving explicit empty chain selections.
- [x] 3.2 Expose lifecycle preset operations through application contracts and Wails bindings, removing per-hook default-chain binding.
- [x] 3.3 Add focused service and application binding tests.

## 4. Frontend data contract

- [x] 4.1 Add lifecycle preset types and settings conversion.
- [x] 4.2 Add preset API wrappers, remove the per-hook default API, and keep ordinary settings saves from overwriting lifecycle preset fields.
- [x] 4.3 Add focused type and API tests.

## 5. Preset management and task dialog UI

- [x] 5.1 Replace settings per-hook default controls with preset listing, editor, copy/delete, and default-selection interactions.
- [x] 5.2 Add task preset selection, whole-map apply, custom-state detection, and explicit-empty task creation handling.
- [x] 5.3 Add UI tests for preset management, task initialization, custom adjustment, empty selection, and read-only task states.

## 6. Documentation and verification

- [x] 6.1 Update README and baseline specifications to describe lifecycle presets instead of per-hook default chains.
- [x] 6.2 Run Go tests, frontend tests and build, OpenSpec strict validation, formatting/diff checks, and resolve any failures.
