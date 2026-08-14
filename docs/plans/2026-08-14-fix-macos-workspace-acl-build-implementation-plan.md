# 修复 macOS 工作区构建实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除 `internal/workspace` 对 macOS 扩展 ACL 和 cgo 的依赖，使 Wails `darwin/universal` 构建恢复通过。

**Architecture:** Darwin 与 Linux 共用不处理扩展 ACL 的 Unix 实现，现有目录所有者、`0700` 权限、路径边界和所有权令牌校验保持不变。通过 `go list` 验证 Darwin 文件选择结果不再包含 cgo 文件，并保留双架构交叉编译和 Wails 启动验证。

**Tech Stack:** Go 1.23、Go build constraints、Wails v2、OpenSpec、Vitest

---

### Task 1: 建立 OpenSpec 变更

**Files:**
- Create: `openspec/changes/remove-macos-workspace-acl-checks/proposal.md`
- Create: `openspec/changes/remove-macos-workspace-acl-checks/design.md`
- Create: `openspec/changes/remove-macos-workspace-acl-checks/specs/macos-workspace-build-compatibility/spec.md`
- Create: `openspec/changes/remove-macos-workspace-acl-checks/tasks.md`

**Step 1: 创建变更目录**

Run: `openspec new change "remove-macos-workspace-acl-checks"`

**Step 2: 按 OpenSpec 指令生成提案、规格、设计和任务**

规格必须说明：Darwin 工作区包不得因为扩展 ACL 引入 cgo；普通目录所有者与 `0700` 校验继续执行；集成测试必须包含 `go list` 回归、Darwin amd64/arm64 交叉编译、`wails dev` 启动检查和 GitHub Actions universal 构建。

**Step 3: 校验变更可实施**

Run: `openspec status --change "remove-macos-workspace-acl-checks"`

Expected: 所有实施所需文档均为完成状态。

### Task 2: 先增加 Darwin 无 cgo 回归测试

**Files:**
- Create: `internal/workspace/platform_build_test.go`
- Delete: `internal/workspace/ownership_acl_darwin_cgo.go`
- Delete: `internal/workspace/ownership_acl_darwin_nocgo.go`
- Delete: `internal/workspace/ownership_acl_linux.go`
- Create: `internal/workspace/ownership_acl_unix.go`

**Step 1: 写入失败测试**

```go
package workspace

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDarwinWorkspacePackageDoesNotUseCgo(t *testing.T) {
	command := exec.Command("go", "list", "-f", `{{join .CgoFiles "\\n"}}`, ".")
	command.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=amd64", "CGO_ENABLED=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list darwin workspace error = %v, output = %s", err, output)
	}
	if cgoFiles := strings.TrimSpace(string(output)); cgoFiles != "" {
		t.Fatalf("darwin workspace CgoFiles = %q, want empty", cgoFiles)
	}
}
```

**Step 2: 运行测试并确认失败原因**

Run: `go test ./internal/workspace -run TestDarwinWorkspacePackageDoesNotUseCgo -count=1`

Expected: FAIL，输出包含 `ownership_acl_darwin_cgo.go`。

**Step 3: 写入最小实现**

创建 `ownership_acl_unix.go`：

```go
//go:build darwin || linux

package workspace

func securePrivateDirectoryACL(string) error {
	return nil
}

func validateExtendedACL(string) error {
	return nil
}
```

删除三个旧平台文件，不修改 `ownership_unix.go` 中的所有者和普通权限校验。

**Step 4: 运行测试并确认通过**

Run: `go test ./internal/workspace -run TestDarwinWorkspacePackageDoesNotUseCgo -count=1`

Expected: PASS。

**Step 5: 运行工作区包测试**

Run: `go test -race ./internal/workspace`

Expected: PASS。

### Task 3: 验证双架构构建与完整测试

**Files:**
- Modify: `openspec/changes/remove-macos-workspace-acl-checks/tasks.md`

**Step 1: 编译 Darwin amd64 测试程序**

Run: `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go test -c -o <临时目录>/workspace-amd64.test ./internal/workspace`

Expected: exit 0。

**Step 2: 编译 Darwin arm64 测试程序**

Run: `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go test -c -o <临时目录>/workspace-arm64.test ./internal/workspace`

Expected: exit 0。

**Step 3: 合并主工作区最新提交**

Run: `git merge main`

Expected: 合并成功；如有冲突，解决后重新执行相关测试。

**Step 4: 运行完整验证**

Run: `go test -race ./...`

Run: `cd frontend && npm test -- --run && npm run build`

Expected: 所有测试和构建通过。

### Task 4: 编译、启动与文档同步

**Files:**
- Modify: `openspec/changes/remove-macos-workspace-acl-checks/tasks.md`
- Modify: `docs/plans/2026-08-14-fix-macos-workspace-acl-build-implementation-plan.md`

**Step 1: 编译 Linux 可执行程序**

Run: `bash scripts/build-linux.sh amd64`

Expected: `build/bin/taskai` 生成成功。

**Step 2: 启动 Wails 开发模式**

Run: `wails dev -tags webkit2_41`

从输出获取调试地址，保持进程运行，不设置禁用颜色的环境变量。

**Step 3: 执行浏览器集成测试**

使用 chrome-devtools 打开调试地址，确认应用加载、任务列表可见、控制台无启动错误。完成后关闭 `wails dev`。

**Step 4: 等待用户确认**

在用户确认前，不把 worktree 分支合并回主工作区。

**Step 5: 用户确认后完成归档与合并**

归档 OpenSpec 变更，将 worktree 分支合并回当前项目分支，再次编译验证，提交最终文档和代码，最后移除 worktree。

## 执行记录

- 回归测试红灯：修改前稳定输出 `ownership_acl_darwin_cgo.go`，证明测试覆盖原始构建失败来源。
- 回归测试绿灯：移除 Darwin ACL cgo 后，最小测试和 `go test -race ./internal/workspace` 通过。
- Darwin 编译：amd64 生成 Mach-O x86_64 测试程序，arm64 生成 Mach-O arm64 测试程序。
- 完整测试：`go test -race ./...` 通过；前端 16 个测试文件、336 项测试通过；生产前端构建通过。
- 可执行程序：`bash scripts/build-linux.sh amd64` 成功生成 `build/bin/taskai`。
- Wails 集成测试：`wails dev -tags webkit2_41` 成功启动，调试地址为 `http://localhost:34115`。chrome-devtools 确认“任务工作台”、任务状态和任务列表正常显示，业务资源请求成功；唯一 404 为已有的 `/favicon.ico`，与本次启动和 ACL 修复无关。验证后已关闭调试进程。
- 合并后验证：worktree 分支无冲突合并到 `main`；`go test -race ./...`、前端 336 项测试、前端生产构建、Darwin amd64/arm64 工作区包交叉编译与 `bash scripts/build-linux.sh amd64` 均再次通过。
- macOS 最终验证：代码不再包含 Darwin cgo 文件；实际 Wails `darwin/universal` 与 DMG 仍由 macOS GitHub Actions runner 验证。
