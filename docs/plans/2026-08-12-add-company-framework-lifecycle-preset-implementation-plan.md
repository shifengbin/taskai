# 公司框架生命周期预设 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为新安装提供默认选中的“公司框架”生命周期预设，同时完整保留已有用户的预设、默认选择和仓库更新链名称。

**Architecture:** 新安装仍由 `settings.Default(...)` 生成初始数据，但新安装专用的生命周期链与预设构造值必须和旧数据规范化、历史迁移使用的兼容构造值分开。界面继续消费现有设置数据，不新增 Wails API；测试从设置默认值、持久化兼容、任务映射和真实 Wails 界面四层锁定行为。

**Tech Stack:** Go、Wails v2、React 18、Vitest、OpenSpec、Chrome DevTools。

---

### Task 1: 创建隔离 worktree

**Files:**
- Verify: `.gitignore`
- Reference: `AGENTS.md`
- Reference: `openspec/changes/add-company-framework-lifecycle-preset/`

**Step 1: 检查当前分支和工作树**

Run: `git status --short && git branch --show-current && git worktree list`

Expected: 识别当前项目分支、用户已有改动和已存在 worktree，不覆盖任何无关修改。

**Step 2: 准备项目内 worktree 目录**

确认 `.worktrees/` 已被 Git 忽略；若未忽略，先将 `/.worktrees/` 加入 `.gitignore`。随后创建 `.worktrees` 目录。

**Step 3: 创建功能 worktree**

Run: `git worktree add .worktrees/add-company-framework-lifecycle-preset -b feat/add-company-framework-lifecycle-preset`

Expected: 新 worktree 位于项目内 `.worktrees/add-company-framework-lifecycle-preset`，分支为 `feat/add-company-framework-lifecycle-preset`。

**Step 4: 确认 OpenSpec 文档可用**

将当前未提交的本变更 OpenSpec 文档安全带入功能 worktree，不覆盖同名用户文件；在 worktree 内运行：

Run: `openspec status --change add-company-framework-lifecycle-preset`

Expected: proposal、design、specs、tasks 均完成，变更可实施。

### Task 2: 用失败测试定义新安装种子

**Files:**
- Modify: `internal/settings/settings_test.go`
- Test: `internal/settings/settings_test.go`

**Step 1: 添加公司框架预设失败测试**

扩展或拆分 `TestDefaultIncludesLifecyclePreset`，断言：

```go
companyPreset := LifecyclePreset{
    ID:   CompanyFrameworkLifecyclePresetID,
    Name: "公司框架",
    Chains: map[LifecycleHook]string{
        LifecycleHookBeforeStart: LifecycleChainIterationsAIID,
        LifecycleHookPostEnd:     LifecycleChainDeleteWorkspaceID,
        LifecycleHookUpdateTask:  LifecycleChainUpdateRepositoriesID,
    },
}
```

同时断言预设列表仍包含原“默认预设”，`DefaultLifecyclePresetID` 指向公司框架，且 `postStart`、`beforeEnd` 没有映射。

**Step 2: 添加新安装链名称失败测试**

在 `TestDefaultSeedsDefaultBranchTemplateAndRepositoryPresetChains` 中断言稳定 ID `LifecycleChainUpdateRepositoriesID` 对应名称为“更新框架仓库”，适用范围仍只有 `updateTask`，命令顺序与参数保持：更新默认分支、生成清单文件、`dir=workspaces` Git 克隆。

**Step 3: 运行测试确认失败**

Run: `go test ./internal/settings -run 'TestDefault(IncludesLifecyclePreset|SeedsDefaultBranchTemplateAndRepositoryPresetChains)'`

Expected: FAIL，现有默认值只有“默认预设”，且链名仍为“更新仓库”。

### Task 3: 实现新安装专用默认值

**Files:**
- Modify: `internal/settings/settings.go`
- Test: `internal/settings/settings_test.go`

**Step 1: 增加稳定预设 ID**

在现有预置常量旁增加：

```go
CompanyFrameworkLifecyclePresetID = "preset.lifecycle-preset.company-framework"
```

**Step 2: 拆分链构造入口**

保留旧数据规范化和迁移需要的兼容链定义，其仓库更新链名称仍为“更新仓库”。增加仅供 `Default(...)` 使用的新安装链构造入口，复用相同稳定 ID、命令引用、适用范围和参数，只把初始名称设为“更新框架仓库”。避免用调用方环境或持久化文件探测决定默认值。

**Step 3: 拆分预设构造入口**

保留兼容 `DefaultLifecyclePresets()` 与 `DefaultLifecyclePresetChains()` 的原行为，或等价地提供明确的旧数据兼容构造函数。增加新安装预设构造入口，返回原“默认预设”和“公司框架”，并让 `Default(...)` 将默认 ID 指向 `CompanyFrameworkLifecyclePresetID`。

**Step 4: 运行设置测试确认通过**

Run: `go test ./internal/settings -run 'TestDefault(IncludesLifecyclePreset|SeedsDefaultBranchTemplateAndRepositoryPresetChains|LifecycleConfigurationValidates)'`

