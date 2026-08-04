# Lifecycle Command Presets Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace per-hook lifecycle default chains with reusable, named lifecycle presets and let new tasks apply a default or selected preset before individual adjustment.

**Architecture:** A lifecycle preset is a settings-owned map from lifecycle hooks to command-chain IDs. Tasks continue to persist only their resolved map, so presets are batch-fill configuration rather than task references. The repository owns preset validation, migration, CRUD, and command-chain reference protection; the React form owns the temporary selected-preset state.

**Tech Stack:** Go, Wails, React, TypeScript, Material UI, Vitest, Go testing.

---

### Task 1: Add the lifecycle preset settings model and one-time migration

**Files:**
- Modify: `internal/settings/settings.go`
- Modify: `internal/settings/settings_test.go`
- Modify: `internal/storage/repository.go`
- Modify: `internal/storage/repository_test.go`

**Step 1: Write failing defaults, normalization, and migration tests**

Add coverage for these cases:

- `settings.Default()` exposes one stable-ID “默认预设”, selects it as `DefaultLifecyclePresetID`, and its map preserves the existing `beforeStart` and `postEnd` behavior.
- Preset names are trimmed and must be non-empty and case-insensitively unique; every non-empty selected chain exists and covers its hook; an empty map is valid.
- A default preset ID must refer to an existing preset, while an empty default ID is valid.
- A persisted v4 settings record with `lifecycleDefaultChains` becomes one default preset, preserves the original map including an explicit empty map, clears the legacy field before persisting, and does not recreate a later-deleted migrated preset.
- Legacy tasks without `lifecycleChains` receive the resolved default-preset map using the existing pending/running/completed status rules.

Use a representative expected value in the tests:

```go
want := settings.LifecyclePreset{
    ID:   settings.DefaultLifecyclePresetID,
    Name: "默认预设",
    Chains: map[task.LifecycleHook]string{
        task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
        task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
    },
}
```

**Step 2: Run the focused Go tests and verify they fail**

Run:

```bash
go test ./internal/settings ./internal/storage -run 'Test(DefaultIncludesLifecyclePreset|NormalizeLifecyclePresets|RepositoryMigratesLifecycleDefaultChainsToPreset)'
```

Expected: FAIL because `LifecyclePreset`, `DefaultLifecyclePresetID`, and migration behavior do not yet exist.

**Step 3: Implement the settings model and migration**

In `internal/settings/settings.go`:

- Add `LifecyclePreset { ID, Name, Chains }`, `LifecyclePresets`, and `DefaultLifecyclePresetID`.
- Rename the old JSON-backed `LifecycleDefaultChains` member to a clearly legacy-only compatibility field while retaining `json:"lifecycleDefaultChains,omitempty"`; do not use it for runtime defaults.
- Add stable default-preset constants and have `Default()` seed the standard workspace-create/workspace-delete preset.
- Normalize presets after command chains: deep-copy maps, trim IDs/names, reject duplicate IDs or names, and validate each chain reference and hook applicability.
- Add a safe helper returning a copy of the current default preset's map; this replaces direct reads of the legacy map.
- Advance `CurrentPresetVersion` and extend `ApplyPresetMigration` to create the default preset from the legacy map exactly once, then clear the legacy field.

In `internal/storage/repository.go`:

- Resolve the default preset map before calling `normalizeTaskLifecycle` for old tasks.
- Ensure migration runs before final preset validation and is persisted by the existing normalization write-back path.

**Step 4: Run the focused Go tests and verify they pass**

Run the command from Step 2. Expected: PASS.

**Step 5: Commit the completed model and migration slice**

```bash
git add internal/settings/settings.go internal/settings/settings_test.go internal/storage/repository.go internal/storage/repository_test.go
git commit -m "feat: add lifecycle preset settings"
```

### Task 2: Add preset repository operations and protect preset chain references

**Files:**
- Modify: `internal/storage/repository.go`
- Modify: `internal/storage/repository_test.go`

**Step 1: Write failing repository tests**

Add tests for:

- Saving a preset assigns an ID, normalizes its values, and listing returns an independent snapshot.
- Updating, copying, deleting, and setting/clearing the default preset preserve unrelated settings.
- Deleting the current default preset clears its default ID.
- Saving a preset rejects a missing chain or a chain that does not cover the selected hook.
- Deleting a chain referenced by a preset fails, even when only completed tasks reference it.
- Editing a chain to remove a hook referenced by a preset fails.
- Editing or deleting a preset does not alter a previously saved task's `LifecycleChains`.

