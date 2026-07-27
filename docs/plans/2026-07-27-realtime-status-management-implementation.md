# 实时状态管理实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 为任务终端提供可聚合的实时状态、默认标题变化判定和可选的本机 HTTP 上报接口。

**架构：** 后端实时状态服务持有临时终端状态、任务直接覆盖、选中终端和 1.5 秒计时器；前端继续解析标题并报告真实变化，HTTP 请求与终端生命周期事件也写入同一服务。服务通过 Wails 事件向前端投影状态，任务树按展开状态决定显示任务或终端圆点。

**技术栈：** Go、Wails v2、`net/http`、React、TypeScript、Vitest、Go 标准测试工具。

---

### 任务 1：设置模型和实时状态领域模型

**文件：**
- 修改：`internal/settings/settings.go`
- 测试：`internal/settings/settings_test.go`
- 新建：`internal/realtime/status.go`
- 测试：`internal/realtime/status_test.go`

**步骤 1：编写失败测试**

为设置默认模式、HTTP 模式端口校验、旧 JSON 缺失字段，以及状态优先级、任务覆盖、未读清除、超时与旧计时器失效编写测试。

**步骤 2：验证失败**

运行：`go test ./internal/settings ./internal/realtime`

预期：因状态管理字段或实时状态服务尚未定义而失败。

**步骤 3：实现最小领域模型**

新增状态方式和端口设置；实现只保存在内存的状态服务、版本化计时器、任务聚合和状态变更回调。

**步骤 4：验证通过**

运行：`go test ./internal/settings ./internal/realtime`

预期：全部通过。

### 任务 2：终端标识、环境和关闭原因

**文件：**
- 修改：`internal/terminal/types.go`
- 修改：`internal/terminal/manager.go`
- 修改：`internal/terminal/backend_unix.go`
- 修改：`internal/terminal/backend_windows.go`
- 测试：`internal/terminal/manager_test.go`
- 测试：`internal/terminal/backend_unix_test.go`
- 测试：`internal/terminal/backend_windows_test.go`

**步骤 1：编写失败测试**

覆盖在进程启动前分配的终端 ID、启动环境变量、主动关闭、任务结束关闭及自然退出的退出原因。

**步骤 2：验证失败**

运行：`go test ./internal/terminal`

预期：新字段或关闭原因尚未实现导致断言失败。

**步骤 3：实现最小生命周期扩展**

让 `StartRequest` 携带预生成 ID 与环境，并让平台后端继承环境；管理器记录预期关闭原因并在事件中传播。

**步骤 4：验证通过**

运行：`go test ./internal/terminal`

预期：全部通过。

### 任务 3：HTTP 状态服务和应用层接入

**文件：**
- 新建：`internal/realtime/http.go`
- 测试：`internal/realtime/http_test.go`
- 修改：`app.go`
- 测试：`app_test.go`
- 修改：`internal/application/contracts.go`

**步骤 1：编写失败测试**

覆盖回环监听、查询、任务和终端更新、错误状态码、端口冲突、原子重配、应用启动/保存设置、终端环境变量和退出事件。

**步骤 2：验证失败**

运行：`go test . ./internal/realtime`

预期：HTTP 服务和新的应用绑定入口尚未实现。

**步骤 3：实现最小 HTTP 与应用编排**

实现 `/api/v1` 路由、JSON 响应和回环监听；应用在启动与保存设置时重配服务，在终端创建、标题活动、选择、关闭和任务结束时同步实时状态并发布 Wails 事件。

**步骤 4：验证通过**

运行：`go test . ./internal/realtime`

预期：全部通过。

### 任务 4：前端状态投影和任务树

**文件：**
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/state.ts`
- 修改：`frontend/src/api.ts`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/components/TerminalStatusDot.tsx`
- 修改：`frontend/src/components/TaskTree.tsx`
- 测试：`frontend/src/state.test.ts`
- 测试：`frontend/src/App.test.tsx`
- 测试：`frontend/src/components/TaskTree.test.tsx`

**步骤 1：编写失败测试**

覆盖仅不同标题上报活动、状态事件版本去重、选中未读转空闲、主动关闭不变红，以及收起任务的汇总圆点和展开任务的终端圆点。

**步骤 2：验证失败**

运行：`cd frontend && npm test -- --run src/state.test.ts src/App.test.tsx src/components/TaskTree.test.tsx`

预期：新实时状态类型、事件处理和视觉断言尚未满足。

**步骤 3：实现最小前端投影**

接收实时状态事件，标题仅在实际改变时上报，选择行为同步后端；状态圆点按四色和展开层级显示。

**步骤 4：验证通过**

运行：`cd frontend && npm test -- --run src/state.test.ts src/App.test.tsx src/components/TaskTree.test.tsx`

预期：全部通过。

### 任务 5：设置界面、使用文档和绑定

**文件：**
- 修改：`frontend/src/App.tsx`
- 测试：`frontend/src/App.test.tsx`
- 生成：`frontend/wailsjs/go/main/App.js`
- 生成：`frontend/wailsjs/go/main/App.d.ts`
- 生成：`frontend/wailsjs/go/models.ts`

**步骤 1：编写失败测试**

覆盖默认标题变化方式、HTTP 端口条件字段、保存失败提示和 HTTP 模式中的“查看使用文档”弹窗。

**步骤 2：验证失败**

运行：`cd frontend && npm test -- --run src/App.test.tsx`

预期：设置控件和文档入口尚未存在。

**步骤 3：实现设置和文档**

添加实时状态设置页，在 HTTP 模式下显示端口与文档入口；文档说明接口、环境变量、`curl`、错误语义、本机限制和既有终端限制。

**步骤 4：重新生成并验证绑定**

运行：`./scripts/build-linux.sh amd64`

预期：Wails 绑定包含新增应用方法和设置字段。

### 任务 6：中文文档和完整验证

**文件：**
- 修改：`README.md`
- 修改：`openspec/changes/add-realtime-status-management/tasks.md`

**步骤 1：补充中文使用说明**

记录状态颜色及优先级、标题判定、HTTP 设置、环境变量、接口和端口变更限制。

**步骤 2：标记 OpenSpec 任务**

每完成一个 OpenSpec 子任务立即将对应复选框更新为 `- [x]`。

**步骤 3：执行完整验证**

运行：`go test -race ./...`、`cd frontend && npm test && npm run build`、`./scripts/build-linux.sh amd64`、`openspec validate add-realtime-status-management --strict`、`git diff --check`。

预期：所有测试、构建与严格规格校验通过。
