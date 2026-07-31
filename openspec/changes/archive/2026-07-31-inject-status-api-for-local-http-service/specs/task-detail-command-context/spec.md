## MODIFIED Requirements

### Requirement: 任务详情仅展示系统环境变量

系统 MUST 在任务详情中提供“系统环境变量”说明区，并且只展示应用稳定注入的 `TASKAI_TASK_ID`、`TASKAI_TERMINAL_ID` 及条件性注入的 `TASKAI_STATUS_API`。说明区 MUST 标明每个变量的适用范围：任务 ID 用于任务关联的自定义命令和脚本，终端 ID 用于新建终端，状态 API 在本机 HTTP 服务正在监听时注入到之后新建的终端。`TASKAI_STATUS_API` 的值 MUST 等于当前状态 API 基址。系统 MUST 不在该区展示模板字段派生的环境变量或 Shell 运行器内部变量。

#### Scenario: 本机 HTTP 服务未监听时展示变量范围
- **WHEN** 用户查看任意任务详情且本机 HTTP 服务未监听
- **THEN** 系统展示 `TASKAI_TASK_ID` 和 `TASKAI_TERMINAL_ID` 的适用范围，并说明 `TASKAI_STATUS_API` 仅在本机 HTTP 服务正在监听时注入到之后新建的终端

#### Scenario: 独立 HTTP 服务为新终端注入状态 API
- **WHEN** 用户保持标题变化状态管理、启用独立本机 HTTP 服务并在服务监听后新建终端
- **THEN** 该终端环境包含值等于当前状态 API 基址的 `TASKAI_STATUS_API`

#### Scenario: HTTP 状态管理为新终端注入状态 API
- **WHEN** 用户选择 HTTP 状态管理方式并在服务监听后新建终端
- **THEN** 该终端环境包含值等于当前状态 API 基址的 `TASKAI_STATUS_API`

#### Scenario: 设置变更不修改已有终端环境
- **WHEN** 用户在终端已创建后启用、关闭或重新配置本机 HTTP 服务
- **THEN** 已有终端的进程环境保持不变，之后新建的终端根据当时的 HTTP 服务监听状态获得环境变量

#### Scenario: 不展示模板派生变量
- **WHEN** 当前任务模板将字段 `environment` 标记为生命周期环境变量注入
- **THEN** 任务详情的系统环境变量区不展示 `TASKAI_ENVIRONMENT`
