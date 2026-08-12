# Git 扫描深度 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在设置中保存 Git 最大扫描深度，并用它限制任务 Git 标签的目录遍历。

**Architecture:** `settings.Settings` 持久化最大扫描深度并为缺失旧值回退到 2。应用层将该设置传入 `repositorygit.Service.List`，服务以任务目录为第 1 层，在到达上限时停止进入子目录；前端在“工作区与外观”中编辑同一设置字段。

**Tech Stack:** Go、Wails、React、Vitest、OpenSpec。

---

### Task 1: 设置默认值与校验

**Files:**
- Modify: `internal/settings/settings.go`
- Test: `internal/settings/settings_test.go`

**Step 1: Write the failing test**

断言默认设置为 2，零值归一为 2，低于 1 或高于 10 的保存值被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/settings -run 'Test(DefaultUsesGitScanDepth|ValidateGitScanDepth)'`

Expected: FAIL，因为设置尚无 Git 扫描深度。

**Step 3: Write minimal implementation**

为 `Settings` 添加 `gitScanDepth`，并在 `Default` 和 `Validate` 中设置默认值和范围校验。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/settings -run 'Test(DefaultUsesGitScanDepth|ValidateGitScanDepth)'`

Expected: PASS。

### Task 2: 限制仓库遍历

**Files:**
- Modify: `internal/repositorygit/service.go`
- Test: `internal/repositorygit/service_test.go`

**Step 1: Write the failing test**

建立第 1、2、3 层的仓库，断言深度 2 不包含第 3 层，深度 3 包含它。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/repositorygit -run TestServiceLimitsRepositoryScanDepth`

Expected: FAIL，因为现有遍历没有深度参数。

**Step 3: Write minimal implementation**

让 `List` 和目录发现接收最大深度，按相对路径目录段数计算层级，到达上限后跳过子目录。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/repositorygit -run TestServiceLimitsRepositoryScanDepth`

Expected: PASS。

### Task 3: 应用绑定和设置界面

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Step 1: Write the failing tests**

断言应用层将保存的深度传入服务；断言设置显示默认值 2 并保存用户输入的深度。

**Step 2: Run tests to verify they fail**

Run: `go test . -run TestAppListsGitRepositoriesWithConfiguredScanDepth` and `npm test -- --run src/App.test.tsx`

Expected: FAIL，因为绑定和界面尚未携带字段。

**Step 3: Write minimal implementation**

应用层从已保存设置读取深度，前端类型和工作区设置字段同步更新；重新生成 Wails 模型。

**Step 4: Run tests to verify they pass**

Run: `go test . -run TestAppListsGitRepositoriesWithConfiguredScanDepth` and `npm test -- --run src/App.test.tsx`

Expected: PASS。

### Task 4: 完整验证

**Files:**
- Modify: `openspec/changes/task-repository-git-management/tasks.md`

**Step 1: Run verification**

Run: `go test -race ./...`, `npm test -- --run`, `npm run build`, `openspec validate task-repository-git-management --strict`。

**Step 2: Mark validation tasks**

将 OpenSpec 中已完成的深度与验证任务标为完成；按项目流程运行 Wails 开发模式、构建并启动程序，再等待用户确认合并。
