# Windows Git 仓库发现修复实施记录

## 目标

- Windows 图形应用打开任务 Git 标签或执行仓库操作时，TaskAI 直接启动的 Git 子进程不显示控制台窗口。
- 默认扫描范围可以发现任务工作目录下的 `workspaces/<项目>` 仓库。
- 普通目录不再触发 Git 校验；单个损坏或不可读取仓库不遮蔽其他有效仓库。

## 实施结果

1. 将 Windows 后台进程配置提取到 `internal/backgroundprocess`，统一设置 `HideWindow` 与 `CREATE_NO_WINDOW`。
2. 生命周期命令、任务菜单后台命令和任务 Git 仓库服务统一复用该配置；内嵌终端不经过该组件。
3. 仓库扫描只将带 `.git` 目录或文件的父目录作为候选，再执行 Git 顶层目录与真实路径边界验证。
4. 有效仓库基础状态读取失败时返回带路径和错误提示的不可操作卡片，并继续聚合其他仓库。
5. 默认 Git 扫描深度由 2 调整为 3，通过预置版本 6 一次性迁移旧默认值；其他深度保持不变，迁移后仍可显式改回 2。
6. 前端缺失值回退和设置提示同步为默认深度 3，Wails 传输结构未改变。

## 验证记录

- `go test -race ./...` 通过。
- 前端 16 个测试文件、311 项测试通过，生产构建通过。
- Windows amd64 Wails 可执行文件构建通过，Windows 专用测试二进制交叉编译通过。
- `wails dev -tags webkit2_41` 在 Linux 启动成功，通过浏览器交互确认任务 Git 标签展示 `taskai / main / origin`。
- `scripts/build-linux.test.sh` 通过；`scripts/build-linux.sh amd64` 构建正式 Linux 产物成功。
- 正式 Linux 产物在隔离 D-Bus 会话与配置副本中启动，实际选择任务并切换 Git 标签后正常展示仓库、分支、远端和未提交状态。
- OpenSpec 严格校验与 `git diff --check` 通过。

## 待验事项

- Windows 黑框属于真实桌面窗口行为，Linux 交叉构建和 Wine 无法替代 Windows 实机确认。Windows 实机点击 Git 标签及执行受控 Git 操作的回归由用户在其他 Windows 系统完成。