Expected: PASS。

### Task 4: 锁定首次持久化与旧数据兼容

**Files:**
- Modify: `internal/storage/repository_test.go`
- Modify if required: `internal/storage/repository.go`
- Modify if required: `internal/settings/settings.go`

**Step 1: 添加首次持久化失败测试**

用不存在的数据文件创建仓库并加载，断言保存后的设置包含两个预设、公司框架默认 ID 和名称为“更新框架仓库”的稳定更新链。重新创建仓库实例加载同一文件，断言结果保持一致。

**Step 2: 添加当前版本已有数据回归测试**

写入带 `presetVersion: 6` 的设置快照，其中只含用户自己的预设、用户默认选择，以及名称为“更新仓库”或自定义名称的稳定更新链。加载后断言不新增公司框架、不切换默认 ID、不修改链名称。

**Step 3: 扩展历史缺字段回归测试**

沿用 `TestRepositorySeedsRepositoryPresetChainsOnlyOnce` 和生命周期预设迁移测试，明确断言历史数据仍得到“更新仓库”和原“默认预设”，删除后的预置链仍不会重建。

**Step 4: 运行测试确认至少一个新增场景失败**

Run: `go test ./internal/storage -run 'TestRepository(SeedsCompanyFrameworkForNewInstall|PreservesExistingLifecycleDefaults|SeedsRepositoryPresetChainsOnlyOnce|MigratesLegacyLifecycleDefaultsToPreset)'`

Expected: 在实现兼容分界前 FAIL，失败信息指出新安装或已有设置行为不符合断言。

**Step 5: 实现最小初始化边界调整**

仅在必要时修改仓库初始化，让不存在数据文件的路径使用新安装种子；已有文件始终由保存内容和既有迁移规则决定。不得提升 `CurrentPresetVersion`，不得新增公司框架迁移。

**Step 6: 运行存储测试确认通过**

重复 Step 4 命令。

Expected: PASS。

### Task 5: 验证任务保存展开后的默认映射

**Files:**
- Modify: `internal/lifecycle/service_test.go`
- Modify if existing coverage is better: `app_test.go`

**Step 1: 添加失败测试**

使用新安装设置创建仓库和生命周期服务，通过不显式提交链选择的兼容创建入口创建任务，断言：

```go
map[task.LifecycleHook]string{
    task.LifecycleHookBeforeStart: settings.LifecycleChainIterationsAIID,
    task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
    task.LifecycleHookUpdateTask:  settings.LifecycleChainUpdateRepositoriesID,
}
```

