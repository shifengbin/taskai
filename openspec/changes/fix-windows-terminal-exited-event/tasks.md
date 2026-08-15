# 任务：修复 Windows ConPTY exited 事件不到达

- [x] 用独立探针定位根因（conhost 不结束输出管道；捆绑关闭丢数据），记录验证基线
- [x] 重写 `internal/terminal/backend_windows.go`：直接管理 ConPTY，两阶段拆卸
- [x] `read_error_windows_common.go` 识别错误 109 为预期读取结束
- [x] 集成测试加强：`TERMINAL_EXIT_SENTINEL` 最终输出不丢断言；预期读取错误判定
- [x] 移除 charmbracelet/x/conpty 依赖（go.mod/go.sum）
- [x] `go vet` 与 `go test -race`（terminal 包 20 次循环 + 全部包；settings 两个与拖放转义一个测试为与本变更无关的预存失败，已在 clean HEAD 验证）
- [ ] openspec 校验与变更文档归档
- [ ] `wails dev` + chrome-devtools 集成测试（见下）
- [ ] 编译可执行程序并打开程序，等待确认
- [ ] 合并 worktree 分支、编译验证、文档同步、提交、发布新版本

## 集成测试细节（wails dev + chrome-devtools MCP）

前置：`wails dev` 启动，从输出取得调试地址（形如 `http://localhost:34115`），用 chrome-devtools 打开。

### 冒烟测试

1. 打开应用 → 新建/选择一个任务 → 打开终端标签：确认 cmd 横幅与提示符正常显示（输出管道工作正常）。
2. 输入 `echo smoke-ok` 回车：确认回显 `smoke-ok` 出现（输入管道工作正常）。
3. 调整窗口大小：确认终端内容随尺寸重排（ResizePseudoConsole 路径）。

### 本次修改专项

4. 自然退出（核心场景）：在终端输入 `echo before-exit && exit` 回车。预期：
   - `before-exit` 回显先出现（最终输出不丢）；
   - 终端条目在约 1 秒内消失或显示退出提示（`exited` 事件及时到达），不再长时间停留"运行中"；
   - 前端无报错（console 无新增 error）。
5. 非零退出：输入 `cmd /c exit 3` 回车。预期：终端保留为异常退出快照，状态点异常，退出码按 `unexpected` 分类（验证真实退出码透传）。
6. 主动关闭：另开一个终端，运行 `ping -t 127.0.0.1`，然后点击关闭该终端。预期：关闭即时完成，无残留条目，再次打开新终端正常（句柄已释放）。
7. 重复 4 若干次（≥3）：确认行为稳定。

### 完成后

- 关闭 `wails dev`（注意：不要动 `E:\taskai\build\bin\taskai.exe`，用户在该实例里开发）。
