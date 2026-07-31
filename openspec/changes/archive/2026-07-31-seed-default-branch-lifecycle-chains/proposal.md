## Why

新安装和既有用户需要一套可直接选择的分支模板及仓库初始化链，避免每次手工重建相同的任务字段和生命周期步骤。预置 `iterations-ai` 应在任务进入执行中前完成初始化；同时 Git 远程分支检查不能依赖桌面应用可能失效的启动目录。

## What Changes

- 新增并默认激活“默认分支”任务模板：包含必填字符串字段 `branch`，显示名称为“默认分支”，默认值为空。
- 预置可删除的 `beforeStart` 命令链“iterations-ai”：依次创建任务工作目录、克隆 `git@gitlab.jiandan100.cn:webdev/iterations-ai.git`、生成清单文件、以 `dir=workspaces` 克隆任务选定的 Git 仓库。
- 预置可删除的 `updateTask` 命令链“更新仓库”：依次生成清单文件、以 `dir=workspaces` 克隆任务选定的 Git 仓库。
- 保留“创建任务工作目录”和“Git 仓库克隆”系统命令对 `beforeStart`、`postStart` 的支持，供用户配置其他命令链。
- 为已有设置增加一次性预置迁移；版本 2 只将结构未修改的旧 `iterations-ai` 从 `postStart` 调整为 `beforeStart`，版本 3 会补回版本 2 中异常缺失的默认分支模板。迁移完成后，用户删除预置模板或链不会在后续启动时重新出现。
- Git 远程检查和克隆使用克隆目标的父目录作为显式工作目录；开始前链的最终事件不得被旧的开始请求响应覆盖。
- 两条预置链不写入任何生命周期默认链映射，因此新建任务不会自动选择它们。

## Capabilities

### New Capabilities

- `default-branch-lifecycle-presets`: 定义默认分支模板、可删除的仓库命令链预置、一次性迁移及其钩子范围规则。

### Modified Capabilities

- 无。

## Impact

- `internal/settings` 的默认设置、生命周期命令范围、预置版本标记与归一化逻辑。
- `internal/storage` 的设置迁移与预置补齐逻辑；需与进行中的生命周期配置持久化修复保持字段所有权一致。
- 新建任务的默认模板选择、生命周期命令链管理、README 和 Go/前端测试。
