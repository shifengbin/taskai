## Context

清单文件命令在 Windows 上写入前由 `validateManifestWindowsTarget` 探测目标是否为重解析点。该函数经 `createManifestWindowsObject` 调用 `NtCreateFile`，其 `CreateOptions` 含 `FILE_SYNCHRONOUS_IO_NONALERT`，但 `DesiredAccess` 只给了 `FILE_READ_ATTRIBUTES`（0x80），缺少 `SYNCHRONIZE`（0x100000）。

`NtCreateFile` 的契约要求：当 `CreateOptions` 指定同步 IO（`FILE_SYNCHRONOUS_IO_ALERT` 或 `FILE_SYNCHRONOUS_IO_NONALERT`）时，`DesiredAccess` 必须包含 `SYNCHRONIZE`，否则 I/O 管理器在参数校验阶段即返回 `STATUS_INVALID_PARAMETER`（0xC000000D），其 ntdll 文本为 “An invalid parameter was passed to a service or function.”。该参数校验发生在路径解析之前，因此即便目标不存在也会被拒绝，错误逐层包装为「执行生成清单文件失败: 检查清单文件目标失败: ...」。

同一文件内其余 5 处打开/创建调用都通过 `FILE_GENERIC_READ`/`FILE_GENERIC_WRITE`/`FILE_DELETE`（这些常量定义里均 `| SYNCHRONIZE`）隐式带上了 `SYNCHRONIZE`，唯独目标验证这一处用裸 `FILE_READ_ATTRIBUTES` 而遗漏。全仓库 `NtCreateFile` 仅经此 helper 使用，缺少 `SYNCHRONIZE` 的调用点仅此一处。该缺陷自 `b98fc9b feat(lifecycle): add manifest file command` 引入即存在，使该命令在 Windows 上从未真正可用。

## Goals / Non-Goals

**Goals:**

- 让目标验证探测满足 `NtCreateFile` 同步 IO 选项的访问掩码契约，使命令在 Windows 上正常工作。
- 保留既有安全边界：仍以 `FILE_OPEN_REPARSE_POINT` 打开目标、检查 `FILE_ATTRIBUTE_REPARSE_POINT`，重解析点目标继续被拒，写入仍以已验证目录句柄为根完成原子替换。

**Non-Goals:**

- 不改变 Unix 实现、清单 YAML 内容、参数校验或命令链语义。
- 不放宽或收紧重解析点拒绝策略。

## Decisions

### 补 `SYNCHRONIZE` 访问权，而非移除同步 IO 选项

将 `validateManifestWindowsTarget` 的 `DesiredAccess` 由 `windows.FILE_READ_ATTRIBUTES` 改为 `windows.FILE_READ_ATTRIBUTES | windows.SYNCHRONIZE`。

备选方案是从 `CreateOptions` 移除 `FILE_SYNCHRONOUS_IO_NONALERT`。该句柄仅用于随后的 `GetFileInformationByHandle`（内部同步），确实不需要同步 IO 选项。但同文件其余所有调用都以同步 IO 选项配 `SYNCHRONIZE` 访问权，移除选项会让这一处在风格上与众不同。补 `SYNCHRONIZE` 直接消除「这一处与众不同」的不一致本身，与既有模式一致，故采用。`SYNCHRONIZE` 是无害的访问权（允许在句柄上等待），不放松安全语义。

### 行为级回归门禁已由现有测试覆盖

`internal/lifecycle/manifest_test.go` 的跨平台 happy-path 测试在 Windows 上修复前失败、修复后通过，天然作为本修复的回归门禁：若日后再次移除 `SYNCHRONIZE`，这些测试会在 Windows 上转红。额外在访问掩码处加一行注释，固化该约束以防「清理式」回归。不再新增专项测试，以免与既有覆盖重复。

## Risks / Trade-offs

- [误清理掉 `SYNCHRONIZE`] → 访问掩码处注释说明契约，且 happy-path 测试在 Windows 上行为级兜底。
- [该缺陷长期被掩盖] → 此前「Windows 测试失败≈环境噪音」的启发式把这类真正的 Windows 代码 bug 也一并忽略；本次以 `git stash` 对照证明包内符号链接失败确属环境权限问题，与本次改动无关。

## Migration Plan

无需迁移配置或数据。发布更新后，Windows 上此前失败的生命周期命令链可直接重试；回退到旧版本会恢复失败行为，但不会影响已生成的清单文件。

## Open Questions

- 无。
