# 验证基线：修复 cmd 拖放路径转义在 Windows 26200 构建上的回归

环境：Windows 11 Home China 10.0.26200.9168，Go 1.23，工作区 worktree `fix-cmd-dropped-path-escaping`。

## 预存失败（与本次改动无关，改动仅触及 internal/terminal 三个文件）

- `taskai` 根包：`all:frontend/dist` embed 缺失（worktree 未构建前端）。
- `internal/settings`：git-clone 目录参数与清单文件参数 3 个用例（历史已知失败）。
- `internal/appdata`、`internal/lifecycle`：符号链接用例（本机无 symlink 特权）与 macOS 迁移路径用例。
- `internal/appdata` 并发迁移用例偶发超时（5 分钟 timeout）。

## 已完成验证

- `go test ./internal/terminal/`：全绿（含 ConPTY 生命周期 20× 稳定套件与本次更新的拖放用例）。
- `go vet ./internal/terminal/`：无告警。
- `TestCmdDroppedFilePathParsesAsOneLiteralArgument`（cmd.exe /D /V:OFF /Q 管道）：输出含 `[C:\Work Files\a&b^f 50% x!y.txt]`，cmd 解析为单一字面参数。
- `TestFormatDroppedFilePathsForSupportedShells/cmd`：期望 `"C:\\Work Files\\a&b|c<d>e^f 50% x!y.txt"`（原样引号包裹）。

## 待完成验证（合入前）

- wails dev + chrome-devtools 端到端（design.md 细节）：插入文本原样、`for %~A` 单参数精确还原、多文件空格连接、控制台无 error。
- 可执行程序构建 + 人工确认拖放行为。
- 合并后整包编译验证。