**Step 2: Run the focused repository tests and verify they fail**

Run:

```bash
go test ./internal/storage -run 'TestRepository(ManagesLifecyclePresets|ProtectsLifecyclePresetChainReferences)'
```

Expected: FAIL because the preset repository operations and protections are absent.

**Step 3: Implement dedicated preset operations**

Add repository methods parallel to the existing chain operations:

```go
ListLifecyclePresets() ([]settings.LifecyclePreset, error)
SaveLifecyclePreset(settings.LifecyclePreset) (settings.LifecyclePreset, error)
CopyLifecyclePreset(id string) (settings.LifecyclePreset, error)
DeleteLifecyclePreset(id string) error
SaveDefaultLifecyclePreset(id string) (settings.Settings, error)
```

Requirements for the implementation:

- Generate preset IDs with the existing lifecycle ID helper using a distinct `preset` kind.
- Make deep copies of each preset's `Chains` map when returning or copying data.
- Reuse `settings.NormalizeLifecycle` after every write, so references are always checked against the current command-chain collection.
- Update `SaveLifecycleCommandChain` and `DeleteLifecycleCommandChain` to reject changes that invalidate any preset reference.
- Delete `SaveLifecycleDefaultChain`; it represents the removed per-hook default behavior.

**Step 4: Run the focused repository tests and verify they pass**

Run the command from Step 2. Expected: PASS.

**Step 5: Commit the repository slice**

```bash
git add internal/storage/repository.go internal/storage/repository_test.go
git commit -m "feat: manage lifecycle command presets"
```

### Task 3: Resolve default presets in task creation and expose preset bindings

**Files:**
- Modify: `internal/lifecycle/service.go`
- Modify: `internal/lifecycle/service_test.go`
- Modify: `internal/application/contracts.go`
- Modify: `app.go`
- Modify: `app_test.go`
- Regenerate: `frontend/wailsjs/go/main/App.js`
- Regenerate: `frontend/wailsjs/go/main/App.d.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Step 1: Write failing service and binding tests**

Cover these contracts:

- Existing creation entry points without an explicit chain map resolve the current default preset.
- Explicit creation with `{}` persists no selections and never falls back to the default preset.
- Updating or deleting a preset leaves an existing task's map unchanged.
- The application exposes the five preset operations and no longer exposes `SaveLifecycleDefaultChain`.

**Step 2: Run the focused Go tests and verify they fail**

Run:

```bash
go test ./internal/lifecycle . -run 'Test(ServiceCreatesTaskFromDefaultLifecyclePreset|ServicePersistsExplicitEmptyLifecycleChains|AppExposesLifecyclePresetBindings)'
```

Expected: FAIL because default resolution still reads `LifecycleDefaultChains` and bindings do not exist.

**Step 3: Implement task-default resolution and bindings**

- Replace `copyLifecycleChains(data.Settings.LifecycleDefaultChains)` in `createTask` with the safe default-preset map helper.
- Preserve the existing `useDefaults` distinction so an explicit empty map remains a valid user choice.
- Add the repository calls to `App` and expose the new methods through the lifecycle configuration contract.
- Remove the per-hook default binding and regenerate Wails bindings from the application root:

```bash
wails generate module
```

**Step 4: Run the focused Go tests and verify they pass**

Run the command from Step 2. Expected: PASS.

**Step 5: Commit the task-resolution and binding slice**

```bash
git add internal/lifecycle/service.go internal/lifecycle/service_test.go internal/application/contracts.go app.go app_test.go frontend/wailsjs/go/main/App.js frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/models.ts
git commit -m "feat: apply default lifecycle presets to new tasks"
```

### Task 4: Add frontend preset types and API wrappers

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/types.test.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/api.test.ts`

**Step 1: Write failing TypeScript/API tests**

- Verify Wails settings conversion preserves `lifecyclePresets` and `defaultLifecyclePresetId`.
- Verify the API exposes list/save/copy/delete/default-preset operations and passes correctly converted values to Wails.
- Update the normal-settings test to assert that preset fields, like commands and chains, are stripped from `SaveSettings` payloads.

