# HTTP 复选框响应类型 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 使任务详情 HTTP 接口以 JSON 布尔值返回复选框动态参数，同时保持文本参数与内部快照兼容。

**Architecture:** 任务模型继续以字符串保存参数值。应用层在 `httpExtraInfo` 序列化前检查参数 `inputType`，仅将 `checkbox` 的规范化值映射为 `bool`，其余值保持字符串。

**Tech Stack:** Go、标准库 `encoding/json`、HTTP 测试、OpenSpec、README。

---

### Task 1: HTTP 类型回归测试

**Files:**
- Modify: `app_test.go`

**Step 1: 写出失败的接口测试**

创建带 Git 文本参数和任务级复选框参数的任务；请求 `/api/v1/tasks/{taskId}` 并断言复选框反序列化为 `bool`，文本参数仍为 `string`。

**Step 2: 运行测试并确认失败**

Run: `go test . -run TestAppHTTPTaskDetailReturnsCheckboxParameterAsBoolean -count=1`

Expected: FAIL，因为接口将复选框值输出为字符串。

### Task 2: HTTP 输出边界转换

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: 实现最小映射改动**

将 HTTP 附加信息容器改为可保存 JSON 标量的映射。对 `checkbox` 参数输出 `parameter.Value == "true"` 的布尔结果；固定字段和文本参数保持原字符串值。

**Step 2: 重跑定向测试**

Run: `go test . -run 'TestAppHTTPTaskDetail(FlattensExtraInfoByCatalogue|ReturnsCheckboxParameterAsBoolean)' -count=1`

Expected: PASS。

### Task 3: 文档与完整验证

**Files:**
- Modify: `README.md`
- Modify: `openspec/changes/refine-extra-info-template-flow/{design.md,tasks.md,specs/task-extra-info/spec.md}`

**Step 1: 更新接口契约说明**

说明复选框参数在任务详情 HTTP 响应中是 JSON 布尔值，而内部存储仍兼容字符串值。

**Step 2: 完整验证**

Run: `go test ./... -count=1`

Run: `npm test -- --run`

Run: `openspec validate refine-extra-info-template-flow --strict --no-interactive`

Run: `./scripts/build-linux.sh && git diff --check`
