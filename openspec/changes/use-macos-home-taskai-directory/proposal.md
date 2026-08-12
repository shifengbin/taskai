## Why

TaskAI 当前在 macOS 上跟随 `os.UserConfigDir()`，导致配置和任务工作区默认位于 `~/Library/Application Support/taskai`。将两者统一放到 Home 目录下的 `~/.taskai` 与 `~/.taskai/workspaces`，可以让路径更容易查找、备份和供命令行工具使用，同时需要保护升级用户已有的任务、设置和工作区配置。

## What Changes

- macOS 上的应用配置目录默认改为 `~/.taskai`，持久化文件位于 `~/.taskai/tasks.json`。
- macOS 新安装的“新任务工作区根目录”默认改为 `~/.taskai/workspaces`。
- macOS 升级时，在新目录尚不存在且旧配置存在的情况下，将 `~/Library/Application Support/taskai` 安全迁移到 `~/.taskai`，保留任务、设置以及用户自定义的工作区根目录。
- 新旧配置目录同时存在时优先使用新目录，不合并、不覆盖或删除任一目录，避免不确定的数据覆盖。
- Linux 和 Windows 的配置目录及默认工作区路径保持现有行为；已开始任务继续使用自身保存的工作目录快照。

## Capabilities

### New Capabilities

- `macos-application-data-directory`: 规定 macOS 配置目录、默认新任务工作区路径、旧数据迁移和目录冲突处理行为。

### Modified Capabilities

无。

## Impact

- 影响应用初始化时的数据目录解析、macOS 平台路径处理、存储仓库初始化和相关测试。
- 不改变 Wails 导出 API、`Settings` 数据结构或前端设置表单；前端将通过现有设置接口显示新的默认工作区路径。
- 迁移操作只涉及 TaskAI 明确管理的旧配置目录和新配置目录，不迁移用户在设置中另行指定的外部工作区，也不修改已有任务保存的工作目录快照。