任务类型中不得增加预设 ID 字段。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/lifecycle -run TestServiceCreatesTaskFromCompanyFrameworkDefaultPreset`

Expected: 若默认解析仍指向原预设则 FAIL。

**Step 3: 完成最小实现并确认通过**

默认预设解析应继续复用 `Settings.DefaultLifecyclePresetChains()`，无需为公司框架增加分支。

Run: `go test ./internal/lifecycle -run TestServiceCreatesTaskFromCompanyFrameworkDefaultPreset`

Expected: PASS。

### Task 6: 添加前端数据驱动回归测试

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify only if required: `frontend/src/App.tsx`

**Step 1: 添加设置页测试**

让绑定返回包含两条预设和四条相关命令链的设置数据。打开设置的生命周期区域，断言同时显示“默认预设”“公司框架”，公司框架行显示“默认”，命令链列表显示“更新框架仓库”。

**Step 2: 添加新建任务表单测试**

打开新建任务表单，断言“命令链预设”为“公司框架”，五个钩子分别显示 `iterations-ai`、不执行、 不执行、“删除任务工作目录”、“更新框架仓库”。提交时断言现有 `CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains` 或对应绑定收到三个稳定链 ID，不含预设 ID。

**Step 3: 运行测试**

Run: `cd frontend && npm test -- --run src/App.test.tsx`

Expected: PASS；如果现有界面不能仅靠数据满足断言，测试先 FAIL，再只对 `App.tsx` 做最小修复后重新运行。

### Task 7: 定向与完整自动验证

**Files:**
- Verify: `internal/settings/settings.go`
- Verify: `internal/storage/repository.go`
- Verify: `internal/lifecycle/service.go`
- Verify: `frontend/src/App.tsx`

**Step 1: 运行 Go 定向测试**

Run: `go test -race ./internal/settings ./internal/storage ./internal/lifecycle`

Expected: PASS。

**Step 2: 运行根包相关测试**

Run: `go test -race . -run 'Test(AppExposesLifecyclePresetBindings|LifecyclePresetChainsResolveWithConfiguredParameters)'`

Expected: PASS；根据最终测试名称补充公司框架相关根包测试筛选。

**Step 3: 运行前端验证**

Run: `cd frontend && npm test && npm run build`

Expected: Vitest 全部 PASS，TypeScript 与 Vite 构建成功。

**Step 4: 运行完整 Go 验证**

Run: `go test -race ./...`

Expected: PASS；只修复本变更引起的失败。

### Task 8: 同步文档与校验 OpenSpec

**Files:**
- Modify: `README.md`
- Modify: `openspec/changes/add-company-framework-lifecycle-preset/tasks.md`
- Verify: `openspec/changes/add-company-framework-lifecycle-preset/proposal.md`
- Verify: `openspec/changes/add-company-framework-lifecycle-preset/design.md`
- Verify: `openspec/changes/add-company-framework-lifecycle-preset/specs/task-lifecycle-command-chains/spec.md`
- Verify: `openspec/changes/add-company-framework-lifecycle-preset/specs/default-branch-lifecycle-presets/spec.md`

**Step 1: 更新用户文档**

在 `README.md` 说明新安装含两个预设、公司框架默认映射、更新框架仓库的执行语义，以及已有设置不会被自动更改。

**Step 2: 校验 OpenSpec**

Run: `openspec validate add-company-framework-lifecycle-preset --strict`

Expected: `Change 'add-company-framework-lifecycle-preset' is valid`。

**Step 3: 更新任务状态**

仅将已经由测试或检查验证的任务勾选完成，保持尚未执行的集成、确认、合并和归档步骤未完成。

### Task 9: 执行 Wails 新用户集成测试

**Files:**
- Runtime data: 临时目录中的 `taskai/tasks.json`

**Step 1: 创建隔离配置目录**

Run: `integration_config_dir="$(mktemp -d)" && test ! -e "$integration_config_dir/taskai/tasks.json"`

Expected: 命令成功，目录内没有 TaskAI 持久化数据。不得复用 `$HOME`、`~` 或用户真实配置目录。

**Step 2: 持续启动开发应用**

Run: `XDG_CONFIG_HOME="$integration_config_dir" wails dev`

Expected: 命令保持运行，终端保留颜色输出并显示前端调试地址；不要使用 `NO_COLOR`，不要让应用自动退出。

**Step 3: 使用 Chrome DevTools 验证设置**

打开调试地址，进入设置的“生命周期编排”，确认：

- 预设列表包含“默认预设”和“公司框架”；
- “公司框架”显示“默认”；
- 命令链列表显示“更新框架仓库”。

**Step 4: 验证新建任务默认选择**

打开新建任务表单，确认开始前为 `iterations-ai`、开始后为空、结束前为空、结束后为“删除任务工作目录”、任务更新为“更新框架仓库”。

**Step 5: 验证持久化但不执行生命周期命令**

填写标题与当前模板要求的“默认分支”，创建未执行任务，再打开编辑确认相同选择。不得点击开始或结束，避免实际克隆公司仓库或删除工作目录。

**Step 6: 关闭开发应用并保留证据**

记录调试地址、浏览器断言结果和持久化验证结果，然后正常终止 `wails dev`。临时目录可在确认无需复查后删除。

### Task 10: 编译、打开并等待确认

**Files:**
- Build output: `build/bin/taskai`

**Step 1: 编译 Linux 可执行程序**

Run: `./scripts/build-linux.sh`

Expected: 输出 `Linux 构建完成: .../build/bin/taskai` 且产物可执行。

**Step 2: 打开程序且保持运行**

启动 `build/bin/taskai`，不得通过自动退出、超时或禁色环境变量运行；确认窗口正常打开。

**Step 3: 等待用户确认**

报告自动测试、浏览器集成测试、构建和启动结果，保持程序运行，等待用户明确确认后才进入分支合并。

### Task 11: 合并、复验、归档与提交

**Files:**
- Modify during archive: `openspec/specs/task-lifecycle-command-chains/spec.md`
- Modify during archive: `openspec/specs/default-branch-lifecycle-presets/spec.md`
- Archive: `openspec/changes/add-company-framework-lifecycle-preset/`

**Step 1: 合并当前工作区分支进功能分支**

在用户确认后，把当前工作区项目对应分支合并到 `feat/add-company-framework-lifecycle-preset`；若有冲突，逐项保留双方有效变更并重新运行受影响测试。

**Step 2: 合并功能分支回当前工作区分支**

确认功能分支干净且测试通过后，在当前工作区项目对应分支执行非破坏性合并。

**Step 3: 验证合并结果**

Run: `go test -race ./...`

Run: `cd frontend && npm test && npm run build`

Run: `./scripts/build-linux.sh`

Expected: 全部成功。

**Step 4: 归档 OpenSpec**

使用 `openspec-archive-change` 流程归档 `add-company-framework-lifecycle-preset`，确认正式规格包含最终需求，归档后的严格校验通过。

**Step 5: 提交 Git 变更**

检查 `git diff` 和 `git status`，只暂存本变更文件，创建描述公司框架默认预设的提交；不得混入用户无关修改。

**Step 6: 移除 worktree**

确认功能分支已合并、提交已存在且 worktree 无未提交内容后，移除 `.worktrees/add-company-framework-lifecycle-preset`，再检查 `git worktree list` 和最终工作树状态。
