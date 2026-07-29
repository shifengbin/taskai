# 信息级动态参数与分类稳定展示 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让可复用信息保存专属动态参数默认值，并让额外信息管理的分类与搜索在各种宽度下稳定展示。

**Architecture:** 在 `task.ExtraInfo` 复用带值参数结构，并由服务按参数来源构建任务快照。前端信息编辑器维护该集合，任务编辑器据来源限制定义编辑；管理弹窗将模板定义和信息分组拆分为互不干扰的可折叠区域。

**Tech Stack:** Go、JSON 持久化、Wails、React、TypeScript、Material UI、Vitest。

---

### Task 1: 领域模型和持久化兼容

**Files:**
- Modify: `internal/task/model.go`
- Test: `internal/task/model_test.go`
- Modify: `internal/storage/repository.go`
- Test: `internal/storage/repository_test.go`

**Step 1: 写出失败的领域与仓储测试**

覆盖信息级参数的默认值规范化、与模板字段/参数冲突的拒绝、任务快照复制信息参数，以及缺少 `parameters` 的历史信息加载后变为空集合。

**Step 2: 运行定向 Go 测试并确认失败原因是未实现信息参数。**

Run: `go test ./internal/task ./internal/storage -run 'Information.*Parameter|ExtraInfo.*Parameter' -count=1`

**Step 3: 实现最小领域和归一化逻辑**

为 `ExtraInfo` 加入参数集合；在信息规范化中统一修剪、校验键冲突并保证非 nil；任务快照构建把信息参数追加在模板参数后。

**Step 4: 重跑定向测试并确认通过。**

### Task 2: 任务快照来源校验

**Files:**
- Modify: `internal/lifecycle/service.go`
- Test: `internal/lifecycle/service_test.go`
- Test: `app_test.go`

**Step 1: 写出失败测试**

验证创建任务时带入信息默认参数，且前端/绑定提交不能篡改信息级参数的显示名称或必填标记；验证任务级新增参数仍可追加。

**Step 2: 运行定向测试并确认失败。**

Run: `go test ./internal/lifecycle . -run 'Information.*Parameter|ExtraInfo.*Snapshot' -count=1`

**Step 3: 实现参数来源拆分与任务快照合成**

将拆分函数同时接收信息与模板，按模板、信息和任务级来源校验定义；调用 `NewTaskExtraInfo` 时保留信息参数默认值且接受任务改写后的值。

**Step 4: 重跑定向测试并确认通过。**

### Task 3: Wails 类型和信息编辑表单

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: 写出失败的前端交互测试**

覆盖新建 Git 信息时新增“环境”参数并保存参数默认值；选择该信息创建任务后只显示“环境”值输入且预填默认值。

**Step 2: 运行该测试并确认失败。**

Run: `npm test -- --run src/App.test.tsx -t '信息级动态参数'`

**Step 3: 实现类型、深拷贝、信息参数编辑与任务来源识别**

在信息草稿创建/克隆时复制参数；信息弹窗使用响应式参数编辑区；任务快照从信息参数初始化，并把模板和信息参数均设为定义只读。

**Step 4: 重跑该测试并确认通过。**

### Task 4: 分类模板折叠与稳定搜索

**Files:**
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: 写出失败测试**

验证“分类模板”面板可整体折叠；搜索一个分类的信息时，其他分类标题仍可见而命中分类自动展开。

**Step 2: 运行该测试并确认失败。**

Run: `npm test -- --run src/App.test.tsx -t '分类模板|搜索保留分类'`

**Step 3: 实现最小界面改动**

将模板列表包裹为独立 Accordion；模板行改为可换行的响应式网格；所有信息分组始终渲染，仅筛选其信息行并展示空搜索状态。

**Step 4: 重跑该测试并确认通过。**

### Task 5: 文档和完整验证

**Files:**
- Modify: `README.md`
- Modify: `openspec/changes/refine-extra-info-template-flow/tasks.md`

**Step 1: 更新 README 与 OpenSpec 勾选状态**

说明信息可保存专属动态参数默认值，任务中自动带入且只编辑值；勾选本次完成任务。

**Step 2: 运行完整验证**

Run: `go test ./... -count=1`

Run: `npm test -- --run && npm run build`

Run: `openspec validate refine-extra-info-template-flow --strict --no-interactive`

Run: `./scripts/build-linux.sh`

**Step 3: 检查最终变更**

Run: `git diff --check`
