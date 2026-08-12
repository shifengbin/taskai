# Git 同步状态 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 准确识别已同步的 Git 仓库并禁用其同步按钮，同时在没有工作目录的任务详情中隐藏 Git 标签。

**Architecture:** 仓库服务复用现有远程分支查询获得远程提交号，再核对本地跟踪分支、当前提交与提交差异；只有三者一致时输出 `synchronized`。前端依据该字段禁用“同步远程”按钮，并仅在任务具备工作目录时渲染 Git 标签。

**Tech Stack:** Go、Wails、React、Vitest、OpenSpec。

---

### Task 1: 计算仓库同步状态

**Files:**
- Modify: `internal/repositorygit/service.go`
- Test: `internal/repositorygit/service_test.go`

**Step 1: Write the failing tests**

为已发布且干净的仓库断言 `Synchronized` 为真；本地新提交但未推送、远程被其他克隆更新、以及未设置上游分支时断言为假。

**Step 2: Run the tests to verify they fail**

Run: `go test ./internal/repositorygit -run 'TestService(ReportsSynchronizedRepository|DoesNotReportSynchronizedWhenLocalOrRemoteDiffer|SynchronizesExistingRemoteBranchWithoutLocalUpstream)'`

Expected: FAIL，因为 `Repository` 尚无同步状态且服务未比较提交号。

**Step 3: Implement the minimal state calculation**

- 为 `Repository` 增加 JSON 字段 `synchronized`。
- 让远程分支查询返回远程提交号，并保持“不存在分支”与“读取失败”的既有区别。
- 当上游正是选定远程的当前分支、本地远程跟踪引用与远程提交号一致、并且 `remote/branch...HEAD` 的领先和落后计数均为零时，设为已同步。
- 在任一步无法可靠确认时返回未同步，不改变原有提交、发布、同步操作选择。

**Step 4: Run the focused tests**

Run: `go test ./internal/repositorygit -run 'TestService(ReportsSynchronizedRepository|DoesNotReportSynchronizedWhenLocalOrRemoteDiffer|SynchronizesExistingRemoteBranchWithoutLocalUpstream)'`

Expected: PASS。

### Task 2: 显示禁用同步按钮与条件标签

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing tests**

在已有多仓库详情测试中返回 `synchronized: true` 的同步仓库，断言名为“同步远程”的按钮禁用；新增已保存路径但任务目录尚未创建的详情测试，断言不显示“Git”标签且不调用仓库列表接口。

**Step 2: Run the tests to verify they fail**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: FAIL，因为类型、按钮禁用条件和标签渲染条件尚未更新。

**Step 3: Implement the minimal UI**

- 为 `TaskGitRepository` 添加 `synchronized`。
- 仅当任务有非空 `workspacePath` 且目录实际存在时渲染 Git 标签及内容；切换任务时受控标签重置为“项目信息”，并拒绝选择已隐藏的 Git 标签。
- 当仓库动作是 `sync` 且 `synchronized` 为真时，为文字保持“同步远程”的主按钮传入 `disabled`。

**Step 4: Run the focused frontend tests**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: PASS。

### Task 3: 更新绑定与行为规格

**Files:**
- Modify: `frontend/wailsjs/go/models.ts`
- Modify: `openspec/changes/task-repository-git-management/specs/task-repository-git-management/spec.md`
- Modify: `openspec/changes/task-repository-git-management/tasks.md`

**Step 1: Regenerate bindings**

Run: `wails generate module`

Expected: `repositorygit.Repository` 的生成模型带有 `synchronized`。

**Step 2: Update specifications**

补充“已同步时禁用同步按钮”和“无工作目录不显示 Git 标签”两个场景，并把这两项实施任务标记完成。

**Step 3: Validate the proposal**

Run: `openspec validate task-repository-git-management --strict`

Expected: PASS。

### Task 4: 完整验证和运行验证

**Files:**
- Modify: `openspec/changes/task-repository-git-management/tasks.md`

**Step 1: Run automated validation**

Run: `go test -race ./...`, `cd frontend && npm test -- --run && npm run build`, `openspec validate task-repository-git-management --strict`

Expected: 全部 PASS。

**Step 2: Run the application**

Run: `wails dev`

Expected: 任务有工作目录时显示 Git 标签；已同步仓库的“同步远程”不可点击；无工作目录任务不显示 Git 标签。

**Step 3: Build and launch the executable**

Run: `./scripts/build-linux.sh amd64`，然后运行 `./build/bin/taskai`。

Expected: 构建成功且应用能够启动。

**Step 4: Record completion and await merge confirmation**

将验证项标为完成，保留 worktree 与改动，等待用户确认后再合并、归档并提交。
