# 动态参数复选框类型 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让动态参数支持复选框输入，并在模板、信息和任务快照中一致保存 `true` 或 `false`。

**Architecture:** 参数定义和带值参数使用 `inputType` 保存输入语义，模型负责旧数据归一化、布尔值校验和必填规则。React 根据该字段选择文本框或复选框，任务快照保留来源参数的类型。

**Tech Stack:** Go、JSON 持久化、Wails、React、TypeScript、Material UI、Vitest。

---

### Task 1: 领域模型与快照校验

**Files:**
- Modify: `internal/task/model.go`
- Test: `internal/task/model_test.go`

**Step 1: 写出失败的模型测试**

覆盖旧参数归一化为文本类型、复选框强制非必填并默认 `false`、非法复选框值被拒绝，以及信息和模板参数进入任务快照时保留类型和值。

**Step 2: 运行失败测试**

Run: `go test ./internal/task -run 'Checkbox|ExtraInfo.*Parameter' -count=1`

**Step 3: 实现最小模型逻辑**

增加 `inputType` 常量和字段；在模板、信息、任务快照的归一化路径中校验类型和布尔值，并在快照未提供模板复选框值时使用 `false`。

**Step 4: 重跑模型测试**

Run: `go test ./internal/task -run 'Checkbox|ExtraInfo.*Parameter' -count=1`

### Task 2: 前端参数编辑与任务输入

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: 写出失败的前端交互测试**

验证模板参数可选复选框且不出现必填控件，信息复选框默认未选中，以及任务选择该信息后能切换为 `true` 并保存。

**Step 2: 运行失败测试**

Run: `npm test -- --run src/App.test.tsx -t '复选框动态参数'`

**Step 3: 实现最小界面改动**

为三个参数新增入口添加类型初值和类型选择器；类型为复选框时使用 `Checkbox`，隐藏必填控件并在状态更新中写入 `true` 或 `false`；类型切换为复选框时清空必填状态并重置值。

**Step 4: 重跑前端测试**

Run: `npm test -- --run src/App.test.tsx -t '复选框动态参数'`

### Task 3: 绑定、文档与完整验证

**Files:**
- Modify: `frontend/wailsjs/go/models.ts`
- Modify: `README.md`
- Modify: `openspec/changes/refine-extra-info-template-flow/{design.md,tasks.md,specs/task-extra-info/spec.md}`

**Step 1: 重新生成 Wails 绑定并更新文档**

说明复选框参数使用布尔字符串、不支持必填设置且旧参数默认文本类型。

**Step 2: 运行完整验证**

Run: `go test ./... -count=1`

Run: `npm test -- --run && npm run build`

Run: `openspec validate refine-extra-info-template-flow --strict --no-interactive`

Run: `./scripts/build-linux.sh && git diff --check`
