# HTTP 服务生命周期与命令环境 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 避免无关设置保存重启 HTTP 服务，在所有命令路径注入正确的任务上下文，并明确 HTTP 文档参数和工作状态动效。

**Architecture:** 应用层在保存设置时比较 HTTP 相关字段，HTTP 服务器在同端口重配时保持幂等作为防御。终端和无终端命令分别构造环境变量并传入统一的进程启动函数；前端仅调整说明文本与状态圆点样式。

**Tech Stack:** Go、`net/http`、`os/exec`、React、TypeScript、MUI、Vitest。

---

### Task 1: 防止无关保存重复监听 HTTP 端口

**Files:**
- Modify: `internal/realtime/http.go:75-102`
- Modify: `internal/realtime/http_test.go:1-180`
- Modify: `app.go:179-195,477-496`
- Modify: `app_test.go`

**Step 1: 写失败测试**

增加两个测试：

```go
func TestHTTPServerConfigureSamePortKeepsExistingListener(t *testing.T) {
    server := startHTTPServer(t, New(Options{}), nil)
    port := server.listener.Addr().(*net.TCPAddr).Port
    previousURL := server.APIURL()
    if err := server.Configure(port); err != nil {
        t.Fatalf("同端口配置: %v", err)
    }
    if server.APIURL() != previousURL {
        t.Fatalf("API 地址变化: %q", server.APIURL())
    }
}
```

应用层测试先以 HTTP 模式保存固定端口，再仅改变 `ActiveTaskStatus` 保存一次，断言不返回端口占用错误且 `APIURL()` 不变。

**Step 2: 运行测试确认失败**

```bash
go test . ./internal/realtime -run 'Test(App.*HTTP|HTTPServerConfigureSamePort)'
```

预期：同端口重配因端口被占用失败。

**Step 3: 最小实现**

在 `HTTPServer.Configure` 获取互斥锁后，若现有监听器是相同的非零端口则立即返回。应用层新增比较旧、新 HTTP 服务相关字段的帮助函数；`SaveSettings` 只在字段变化时调用 `applyStatusSettings`。启动路径仍直接调用 `applyStatusSettings`。

**Step 4: 运行通过测试**

```bash
go test . ./internal/realtime -run 'Test(App.*HTTP|HTTPServerConfigureSamePort)'
```

预期：测试通过，保存任务标签不重新绑定端口。

### Task 2: 为全部命令路径传递任务环境变量

**Files:**
- Modify: `app.go:35-40,235-271,498-520,633-690`
- Modify: `app_test.go`

**Step 1: 写失败测试**

为普通终端、显示终端命令、无终端菜单命令、前置脚本和后置脚本分别断言环境变量：终端包含任务和终端 ID；无终端路径仅包含任务 ID；HTTP 状态管理模式下终端额外包含 API 地址。

**Step 2: 运行测试确认失败**

```bash
go test . -run 'TestApp.*(Environment|TaskID)'
```

预期：非 HTTP 终端和无终端命令路径尚未收到这些变量。

**Step 3: 最小实现**

将命令和脚本启动函数增加 `environment []string` 参数。`configureCommandProcess` 在保留 PowerShell 已有自定义环境的基础上合并系统环境与额外变量。应用层分别构造终端环境和无终端任务环境，按照设计表传入所有启动路径。

**Step 4: 运行通过测试**

```bash
go test . -run 'TestApp.*(Environment|TaskID)'
```

预期：所有变量注入测试通过。

### Task 3: 明确参数文档并统一活动状态动效

**Files:**
- Modify: `frontend/src/App.tsx:843-870`
- Modify: `frontend/src/App.test.tsx:451-490`
- Modify: `frontend/src/components/TerminalStatusDot.tsx`
- Create: `frontend/src/components/TerminalStatusDot.test.tsx`
- Modify: `README.md:17-78`

**Step 1: 写失败测试**

在 HTTP 帮助弹窗测试中断言以下文字：

```ts
expect(screen.getByText(/查询参数.*pending.*running.*completed/)).toBeInTheDocument()
expect(screen.getByText(/请求体.*idle.*working.*unread.*error/)).toBeInTheDocument()
```

在状态圆点测试中渲染 `working`，断言它带有代表扩散动画的 `data-pulse="active"`；`idle` 与 `error` 不带该属性。

**Step 2: 运行测试确认失败**

```bash
cd frontend && npm test -- --run src/App.test.tsx src/components/TerminalStatusDot.test.tsx
```

预期：合法参数提示和工作状态动效标识缺失。

**Step 3: 最小实现**

在帮助弹窗中分别增加“任务列表查询参数”和“状态更新请求体”的合法值说明。提取状态点的活动判断，给 `working`、`unread` 增加同一扩散关键帧和各自主题颜色的扩散环；保留 `prefers-reduced-motion` 禁用规则。README 同步为相同的参数与变量规则。

**Step 4: 运行通过测试**

```bash
cd frontend && npm test -- --run src/App.test.tsx src/components/TerminalStatusDot.test.tsx
```

预期：文档和动效测试通过。

### Task 4: 完整验证

**Files:**
- Verify: 当前工作区

**Step 1: 运行后端测试**

```bash
go test -race ./...
```

**Step 2: 运行前端测试与构建**

```bash
cd frontend && npm test && npm run build
```

**Step 3: 使用项目构建脚本并检查差异**

```bash
./scripts/build-linux.sh amd64
git diff --check
```

预期：全部命令退出码为零。
