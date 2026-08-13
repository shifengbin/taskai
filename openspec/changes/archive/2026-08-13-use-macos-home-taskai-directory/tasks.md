## 1. 隔离开发与建立测试基线

- [x] 1.1 在 `taskai/.worktrees/` 创建 `use-macos-home-taskai-directory` 功能分支及专用 Git worktree，读取适用的 `AGENTS.md`，并确认主工作区已有变更不会被覆盖。
- [x] 1.2 在 worktree 中运行 `go test ./...` 建立 Go 测试基线，并记录当前 macOS 默认数据目录和 `settings.Default()` 工作区派生行为。

## 2. 以测试定义 macOS 路径与迁移

- [x] 2.1 先增加失败测试，覆盖 macOS 新安装解析 `~/.taskai`、Linux/Windows 继续使用各自用户配置目录，以及 Home 目录不可用时的既有回退行为。
- [x] 2.2 先增加失败测试，覆盖旧配置成功迁移、旧默认工作区更新为 `~/.taskai/workspaces`、自定义工作区保留、已有任务工作目录快照不改写。
- [x] 2.3 先增加失败测试，覆盖新旧目录同时存在时只使用新目录，以及复制、保存或目录发布失败时清理临时目录、保留旧配置并回退使用旧目录。
- [x] 2.4 运行新增的精确 Go 测试，确认它们因尚未实现 macOS 路径解析和迁移而失败，且失败原因与规格一致。

## 3. 实现平台路径与安全迁移

- [x] 3.1 将应用数据目录解析提取为可测试的平台路径组件：macOS 使用用户 Home 下的 `.taskai`，Linux 和 Windows 保持 `os.UserConfigDir()/taskai` 的现有行为。
- [x] 3.2 在存储仓库创建前实现 macOS 旧配置迁移，使用 Home 同级临时目录完成复制、加载、必要的默认工作区更新和原子目录发布，任一步骤失败都清理临时内容并回退旧目录。
- [x] 3.3 仅在旧设置值精确等于旧默认工作区时改为 `~/.taskai/workspaces`；保留自定义设置和所有任务的 `WorkspaceRoot`、`WorkspacePath` 快照，并且不移动或删除旧工作区。
- [x] 3.4 运行新增路径与迁移测试及 `go test ./...`，确认所有新场景通过且存储、设置、生命周期和工作目录安全测试没有回归。

## 4. 同步工作区并自动集成测试

- [x] 4.1 将当前工作区项目分支合并到 worktree 功能分支，解决冲突后重新运行相关 Go 测试。
- [x] 4.2 在 macOS 测试 Home 中预置仅含旧 `~/Library/Application Support/taskai/tasks.json` 的数据：包含一项使用旧默认工作区的设置、一项已有任务工作目录快照和可识别任务；另准备使用自定义工作区及新旧目录同时存在的独立场景。
- [ ] 4.3 在 macOS 上不设置 `NO_COLOR` 等禁用终端颜色的环境变量，以持续运行方式执行 `wails dev`，从输出读取调试地址，并使用 Chrome DevTools 浏览器工具打开该地址。
- [ ] 4.4 通过浏览器进入“设置 → 工作区与外观”，确认首次启动迁移后“新任务工作区根目录”为 `~/.taskai/workspaces`；确认旧任务仍显示原工作目录快照，创建并开始新任务后确认其目录位于 `~/.taskai/workspaces/<task-id>`。
- [ ] 4.5 关闭 `wails dev` 后检查文件系统：`~/.taskai/tasks.json` 存在、旧配置和旧工作区内容未被改动、没有残留迁移临时目录；分别复验自定义工作区保持不变以及新旧目录冲突时新目录优先且双方数据未被合并或删除。

## 5. 构建、确认与交付

- [x] 5.1 运行 `go test -race ./...`、`cd frontend && npm test && npm run build` 以及 `openspec validate use-macos-home-taskai-directory --strict`，修复本变更引入的所有失败后重新验证。
- [ ] 5.2 在 macOS 上执行 `./scripts/build-macos.sh` 编译应用，打开生成的 TaskAI `.app` 且保持程序运行，复查设置中的默认工作区路径并等待用户确认。
- [x] 5.3 用户确认后，将 worktree 功能分支合并回当前工作区项目分支；若出现冲突则解决冲突，并再次运行相关测试和当前主机可用的构建脚本验证合并结果。
- [x] 5.4 同步中文设计与实施文档，将完成后的 OpenSpec 规格合入主规格并归档 `use-macos-home-taskai-directory` 变更。
- [ ] 5.5 提交全部 Git 变更，确认功能分支已经合并后移除 `.worktrees/` 下的专用 worktree。

> 归档说明（2026-08-13）：实现和自动化测试在 Linux 主机完成，并已通过用户确认继续交付。任务 4.3–4.5 与 5.2 需要 macOS 原生主机及 `.app`，当前环境无法执行，因此保持未勾选；合并后改用 `./scripts/build-linux.sh` 完成当前主机生产构建验证。
