# 任务 HTTP 终端详情 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让任务详情 HTTP 接口返回当前活动终端的 ID、执行命令和实时状态。

**Architecture:** 终端管理器保存创建会话时的有效命令，并暴露按 ID 排序的活动会话快照。应用层只在详情查询时将该快照与实时服务状态组合，填入 HTTP 资源的可选 `terminals` 数组；任务列表不设置该字段。

**Tech Stack:** Go、标准库 HTTP/JSON、终端管理器、实时状态服务、OpenSpec、README。

---

### Task 1: 写出 HTTP 详情失败测试

**Files:**
- Modify: `app_test.go`

**Step 1: 写出失败测试**

使用可控终端后端为运行中任务创建普通和命令终端，设置 `working` 与 `idle` 状态；请求任务详情，断言 `terminals` 含 ID、命令和状态，任务列表不含该字段。

**Step 2: 运行并确认失败**

Run: `go test . -run TestAppHTTPTaskDetailIncludesActiveTerminalDetails -count=1`

Expected: FAIL，因为 `TaskResource` 尚未定义或填充 `terminals`。

### Task 2: 暴露活动终端并映射 HTTP 资源

**Files:**
- Modify: `internal/terminal/types.go`
- Modify: `internal/terminal/manager.go`
- Modify: `internal/realtime/http.go`
- Modify: `app.go`
- Test: `app_test.go`
- Test: `internal/terminal/manager_test.go`

**Step 1: 实现最小会话快照**

创建会话时记录有效命令；未传命令时记录 Shell 路径。新增活动会话查询方法并按终端 ID 排序，退出时沿用现有会话移除逻辑。

**Step 2: 填充详情响应**

为 HTTP 资源增加可选终端数组。应用层仅在详情查询中读取活动会话，并通过实时服务映射状态；没有活动终端时仍输出 `[]`。

**Step 3: 重跑定向测试**

Run: `go test . -run TestAppHTTPTaskDetailIncludesActiveTerminalDetails -count=1`

Run: `go test ./internal/terminal -run 'Snapshot|Command' -count=1`

Expected: PASS。

### Task 3: 文档与完整验证

**Files:**
- Modify: `README.md`
- Modify: `openspec/changes/refine-extra-info-template-flow/{design.md,tasks.md,specs/task-extra-info/spec.md}`

**Step 1: 说明终端详情字段和生命周期边界**

更新任务详情接口文档，说明字段、状态枚举、普通终端 Shell 值以及仅返回活动会话的语义。

**Step 2: 完整验证**

Run: `go test ./... -count=1`

Run: `npm test -- --run`

Run: `openspec validate refine-extra-info-template-flow --strict --no-interactive`

Run: `./scripts/build-linux.sh && git diff --check`
