# 任务 HTTP 查询与未读动效 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在现有本机 HTTP 服务中提供按生命周期筛选的任务列表与详情查询，并支持独立启用服务和未读状态心跳扩散动效。

**Architecture:** HTTP 服务继续监听现有回环端口，通过应用层提供的任务目录回调查询持久化任务数据。设置使用独立服务开关与已有状态管理方式共同决定监听器生命周期；状态 HTTP 模式优先强制服务开启。前端在同一设置分区呈现联动配置，未读圆点以受“减少动态效果”约束的 CSS 伪元素动画呈现。

**Tech Stack:** Go、`net/http`、Wails、React、MUI、Emotion `sx` 动画、Vitest、Go 标准测试。

---

### Task 1: 设置模型与 HTTP 服务启用规则

**Files:**
- Modify: `internal/settings/settings.go`
- Modify: `internal/settings/settings_test.go`
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: 写失败测试**

在 `internal/settings/settings_test.go` 增加：标题变化模式且 `httpServiceEnabled=true` 时端口为零应失败；标题变化模式且开关关闭时允许零端口；旧 JSON 缺少开关字段默认关闭。在 `app_test.go` 增加保存独立服务设置后 `APIURL()` 非空、关闭开关后为空、状态 HTTP 模式即使开关为假也保持服务开启的测试。

**Step 2: 运行失败测试**

Run: `go test . ./internal/settings -run 'Test.*HTTP.*(Enabled|Service)'`

Expected: FAIL，新增 `HTTPServiceEnabled` 字段和服务生命周期尚不存在。

**Step 3: 最小实现**

在 `settings.Settings` 增加：

```go
HTTPServiceEnabled bool `json:"httpServiceEnabled"`
```

在 `Validate` 中，当 `HTTPServiceEnabled` 或 `StatusManagementMode == StatusManagementModeHTTP` 时要求 `StatusManagementHTTPPort` 为 `1–65535`。在 `app.applyStatusSettings` 中以：

```go
shouldRunHTTP := current.HTTPServiceEnabled || current.StatusManagementMode == settings.StatusManagementModeHTTP
```

决定调用 `statusHTTP.Configure(port)` 或 `statusHTTP.Close()`；再独立设置实时状态模式。

**Step 4: 运行通过测试**

Run: `go test . ./internal/settings -run 'Test.*HTTP.*(Enabled|Service)'`

Expected: PASS。

### Task 2: 任务列表与详情 HTTP 契约

**Files:**
- Modify: `internal/realtime/http.go`
- Modify: `internal/realtime/http_test.go`
- Modify: `app.go`
- Modify: `app_test.go`

**Step 1: 写失败测试**

在 `internal/realtime/http_test.go` 添加任务目录回调，并覆盖：

```go
GET /api/v1/tasks?status=pending
GET /api/v1/tasks?status=running
GET /api/v1/tasks?status=completed
GET /api/v1/tasks/{taskID}
```

断言列表仅返回请求状态、详情返回完整字段；无效 `status` 得到 `400`，未知任务得到 `404`，对两个路径使用非 GET 得到 `405`。在 `app_test.go` 覆盖接口读取实际仓储中的未执行、执行中、已完成任务。

**Step 2: 运行失败测试**

Run: `go test ./internal/realtime . -run 'TestHTTPServer(Task|.*Task)'`

Expected: FAIL，路由与任务目录回调不存在。

**Step 3: 最小实现**

在 `internal/realtime/http.go` 定义与领域模型解耦的 `TaskResource` 及 `TaskCatalog` 回调：

```go
type TaskCatalog struct {
    List func() ([]TaskResource, error)
    Get  func(taskID string) (TaskResource, bool, error)
}
```

扩展 `HTTPServerOptions` 并处理以下 GET 路径：

```text
/api/v1/tasks
/api/v1/tasks/{taskId}
```

使用 `status` 查询参数筛选 `pending`、`running`、`completed`，并用既有 JSON 错误格式处理 `400`、`404`、`405` 与 `500`。在 `app.go` 中由仓储任务映射为 `TaskResource` 后传入 `NewHTTPServer`。

**Step 4: 运行通过测试**

Run: `go test ./internal/realtime . -run 'TestHTTPServer(Task|.*Task)'`

Expected: PASS。

### Task 3: 设置界面与 Wails 类型

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`
- Generated: `frontend/wailsjs/go/models.ts`

**Step 1: 写失败测试**

在 `App.test.tsx` 增加：标题变化模式可打开“启用本机 HTTP 服务”并显示端口；选择 HTTP 状态管理后开关为开启且不可编辑；独立服务配置保存时传递 `httpServiceEnabled: true` 及端口；端口缺失时展示后端错误。

**Step 2: 运行失败测试**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: FAIL，设置记录缺字段且页面未展示独立 HTTP 服务配置。

**Step 3: 最小实现**

给 `SettingsRecord` 增加：

```ts
httpServiceEnabled: boolean
```

在“实时状态”页把端口字段显示条件改为“独立服务开启或状态管理为 HTTP”。标题变化模式用可切换的 `Switch` 控制独立服务；HTTP 模式固定开启并显示“由状态管理自动开启”。保持 HTTP 使用说明入口在服务启用时可访问。

**Step 4: 运行通过测试**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: PASS。

### Task 4: 未读状态心跳扩散动效

**Files:**
- Modify: `frontend/src/components/TerminalStatusDot.tsx`
- Modify: `frontend/src/components/TaskTree.test.tsx`

**Step 1: 写失败测试**

为 `TerminalStatusDot` 增加稳定的 `data-status` 属性。在任务树测试中渲染 `unread` 终端，断言状态点拥有 `data-status="unread"`；渲染其他状态时断言对应状态值保留。

**Step 2: 运行失败测试**

Run: `cd frontend && npm test -- --run src/components/TaskTree.test.tsx`

Expected: FAIL，状态点没有动效所需的稳定状态标识。

**Step 3: 最小实现**

在 `TerminalStatusDot.tsx` 为圆点加入 `data-status={status}`。仅当 `status === 'unread'` 时，以 `::before` 绘制紫色半透明环并执行约 1.4 秒的缩放淡出动画；添加：

```ts
'@media (prefers-reduced-motion: reduce)': { '&::before': { animation: 'none' } }
```

使动效不改变文档流、圆点尺寸或 `aria-label`。

**Step 4: 运行通过测试**

Run: `cd frontend && npm test -- --run src/components/TaskTree.test.tsx`

Expected: PASS。

### Task 5: 文档、绑定与完整验证

**Files:**
- Modify: `README.md`
- Modify: `frontend/src/App.tsx`
- Generated: `frontend/wailsjs/go/main/App.js`
- Generated: `frontend/wailsjs/go/main/App.d.ts`
- Generated: `frontend/wailsjs/go/models.ts`

**Step 1: 更新中文文档**

在 README 和应用内 HTTP 说明增加：独立服务开关、状态 HTTP 自动开启规则、`GET /api/v1/tasks` 的三种 `status` 筛选值、详情路径、成功响应字段、`400/404/405/500` 错误语义与 `curl` 示例。

**Step 2: 重新生成绑定**

Run: `./scripts/build-linux.sh amd64`

Expected: Wails 绑定包含 `httpServiceEnabled`，Linux 包生成成功。

**Step 3: 完整验证**

Run:

```bash
go test -race ./...
cd frontend && npm test && npm run build
cd .. && git diff --check
```

Expected: 全部通过且无行尾空白。根据仓库约束，本任务不自动创建 Git 提交。
