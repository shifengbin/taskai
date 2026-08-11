## Why

Windows 上的清单文件命令在写入前会探测清单目标，以判断它是否为重解析点。该探测在 `internal/lifecycle/manifest_path_windows.go` 的 `validateManifestWindowsTarget` 中调用 `NtCreateFile`，其 `CreateOptions` 含 `FILE_SYNCHRONOUS_IO_NONALERT`，但 `DesiredAccess` 只给了 `FILE_READ_ATTRIBUTES`（0x80），缺少 `SYNCHRONIZE`（0x100000）。

这违反 `NtCreateFile` 的契约：当 `CreateOptions` 指定同步 IO（`FILE_SYNCHRONOUS_IO_ALERT` 或 `FILE_SYNCHRONOUS_IO_NONALERT`）时，`DesiredAccess` 必须包含 `SYNCHRONIZE`，否则 I/O 管理器在参数校验阶段直接返回 `STATUS_INVALID_PARAMETER`（0xC000000D），其 ntdll 消息表文本即 “An invalid parameter was passed to a service or function.”。该参数校验发生在路径解析之前，因此探测连目标存不存在都不会检查，即便是干净任务目录下不存在的普通目标也会被拒绝，错误逐层包装为「执行生成清单文件失败: 检查清单文件目标失败: An invalid parameter was passed to a service or function.」。

同一文件内其余 5 处打开/创建调用都通过 `FILE_GENERIC_READ`/`FILE_GENERIC_WRITE`/`FILE_DELETE` 隐式带上了 `SYNCHRONIZE`，唯独目标验证这一处遗漏。该缺陷自「生成清单文件」命令首次引入（`b98fc9b feat(lifecycle): add manifest file command`）即存在，并非后续重解析点修复的回归，使该命令在 Windows 上从未真正可用。仓库自带的跨平台 happy-path 测试在 Windows 上一字不差地复现了该错误。

排查确认 `NtCreateFile` 在整个仓库中仅经由这一个 helper 使用，缺少 `SYNCHRONIZE` 的调用点仅此一处，别处无同类隐患。

## What Changes

- 修正 Windows 清单目标验证探测的 `NtCreateFile` 调用，使其 `DesiredAccess` 包含 `SYNCHRONIZE`，与同文件其它打开/创建调用保持一致。
- 不改变任何安全边界：仍以 `FILE_OPEN_REPARSE_POINT` 打开目标并检查 `FILE_ATTRIBUTE_REPARSE_POINT`，重解析点目标继续被拒绝，写入仍以已验证的任务工作目录句柄为根完成原子替换。
- 以现有跨平台 happy-path 清单测试作为 Windows 回归门禁，并在规格中明确「对不存在或为普通文件的目标，目标验证不得阻止写入」的要求。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `manifest-file-generation`：补充「目标验证不得因普通目标（不存在或为普通文件）而阻止写入」的要求，并新增 Windows 普通目标写入场景，作为回归保护。

## Impact

- 影响 `internal/lifecycle/manifest_path_windows.go` 中 `validateManifestWindowsTarget` 的 `DesiredAccess` 掩码，为一处单点改动。
- 使现有 `internal/lifecycle/manifest_test.go` 跨平台测试（默认位置写入、自定义目录、空仓库、原子覆盖）在 Windows 上由失败转为通过。
- 不改变 Unix 实现、清单 YAML 格式、参数校验、生命周期钩子、命令链数据传递语义或任何公共 API。
