# git-clone-lifecycle-command Specification

## Purpose
TBD - created by archiving change make-git-clone-directory-optional. Update Purpose after archive.
## Requirements
### Requirement: Git 克隆命令支持默认任务目录
系统 SHALL 允许 Git 仓库克隆命令链引用不包含任何非空追加参数。系统 MUST 将该情形的克隆根目录解析为任务工作目录，并将每个被选中的内置 Git 信息克隆到 `<任务工作目录>/<项目名称>`。

#### Scenario: 未填写参数时克隆到任务目录
- **WHEN** 用户将 Git 仓库克隆命令加入命令链且未填写任何追加参数
- **THEN** 系统将每个内置 Git 项目克隆到任务工作目录下以项目名称命名的子目录

#### Scenario: 仅填写空白行时使用默认目录
- **WHEN** Git 仓库克隆命令的追加参数仅包含空白行
- **THEN** 系统将空白行规范化为无参数，并使用任务工作目录作为克隆根目录

#### Scenario: 显式当前目录保持兼容
- **WHEN** 已保存的 Git 仓库克隆命令引用使用 `dir=.`
- **THEN** 系统继续将每个项目克隆到任务工作目录下以项目名称命名的子目录

### Requirement: Git 克隆命令支持安全的自定义目录
系统 SHALL 允许 Git 仓库克隆命令以唯一的 `dir=<相对目录>` 参数指定克隆根目录。系统 MUST 将每个项目克隆到 `<任务工作目录>/<dir>/<项目名称>`，并保持现有目标存在时跳过的行为。

#### Scenario: 配置自定义子目录
- **WHEN** 用户为 Git 仓库克隆命令填写 `dir=repositories`
- **THEN** 系统将每个内置 Git 项目克隆到 `<任务工作目录>/repositories/<项目名称>`

#### Scenario: 拒绝不安全或无效的显式目录
- **WHEN** Git 仓库克隆命令包含多个参数、非 `dir` 参数、空目录、绝对路径或会跳出任务工作目录的相对路径
- **THEN** 系统拒绝保存或执行该命令链配置，并说明 Git 克隆目录参数无效

### Requirement: Git 克隆目录行为可被理解
系统 MUST 在生命周期编排界面和项目 README 中说明 Git 仓库克隆命令的追加参数可留空，以及默认目录和 `dir=<相对目录>` 的最终克隆目标。

#### Scenario: 查看 Git 克隆命令说明
- **WHEN** 用户在生命周期编排界面查看 Git 仓库克隆命令或查阅项目 README
- **THEN** 用户可以得知留空时使用任务工作目录，填写 `dir=<相对目录>` 时使用任务工作目录下的指定子目录
