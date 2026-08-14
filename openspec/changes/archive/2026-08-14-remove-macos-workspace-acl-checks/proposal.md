## Why

macOS universal 构建在 Wails 生成绑定时因 `internal/workspace` 引用 SDK 未声明的 `acl_delete_file_np` 而失败。用户已确认不需要处理 macOS 扩展 ACL，因此应移除这项无效依赖，并让跨平台构建只依赖项目实际采用的目录所有者和普通权限校验。

## What Changes

- 删除 Darwin 专用的扩展 ACL cgo 与非 cgo 实现。
- 让 Darwin 与 Linux 共用不处理扩展 ACL 的 Unix 实现。
- 保留目录当前用户所有权、`0700` 权限、路径边界、所有权令牌和文件系统身份校验。
- 增加 Darwin 工作区包无 cgo 依赖的回归测试，并验证 amd64、arm64 和 Wails universal 构建路径。
- **BREAKING**: macOS 私有工作区元数据目录不再读取、清理或拒绝扩展 ACL，仅依据目录所有者和普通 Unix 权限判断访问边界。

## Capabilities

### New Capabilities

- `macos-workspace-build-compatibility`: 规定 Darwin 工作区包不依赖扩展 ACL cgo，并定义跨架构构建与启动集成测试。

### Modified Capabilities

无。

## Impact

- 影响 `internal/workspace` 的 Darwin 与 Linux 平台文件选择。
- 移除 `internal/workspace` 对 macOS ACL C API 和 cgo 的依赖。
- 不改变 Wails 导出接口、任务持久化结构、工作目录删除边界或前端行为。
- GitHub Actions 的 macOS universal 构建将直接验证修复结果。
