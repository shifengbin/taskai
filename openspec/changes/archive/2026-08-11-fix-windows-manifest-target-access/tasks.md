## 1. 修正 Windows 目标验证访问掩码

- [x] 1.1 在 `internal/lifecycle/manifest_path_windows.go` 的 `validateManifestWindowsTarget` 中，将 `NtCreateFile` 的 `DesiredAccess` 由 `windows.FILE_READ_ATTRIBUTES` 改为 `windows.FILE_READ_ATTRIBUTES | windows.SYNCHRONIZE`，满足同步 IO 选项对访问掩码必须包含 `SYNCHRONIZE` 的契约。
- [x] 1.2 确认改动不放松安全边界：仍以 `FILE_OPEN_REPARSE_POINT` 打开目标并通过 `GetFileInformationByHandle` 检查 `FILE_ATTRIBUTE_REPARSE_POINT`，重解析点目标继续被拒绝。

## 2. 回归测试

- [x] 2.1 确认现有跨平台 `internal/lifecycle/manifest_test.go` happy-path 测试（默认位置写入、自定义目录、空仓库、原子覆盖）在 Windows 上由失败转为通过，作为本次修复的回归门禁。
- [x] 2.2 在 `validateManifestWindowsTarget` 访问掩码处增加一行注释，固化「`FILE_SYNCHRONOUS_IO_NONALERT` 选项需配 `SYNCHRONIZE` 访问权」这一约束，防止同类组合回归；行为级回归由 2.1 的 happy-path 测试覆盖。

## 3. 验证

- [x] 3.1 在 Windows 上运行 `go test ./internal/lifecycle/` 及应用层清单相关测试，确认清单生成、重试与失败语义恢复正常，且重解析点拒绝行为不变。包内唯一失败 `TestCommandChainRunnerRejectsUnsafeSpecifiedRepositoryTargetBeforeGit/符号链接目标` 经 `git stash` 对照确认是既有 Windows 符号链接权限环境问题，与本改动无关。
- [x] 3.2 在非 Windows 平台确认 `manifest_path_unix.go` 路径与跨平台测试不受影响：`GOOS=linux go vet ./internal/lifecycle/` 通过，`git status` 确认仅 `manifest_path_windows.go` 被改动。
