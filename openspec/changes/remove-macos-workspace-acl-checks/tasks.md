## 1. Darwin cgo 回归与最小修复

- [x] 1.1 新增 `internal/workspace/platform_build_test.go`，以 `GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go list -f '{{join .CgoFiles "\n"}}' .` 断言 Darwin 工作区包的 `CgoFiles` 为空。
- [x] 1.2 运行 `go test ./internal/workspace -run TestDarwinWorkspacePackageDoesNotUseCgo -count=1`，确认修改前因输出 `ownership_acl_darwin_cgo.go` 而失败。
- [x] 1.3 删除 Darwin cgo、Darwin 非 cgo 和 Linux 专用 ACL 文件，新增构建约束为 `darwin || linux` 的 `ownership_acl_unix.go` 空实现。
- [x] 1.4 重新运行最小回归测试和 `go test -race ./internal/workspace`，确认无 cgo 约束及工作区行为测试通过。

## 2. 平台编译与完整自动化测试

- [x] 2.1 在 `mktemp -d` 创建的临时目录中分别运行 `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go test -c` 和 `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go test -c`，确认两个工作区测试程序均生成成功并在验证后清理临时目录。
- [x] 2.2 在 worktree 分支执行 `git merge main`，如有冲突先解决冲突，再重新运行受影响测试。
- [x] 2.3 运行 `go test -race ./...`，确认所有 Go 测试通过。
- [x] 2.4 在 `frontend` 运行 `npm test -- --run` 和 `npm run build`，确认 336 项前端测试及生产构建通过。

## 3. 应用编译与 Wails 集成测试

- [x] 3.1 运行 `bash scripts/build-linux.sh amd64`，确认生成 `build/bin/taskai` 可执行程序。
- [x] 3.2 不设置 `NO_COLOR` 等禁用颜色的环境变量，运行 `wails dev -tags webkit2_41` 并保持进程运行，从输出记录调试地址。
- [x] 3.3 使用 chrome-devtools 打开调试地址，确认应用页面加载、任务列表区域显示且浏览器控制台没有启动错误。
- [x] 3.4 浏览器验证完成后关闭 `wails dev`，记录 GitHub Actions 的 macOS `scripts/build-macos.sh --dmg` universal 构建为最终平台验证步骤。

## 4. 确认、归档与合并

- [x] 4.1 同步设计、实施计划和测试结果，向用户提供编译及集成测试证据并等待确认。
- [ ] 4.2 用户确认后归档 `remove-macos-workspace-acl-checks` OpenSpec 变更。
- [ ] 4.3 将 worktree 分支合并回当前项目分支，重新执行项目编译验证合并结果。
- [ ] 4.4 提交最终代码与文档变更并移除已合并的 worktree。
