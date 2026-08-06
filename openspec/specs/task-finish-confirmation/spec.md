# task-finish-confirmation Specification

## Purpose
TBD - created by archiving change correct-finish-task-dialog-copy. Update Purpose after archive.

## Requirements
### Requirement: 结束任务确认弹窗准确说明清理职责
系统 MUST 在用户确认结束执行中任务前显示确认弹窗。弹窗 MUST 明确说明确认将结束目标任务并关闭其全部终端，且结束后命令链将按该任务保存的配置执行。弹窗 MUST NOT 声称结束任务会删除工作目录或其内容；工作目录的删除仅能由已配置结束后命令链中的删除目录命令完成。

#### Scenario: 任务配置结束后删除目录命令
- **WHEN** 用户对配置了包含删除目录命令的 `postEnd` 命令链的执行中任务发起结束操作
- **THEN** 确认弹窗说明结束任务、关闭终端和按配置执行结束后命令链，但不将工作目录删除表述为结束任务的固定结果

#### Scenario: 任务未配置目录删除命令
- **WHEN** 用户对未配置删除目录命令的执行中任务发起结束操作
- **THEN** 确认弹窗不得显示会删除该任务工作目录或其内容的文字

### Requirement: 确认操作命名为结束任务
结束任务确认弹窗的确认按钮 MUST 显示“结束任务”，并继续触发现有的结束任务请求。取消按钮 MUST 保持不触发结束任务请求。

#### Scenario: 取消结束任务
- **WHEN** 用户打开结束任务确认弹窗后选择取消
- **THEN** 系统关闭确认弹窗且不得调用结束任务接口

#### Scenario: 确认结束任务
- **WHEN** 用户在结束任务确认弹窗中选择“结束任务”
- **THEN** 系统调用现有结束任务接口，并沿用既有的终端关闭、任务状态转换和生命周期命令链处理
