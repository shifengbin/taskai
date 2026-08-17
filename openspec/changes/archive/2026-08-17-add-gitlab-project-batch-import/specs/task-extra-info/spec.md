## ADDED Requirements

### Requirement: 使用完整仓库路径辅助区分 Git 信息
系统 MUST 保持 Git 信息 `name` 固定字段及其项目名称语义不变，并 MUST 在额外信息管理列表和任务选择器候选项中为可解析的 Git 仓库地址辅助显示 `namespace/project` 完整路径。辅助路径 MUST 仅由现有 `repository` 字段推导，不得增加持久化字段、修改任务快照或改变 Git 克隆生命周期命令使用项目名称创建目标目录的行为。无法从仓库地址提取完整路径时，系统 MUST 继续正常显示项目名称。

#### Scenario: 区分不同群组中的同名项目
- **WHEN** Git 分类同时包含项目名均为 `api`、仓库路径分别为 `team-a/api` 和 `team-b/api` 的两条信息
- **THEN** 额外信息管理列表和任务选择器候选项均以 `api` 为主名称，并分别辅助显示对应完整路径

#### Scenario: 项目名称继续决定克隆目录
- **WHEN** 用户选择一条由 GitLab 导入、名称为 `api` 且完整路径为 `team-a/api` 的 Git 信息并执行 Git 仓库克隆生命周期命令
- **THEN** 系统继续克隆到以 `api` 命名的子目录，不把 `team-a/api` 用作目标目录

#### Scenario: 无法提取完整路径时保持兼容
- **WHEN** 既有 Git 信息包含系统无法解析的仓库地址
- **THEN** 额外信息管理列表和任务选择器继续显示项目名称，且该信息仍可正常选择和使用
