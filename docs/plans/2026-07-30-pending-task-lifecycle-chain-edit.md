# 未执行任务命令链编辑实施计划

> **For Codex:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**目标：** 允许未执行任务在编辑时修改生命周期命令链，同时保证执行中和已完成任务无法修改。

**架构：** 复用生命周期服务现有的链选择范围校验，提供一个只接收任务详情与链选择的专用更新入口。该入口只允许 `pending` 状态，前端以任务实际状态决定选择器是否可用；执行中和已完成任务仍由前端提示和后端校验双重保护。

**技术栈：** Go、Wails、React、TypeScript、Vitest、OpenSpec。

---

### 任务 1：为未执行任务链更新建立后端失败测试

**文件：**
- 修改：`internal/lifecycle/service_test.go`
- 修改：`app_test.go`

**步骤：**
1. 添加服务测试：未执行任务更新有效链选择成功且不产生 `updateTask` 执行记录；执行中和已完成任务更新链选择被拒绝；范围不匹配选择继续被拒绝。
2. 添加应用绑定测试：公开入口接受未执行任务的更新并拒绝其他状态。
3. 运行 `go test ./internal/lifecycle .`，确认新增断言因缺失专用更新入口而失败。

### 任务 2：实现状态受限的链选择更新入口

**文件：**
- 修改：`internal/lifecycle/service.go`
- 修改：`app.go`
- 自动生成：`frontend/wailsjs/go/main/App.{js,d.ts}`

**步骤：**
1. 在生命周期服务实现接收任务详情和链选择的更新方法：查询任务，拒绝非 `pending` 状态，复用范围校验后持久化详情及链选择。
2. 不调用执行中任务的更新流程，确保 `updateTask` 仅由既有执行中详情更新入口触发。
3. 在 Wails `App` 暴露同名受限方法，并重跑 `go test ./internal/lifecycle .`，确认绿色。

### 任务 3：按编辑任务状态控制前端并建立交互测试

**文件：**
- 修改：`frontend/src/api.ts`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 添加失败交互测试：编辑未执行任务时可更换链并调用专用 API；编辑执行中和已完成任务时选择器禁用且保存不调用该 API。
2. 运行 `npm test -- --run src/App.test.tsx`，确认断言因当前统一禁用及 API 缺失而失败。
3. 在 API 封装中加入 Wails 入口；在表单中只在 `editingTask?.status !== 'pending'` 时禁用选择器，并在保存未执行任务时传递链选择。
4. 重跑 `npm test -- --run src/App.test.tsx`，确认三种状态边界绿色。

### 任务 4：完成增量规格与交付验证

**文件：**
- 修改：`openspec/changes/add-task-lifecycle-command-chains/{proposal.md,design.md,tasks.md}`
- 修改：`openspec/changes/add-task-lifecycle-command-chains/specs/task-lifecycle-command-chains/spec.md`
- 修改：`README.md`

**步骤：**
1. 将已批准的状态边界写入中文设计、规格、README，并登记增量任务。
2. 运行 `openspec validate add-task-lifecycle-command-chains --strict`。
3. 运行 `go test -race ./...`、`npm test`、`npm run build`。
4. 运行 `./scripts/build-linux.sh` 生成 Wails 绑定和 Linux 包；复跑严格规格校验与 `git diff --check`（排除生成器固有尾随空白），然后勾选增量任务。
