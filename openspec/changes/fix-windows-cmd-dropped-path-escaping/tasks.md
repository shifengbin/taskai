# 任务：修复 cmd 拖放路径转义在 Windows 26200 构建上的回归

- [x] 在真实 ConPTY cmd 会话中完成转义证据矩阵（caret 残留、引号内裸放、% 各候选方案、.exe CRT 接收方）
- [x] `internal/terminal/file_paths.go` cmd 分支移除 caret 转义，改为整体加引号、内容原样，保留双引号拒绝
- [x] 更新 `internal/terminal/file_paths_test.go` cmd 表驱动期望
- [x] 重写 `internal/terminal/file_paths_windows_test.go`：路径改单个 `%`，断言改子串匹配，注释记录已知限制
- [x] `go test ./internal/terminal/` 与 `go vet ./internal/terminal/` 通过
- [x] openspec 变更文档（proposal/design/tasks/delta spec/verification-baseline）
- [ ] wails dev + chrome-devtools 端到端集成测试（按 design.md 细节：插入文本、for 解析单参数、多文件、控制台无错误）
- [ ] 编译可执行程序并打开，等待确认
- [ ] 合并 worktree 分支到工作区分支，冲突处理
- [ ] 合并后编译验证
- [ ] 同步 delta spec 到 `openspec/specs/terminal-file-drop/spec.md` 并归档变更
- [ ] 提交 git 变更、打 `v*` 标签发布新版本
- [ ] 移除已合并的 worktree
