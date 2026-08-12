## ADDED Requirements

### Requirement: macOS 使用 Home 下的 TaskAI 配置目录
系统 MUST 在 macOS 上将当前用户 Home 目录下的 `.taskai` 作为默认应用配置目录，并将应用持久化文件保存为 `~/.taskai/tasks.json`。当没有可迁移的旧配置时，系统 MUST 将新任务工作区根目录默认设置为 `~/.taskai/workspaces`。

#### Scenario: macOS 首次启动
- **WHEN** macOS 用户的 `~/.taskai` 与旧配置文件均不存在并启动应用
- **THEN** 系统在 `~/.taskai/tasks.json` 保存应用配置和任务数据，并在设置中显示 `~/.taskai/workspaces` 作为新任务工作区根目录

#### Scenario: 新任务使用新的默认工作区
- **WHEN** macOS 用户未修改默认工作区设置并开始一个新任务
- **THEN** 系统将该任务的工作目录快照设置为 `~/.taskai/workspaces/<task-id>`

### Requirement: macOS 安全迁移旧配置
当 `~/.taskai` 不存在且 `~/Library/Application Support/taskai/tasks.json` 存在时，系统 MUST 在创建存储仓库前将旧配置和任务数据迁移到 `~/.taskai/tasks.json`。迁移 MUST 保留任务、设置和其他持久化数据；系统 MUST NOT 移动、删除或改写旧任务工作目录以及任务自身保存的工作目录快照。

#### Scenario: 迁移使用旧默认工作区的配置
- **WHEN** macOS 旧配置存在、新配置目录不存在，且旧设置的工作区根目录等于 `~/Library/Application Support/taskai/workspaces`
- **THEN** 系统将配置和任务数据迁移到 `~/.taskai/tasks.json`，将设置中的新任务工作区根目录更新为 `~/.taskai/workspaces`，并保持已有任务的工作目录快照及旧工作区内容不变

#### Scenario: 保留用户自定义工作区
- **WHEN** macOS 旧配置存在、新配置目录不存在，且旧设置包含用户自定义的工作区根目录
- **THEN** 系统将配置和任务数据迁移到 `~/.taskai/tasks.json`，并保持该自定义工作区根目录不变

#### Scenario: 迁移失败
- **WHEN** 系统无法完整创建或写入新的 macOS 配置目录
- **THEN** 系统不得留下可被误判为有效新配置的部分迁移目录，不得修改旧配置和旧工作区，并继续使用旧配置启动应用

### Requirement: 配置目录冲突时保护已有数据
当 macOS 新旧配置目录同时存在时，系统 MUST 优先使用 `~/.taskai`，且 MUST NOT 自动合并、覆盖或删除任一目录中的数据。

#### Scenario: 新旧配置同时存在
- **WHEN** macOS 用户启动应用且 `~/.taskai` 与 `~/Library/Application Support/taskai` 同时存在
- **THEN** 系统只从 `~/.taskai/tasks.json` 加载当前配置，不修改旧配置目录及其工作区内容

### Requirement: 其他平台保持现有默认目录
系统 MUST 将 Home 下 `.taskai` 的新默认值和迁移逻辑限制在 macOS，不得改变 Linux 和 Windows 的配置目录解析及默认新任务工作区行为。

#### Scenario: Linux 或 Windows 启动
- **WHEN** 用户在 Linux 或 Windows 上启动应用
- **THEN** 系统继续使用该平台现有的用户配置目录及其 `taskai/workspaces` 子目录作为默认值，并且不执行 macOS 旧目录迁移
