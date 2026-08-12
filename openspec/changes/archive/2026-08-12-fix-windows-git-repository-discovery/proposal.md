## Why

Windows 用户打开任务详情的 Git 标签时，仓库扫描会为大量目录直接启动可见的 Git 控制台进程，连续弹出黑框；同时默认扫描深度无法覆盖默认生命周期创建的 `workspaces/<项目>` 仓库布局，导致扫描结束后仍可能显示“未发现 Git 仓库”。这两个问题使任务 Git 管理在默认配置下不可用，需要统一修复仓库发现的进程行为、扫描范围和效率。

## What Changes

- Windows 上由任务 Git 仓库服务直接启动的查询、提交、发布和同步 Git 子进程使用后台无控制台窗口策略，同时保留标准输出、标准错误和退出状态。
- 仓库发现先依据目录中的 `.git` 目录或文件筛选普通仓库与 Git worktree 候选，再对候选执行 Git 验证，避免对扫描范围内的每个普通目录启动 Git 子进程。
- 将 Git 最大扫描深度的默认值调整为能够发现默认生命周期放在 `workspaces/<项目>` 中的仓库，并通过一次性版本迁移更新旧设置；迁移后用户仍可显式改回更小深度。
- 扫描范围内单个无效候选或暂时不可读取的仓库不得阻止其他有效仓库展示。
- 增加 Windows 进程属性、默认目录布局、候选筛选、部分仓库失败和设置迁移的自动化验证。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `task-repository-git-management`: 调整仓库发现的默认扫描范围、候选识别与部分失败行为，确保默认生命周期创建的仓库可见且单个异常仓库不影响其他仓库。
- `windows-lifecycle-command-execution`: 将 Windows 无控制台窗口要求扩展到任务 Git 仓库服务直接启动的后台 Git 子进程。

## Impact

- 后端：`internal/repositorygit` 的仓库发现、Git 进程创建和状态聚合逻辑。
- 设置：`internal/settings` 与 `internal/storage` 中 Git 最大扫描深度的默认值、旧默认值迁移和持久化测试。
- 规格与文档：任务 Git 仓库管理及 Windows 后台进程行为。
- 前端与 Wails 接口结构不变；现有 Git 标签继续使用当前加载、空态和错误展示接口。
