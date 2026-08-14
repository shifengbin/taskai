# 修复 macOS 工作区构建设计

## 背景

macOS universal 构建在 Wails 生成绑定阶段编译 `internal/workspace` 时失败。`ownership_acl_darwin_cgo.go` 调用了当前 macOS SDK 未声明的 `acl_delete_file_np`，导致 cgo 无法识别该名称。该函数在 Apple Libc 中也不提供可用的 ACL 清理行为，因此不能通过补充声明可靠修复。

用户已确认 TaskAI 不需要处理 macOS 扩展 ACL。工作区私有目录继续依靠当前用户所有权和 `0700` 权限保证访问边界。

## 方案比较

### 方案一：移除 macOS 扩展 ACL 处理

让 Darwin 与 Linux 共用空的 ACL 处理函数，删除 Darwin 的 cgo 与非 cgo 分支。目录所有者、普通 Unix 权限、路径边界和所有权令牌校验保持不变。

优点是完全移除导致构建失败的 cgo 依赖，代码与用户确认的安全范围一致，也不会保留只在部分 SDK 上可编译的路径。缺点是不再检查 macOS 扩展 ACL。

### 方案二：改用 `acl_init` 与 `acl_set_file`

继续通过 cgo 清空扩展 ACL。该方案能保留原设计目标，但用户已明确不要求处理 ACL，并且仍会保留 macOS 专用 cgo 编译链。

### 方案三：调用系统 `chmod -N`

通过外部进程清空 ACL。该方案避免直接引用 ACL C API，但增加系统命令依赖、进程错误处理和路径传递复杂度，也超出当前需求。

## 决定

采用方案一。将现有 Linux ACL 空实现调整为 Darwin 与 Linux 共用的 Unix 空实现，删除 `ownership_acl_darwin_cgo.go` 和 `ownership_acl_darwin_nocgo.go`。`secureAndValidatePrivateDirectory` 仍会把目录权限收紧为 `0700`，并继续验证目录属于当前用户。

## 错误处理

移除 macOS ACL 读取和清理错误。目录不存在、所有者不匹配、普通权限过宽、路径越界、所有权令牌或文件系统身份不匹配时，继续沿用现有错误和拒绝行为。

## 测试

先增加回归测试，通过 `GOOS=darwin`、`CGO_ENABLED=1` 执行 `go list`，断言 `internal/workspace` 不包含任何 `CgoFiles`。修改前该测试会列出 `ownership_acl_darwin_cgo.go` 并失败；修改后应通过。

完成代码修改后按以下顺序验证：

1. 运行新增的最小回归测试，确认 Darwin 工作区包不再依赖 cgo。
2. 运行 `go test -race ./...`，确认工作区所有权和删除测试保持通过。
3. 运行前端测试与构建，确认嵌入资源完整。
4. 将当前主分支合并到 worktree 分支后再次执行完整测试。
5. 使用项目脚本编译 Linux 可执行程序。
6. 使用 `wails dev` 启动应用，从输出取得调试地址，并通过浏览器检查应用能够加载、任务列表能够显示且没有启动错误；验证完成后关闭调试进程。
7. macOS GitHub Actions 继续执行 `scripts/build-macos.sh --dmg`，其中 Wails 的 `darwin/universal` 构建是最终平台集成验证，预期不再出现 `C.acl_delete_file_np` 错误。
