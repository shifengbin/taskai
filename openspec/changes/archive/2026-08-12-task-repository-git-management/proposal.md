## Why

任务工作目录常包含一个或多个 Git 仓库，但用户需要切换到终端才能查看状态、提交改动和同步远程分支。任务详情提供受控的仓库 Git 操作，可在不开放任意命令执行的前提下完成这类高频操作。

## What Changes

- 在选中任务且未选中终端时的右侧任务详情新增“Git”标签页。
- 扫描任务工作目录内的普通 Git 仓库和 Git worktree，并为每个仓库独立展示分支、远程、改动与同步状态。
- 在设置中提供 Git 最大扫描深度，默认扫描到第 2 层目录，限制大目录扫描耗时。
- 为每个仓库提供一个随状态切换的主按钮：有未提交改动时提交；远程不存在当前分支时发布分支；其他可同步情形下同步远程。
- 提供提交信息输入；提交固定执行暂存全部改动后创建提交。
- 后端新增受任务工作目录边界保护的专用 Git 查询与操作接口，不向前端开放任意 Git 参数或任意目录。

## Capabilities

### New Capabilities
- `task-repository-git-management`: 在任务详情中发现任务工作目录内的 Git 仓库，并以受控的提交、发布和同步操作管理每个仓库。

### Modified Capabilities
- `task-detail-command-context`: 任务详情从单页信息展示扩展为包含 Git 仓库管理的标签页。

## Impact

- 前端：`frontend/src/App.tsx` 的任务详情、`api.ts`、`types.ts` 及关联测试。
- 后端：`app.go` 的 Wails 导出方法、`internal/application/contracts.go` 的绑定契约、设置持久化，以及新的 Git 仓库发现和操作服务及测试。
- Wails 绑定：新增导出类型和方法后需要重新生成 `frontend/wailsjs/`。