**Step 2: Run the focused frontend tests and verify they fail**

Run:

```bash
npm test -- --run src/types.test.ts src/api.test.ts
```

Expected: FAIL because the frontend has neither preset types nor API wrappers.

**Step 3: Implement the frontend contract**

- Add `LifecyclePreset` and the two preset fields to `SettingsRecord`.
- Add API wrappers for all generated preset bindings and remove `saveLifecycleDefaultChain`.
- Strip `lifecyclePresets` and `defaultLifecyclePresetId` from ordinary settings saves.
- Use the generated `settings.LifecyclePreset.createFrom` method for preset write requests.

**Step 4: Run the focused frontend tests and verify they pass**

Run the command from Step 2. Expected: PASS.

**Step 5: Commit the frontend API slice**

```bash
git add frontend/src/types.ts frontend/src/types.test.ts frontend/src/api.ts frontend/src/api.test.ts
git commit -m "feat: expose lifecycle preset frontend api"
```

### Task 5: Replace the default-chain controls with preset management and task selection

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

**Step 1: Write failing task-dialog and settings tests**

Add interaction tests proving that:

- The settings lifecycle tab lists presets, opens a preset editor, creates/duplicates/deletes a preset, and marks one as default; it no longer renders five per-hook default controls.
- Opening a new task initializes all hook selections from the default preset.
- Selecting another preset replaces every hook, including clearing hooks omitted by the new preset.
- Changing one hook changes the preset control to “自定义”.
- Clearing every hook and creating the task invokes the explicit lifecycle-chain create binding with `{}`, rather than a no-chain/default-resolving binding.
- Editing a pending task displays a matching current preset or “自定义”; running and completed tasks keep both preset and hook controls disabled.

**Step 2: Run the focused UI tests and verify they fail**

Run:

```bash
npm test -- --run src/App.test.tsx
```

Expected: FAIL because the UI only has per-hook defaults and no preset selector/editor.

**Step 3: Implement the UI in small pieces**

1. Add local task-dialog state for the selected preset ID and helpers that clone maps and compare all five hooks without mutating settings data.
2. Initialize a new dialog from `settings.defaultLifecyclePresetId`; when a preset is selected, replace `taskLifecycleChainsDraft` with a cloned preset map.
3. Have an individual hook edit clear the selected preset ID and render “自定义”; always call the explicit lifecycle-chain create binding when saving a new task.
4. Add a preset selector before `TaskLifecycleChainSelector`, honoring the existing pending/running/completed disabled rules.
5. Replace the `LifecycleManagement` default-chain grid with a preset summary list, default badge/action, and a name-plus-five-hook modal. Use the existing chain applicability filter for each hook selector.
6. Refresh presets and default selection alongside commands and chains after every lifecycle-configuration write.

**Step 4: Run the focused UI tests and verify they pass**

Run the command from Step 2. Expected: PASS.

**Step 5: Commit the UI slice**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: manage and apply lifecycle presets"
```

### Task 6: Update product specifications and run full verification

**Files:**
- Modify: `README.md`
- Modify: `openspec/specs/task-lifecycle-command-chains/spec.md`
- Modify: `openspec/specs/lifecycle-settings-persistence/spec.md`
- Modify: `openspec/specs/default-branch-lifecycle-presets/spec.md`

**Step 1: Update user-facing and normative documentation**

- Replace references to per-hook default chains with named presets and one optional default preset.
- Specify migration, explicit-empty task selection, task independence from preset changes, and command-chain reference protection.
- Keep the README focused on visible behavior rather than implementation details.

**Step 2: Run the full verification suite**

Run from `taskai`:

```bash
go test ./...
```

Run from `taskai/frontend`:

```bash
npm test
npm run build
```

Then run from `taskai`:

```bash
openspec validate --strict
git diff --check
git status --short
```

Expected: all test/build/validation commands exit 0; the final status contains only the intended documentation changes before they are staged.

**Step 3: Commit documentation and verification-ready state**

```bash
git add README.md openspec/specs/task-lifecycle-command-chains/spec.md openspec/specs/lifecycle-settings-persistence/spec.md openspec/specs/default-branch-lifecycle-presets/spec.md
git commit -m "docs: specify lifecycle command presets"
```
