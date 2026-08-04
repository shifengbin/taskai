## Why

Per-hook lifecycle defaults make common command-chain combinations slow to configure and easy to misconfigure. Users need named bundles that can be applied to a new task at once while preserving the ability to make task-specific adjustments.

## What Changes

- Replace the five independent lifecycle default-chain settings with reusable named lifecycle presets and one optional default preset.
- Let a new task apply the default preset or a chosen preset, then independently adjust individual hook chains before saving.
- Persist only the resolved hook-to-chain mapping on tasks, so later preset changes or deletion never alter existing tasks.
- Migrate legacy default-chain mappings once into a default preset without recreating a preset the user later deletes.
- Reject command-chain deletion or applicability reductions that would leave a preset with an invalid chain reference.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `task-lifecycle-command-chains`: Replace per-hook defaults with presets, define task preset application and command-chain reference protection.
- `lifecycle-settings-persistence`: Persist, validate, migrate, and manage lifecycle presets through dedicated settings APIs.
- `default-branch-lifecycle-presets`: Update existing preset/default-chain terminology and guarantees to lifecycle preset semantics.

## Impact

Changes the Go settings model, repository lifecycle APIs and validation, task-creation default resolution, Wails bindings, and the React settings/task dialogs. Existing settings files with `lifecycleDefaultChains` are migrated compatibly; task data remains a direct hook-to-chain map.
