## Why

Windows 上的清单文件命令以 `OBJ_DONT_REPARSE` 打开完整任务工作目录路径。该标志会在路径中的任意重解析点失败，使位于 OneDrive、目录联接点或挂载文件夹下的合法任务目录无法生成清单，即使工作目录本身不是链接。

任务工作目录创建逻辑允许这些上级路径，导致任务能够创建目录却在随后生成清单时失败。需要让清单命令与既有安全语义一致。

## What Changes

- Windows 清单写入器允许任务工作目录的上级路径包含可解析的重解析点。
- 继续拒绝任务工作目录自身、输出子目录和清单目标为重解析点，保持所有写入被限制在已验证的任务目录句柄内。
- 为 Windows 路径边界补充测试，覆盖合法上级重解析点与不安全最终目录或输出路径。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `manifest-file-generation`：明确允许任务工作目录的上级路径通过重解析点解析，同时保持任务工作目录本身及输出路径不得使用重解析点的安全要求。

## Impact

- 影响 `internal/lifecycle/manifest_path_windows.go` 的工作目录打开和验证流程。
- 影响 Windows 平台的清单写入测试；不改变生命周期钩子、清单 YAML 格式、配置参数或公共 API。
