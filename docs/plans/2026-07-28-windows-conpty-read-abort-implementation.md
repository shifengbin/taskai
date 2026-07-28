# Windows ConPTY 关闭读取误报实施计划

> **For Codex:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**目标：** 关闭 Windows ConPTY 时不再将预期的错误 995 上报为终端异常。

**架构：** 在终端包中提供一个与操作系统无关、可执行测试的错误 995 分类辅助函数。Windows 构建文件调用该函数；现有终端管理器和 Unix 的 `EIO` 分类保持不变。

**技术栈：** Go、Windows ConPTY、Wails。

---

### 任务 1：建立错误 995 的回归测试

**文件：**
- 新建：`internal/terminal/read_error_windows_common_test.go`
- 新建：`internal/terminal/read_error_windows_common.go`

**步骤 1：编写失败测试**

为 `isExpectedWindowsTerminalReadError` 增加两项断言：被 `fmt.Errorf("读取失败: %w", syscall.Errno(995))` 包装的错误应返回 `true`；其他错误应返回 `false`。在 `manager_windows_test.go` 验证管理器会将 995 直接作为退出处理，而 996 仍发布错误事件。

**步骤 2：运行测试并确认失败**

运行：`go test ./internal/terminal -run TestExpectedWindowsTerminalReadError -count=1`

预期：测试因缺少辅助函数而失败。

### 任务 2：实现最小的 Windows 分类逻辑

**文件：**
- 修改：`internal/terminal/read_error_windows.go`
- 新建：`internal/terminal/read_error_windows_common.go`

**步骤 1：实现辅助函数**

使用 `errors.Is(err, syscall.Errno(995))` 判断错误链中是否包含 Windows `ERROR_OPERATION_ABORTED`。

**步骤 2：接入 Windows 平台函数**

让 Windows 的 `isExpectedTerminalReadError` 委托给该辅助函数，不扩大对其他 I/O 错误的忽略范围。

**步骤 3：运行定向测试并确认通过**

运行：`go test ./internal/terminal -run TestExpectedWindowsTerminalReadError -count=1`

预期：测试通过。

### 任务 3：全量验证与构建

**文件：**
- 验证：`internal/terminal/read_error_windows_common_test.go`
- 验证：`scripts/build-linux.sh`

**步骤 1：运行全量 Go 测试**

运行：`go test -race ./...`

**步骤 2：运行前端测试和构建**

运行：`cd frontend && npm test && npm run build`

**步骤 3：构建可执行程序**

运行：`./scripts/build-linux.sh`

**步骤 4：检查 Windows 交叉编译**

运行：`GOOS=windows GOARCH=amd64 go test -c ./internal/terminal`

预期：Windows 条件编译通过；Windows 运行时行为在目标主机上由同一回归测试验证。
