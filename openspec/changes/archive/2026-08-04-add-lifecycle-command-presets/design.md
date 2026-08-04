## Context

Lifecycle configuration currently stores one default command chain for each of five hooks. This is flexible but requires repeated selection and makes a common multi-hook configuration easy to assemble incorrectly. Tasks already persist a direct hook-to-chain map and command chains resolve their latest definitions at execution time.

The change crosses persisted settings, migration, repository validation, service creation defaults, Wails bindings, and the settings/task React surfaces. Existing settings files must remain readable and old tasks without a map must still receive the same status-sensitive defaults they received before this change.

## Goals / Non-Goals

**Goals:**

- Store reusable named lifecycle presets, each mapping zero or more hooks to command-chain IDs.
- Allow one optional default preset for newly created tasks.
- Preserve each task's resolved mapping independently of future preset edits or deletion.
- Migrate a legacy per-hook default map exactly once and preserve an explicitly empty legacy map.
- Prevent command-chain changes that would invalidate a preset reference.

**Non-Goals:**

- Persisting a task-to-preset relationship or retroactively applying preset updates to tasks.
- Changing lifecycle execution order, command-chain execution semantics, or task status protections.
- Introducing preset import/export, versioning, or cross-workspace sharing.

## Decisions

### Presets are settings-owned value objects

`LifecyclePreset` has a stable ID, display name, and hook-to-chain map. `Settings` owns a list of presets and an optional `DefaultLifecyclePresetID`. Task records remain unchanged: they store only `LifecycleChains`.

This uses copy-on-apply semantics instead of a task preset reference. A live reference would make command selection for a previously created task depend on unrelated settings changes, contradicting the need for task stability and complicating deletion/history behavior.

### Default selection resolves through a safe settings helper

All runtime callers obtain a copied map from the currently selected preset through settings normalization/helper APIs. An absent default ID and an empty preset map are both valid and return an empty map. The former legacy map remains only as a JSON compatibility input and is never used as a runtime default.

Using a single helper prevents direct legacy-field reads from surviving in services and makes a missing/deleted default safe. It is preferable to duplicating lookup logic in repository and lifecycle service code.

### Migration is versioned and one-time

The preset migration version advances. On the first load of pre-preset data, its legacy `lifecycleDefaultChains` map is copied into one stable-ID default preset, including an explicit empty map; the legacy serialized field is cleared and the version is persisted. Fresh installs seed the same standard preset directly. Once marked migrated, no preset is recreated, so deleting a migrated preset is durable.

This distinguishes an omitted legacy map from an explicitly empty map during JSON decoding. A generic default-on-empty normalization would lose that intent and could resurrect removed defaults.

### Repository is the integrity boundary

Preset writes are dedicated repository read-modify-write operations. They normalize IDs/names, deep-copy maps, check uniqueness, and validate every referenced chain's hook applicability. The chain save and delete paths also inspect all presets: deleting a referenced chain or narrowing its supported hooks is rejected before persistence.

Validating only in the UI would allow API callers and persisted data to create invalid references. Validating whole settings after each write keeps the existing lifecycle integrity model.

### The UI uses temporary preset selection state

The task dialog keeps its draft hook map and a selected preset ID only as local UI state. Selecting a preset replaces the entire draft, including clearing hooks omitted by the preset. Editing any individual hook clears the selected ID and renders "自定义". Saving a new task always uses the explicit-chain creation API, including an empty `{}`, so manually clearing all hooks cannot fall back to the default.

For a pending task, the UI derives a matching preset by deep equality of the saved map and current preset maps. Running and completed tasks remain read-only. This derives a display convenience without adding a persisted association.

## Risks / Trade-offs

- [A legacy empty map can be confused with an absent map during migration] -> Preserve map presence while decoding and add migration tests for both cases.
- [Preset mutation can accidentally mutate task or settings maps through shared references] -> Deep-copy maps at normalization, repository boundaries, and UI apply boundaries.
- [Chain applicability edits can leave indirect invalid references] -> Validate all presets before accepting chain changes and cover save/delete rejection tests.
- [A stale ordinary settings snapshot can overwrite lifecycle preset data] -> Extend the existing ordinary-settings protection to preserve preset fields.
- [Preset and default names may make task UI ambiguous] -> Represent matching state as a derived selector label and always show the five resolved hook selectors.

## Migration Plan

1. Deploy the settings model and migration with the preset version increment.
2. On the next settings load, normalize legacy commands/chains, copy any legacy default map into the default preset once, clear the legacy field, normalize presets, then persist the normalized settings using the existing write-back path.
3. Resolve legacy tasks that have no `LifecycleChains` using the resulting default preset with existing pending/running/completed status rules.
4. Expose dedicated preset operations and replace frontend default-chain controls.

Rollback remains data-compatible because the legacy field is omitted after migration and older binaries use their own defaults when it is absent; no task format changes are made. Rolling back would lose preset management rather than corrupt task maps, so rolling forward is preferred after users begin configuring presets.

## Open Questions

None. The product decisions for default absence, deletion behavior, task independence, and explicit empty selections are confirmed.
