## Context

`internal/workspace/ownership_unix.go` 在 Darwin 和 Linux 上负责收紧并验证私有工作区元数据目录。普通 Unix 检查确认目录属于当前用户且权限不向组或其他用户开放；Darwin 另外通过 cgo 读取和清理扩展 ACL。

当前 Darwin cgo 文件引用 `acl_delete_file_np`。该名称未在当前 macOS SDK 头文件中声明，Apple Libc 中对应实现也不可用于清理 ACL，导致 Wails 在 `darwin/universal` 生成绑定时停止。用户已决定不再处理 macOS 扩展 ACL。

约束包括：不得放宽路径边界、令牌、目录自身标记和文件系统身份校验；不得改变 Wails API 或持久化结构；验证必须覆盖实际应用启动，并保留 GitHub Actions 的 universal 构建作为 macOS 最终检查。

## Goals / Non-Goals

**Goals:**

- 移除 `internal/workspace` 的 Darwin cgo 依赖和无效 ACL API 调用。
- 保留当前用户所有权、`0700` 权限和已有安全删除边界。
- 以自动化测试约束 Darwin 文件选择不再包含 cgo 文件。
- 验证 Darwin amd64、arm64 编译以及 Wails 应用启动路径。

**Non-Goals:**

- 不读取、清理或拒绝 macOS 扩展 ACL。
- 不替换工作区所有权令牌、扩展属性或文件系统身份机制。
- 不修改前端界面、Wails 绑定或 GitHub Release 产物格式。

## Decisions

### Darwin 与 Linux 共用 Unix ACL 空实现

删除 `ownership_acl_darwin_cgo.go`、`ownership_acl_darwin_nocgo.go` 和带隐式 Linux 文件名约束的 `ownership_acl_linux.go`，新增构建约束为 `darwin || linux` 的 `ownership_acl_unix.go`。两个 ACL 函数返回 `nil`，现有调用链无需条件分支。

选择该方案是因为它完整表达用户确认的行为，并从文件选择阶段消除 cgo。备选方案 `acl_init(0)` 加 `acl_set_file` 仍保留 macOS C API；调用 `chmod -N` 则增加外部进程依赖，均不符合当前范围。

### 用 `go list` 验证构建文件选择

回归测试以 `GOOS=darwin`、`GOARCH=amd64`、`CGO_ENABLED=1` 运行 `go list`，断言 `internal/workspace` 的 `CgoFiles` 为空。该测试可在 Linux 开发环境执行，修改前会稳定列出失败文件，修改后直接证明 cgo 源文件已从 Darwin 包中移除。

### 平台编译与运行分层验证

本地先用 `CGO_ENABLED=0` 分别为 Darwin amd64 和 arm64 编译 `internal/workspace` 测试程序，验证 Go 平台文件完整。随后执行 Linux 完整测试、项目编译和 `wails dev` 浏览器测试。GitHub Actions 的 macOS runner 继续负责带系统框架与 cgo 的 Wails universal 和 DMG 最终验证。

## Risks / Trade-offs

- [macOS 扩展 ACL 可能授予普通权限之外的访问] → 用户已明确接受不处理扩展 ACL；文档清楚记录安全边界只包含所有者和普通 Unix 权限。
- [仅在 Linux 上无法完成 macOS 系统框架链接] → 本地执行 Darwin 双架构 Go 编译，最终由 macOS GitHub Actions universal 构建验证系统工具链。
- [嵌套执行 `go list` 增加测试耗时] → 只查询一个包且不编译，影响很小，并提供可在所有开发平台运行的稳定回归信号。

## Migration Plan

该变更没有数据迁移。发布时直接替换平台实现；已有目录不会被修改扩展 ACL，后续访问仍会执行所有者和普通权限检查。

若需要回滚安全语义，恢复 Darwin 专用实现时必须改用当前 SDK 声明且可运行的 API，并先增加真实 macOS ACL 行为测试，不能恢复 `acl_delete_file_np` 调用。

## Open Questions

无。用户已确认不处理 macOS 扩展 ACL。
