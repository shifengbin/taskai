# terminal-output-retention Specification

## Purpose
TBD - created by archiving change improve-terminal-switch-performance. Update Purpose after archive.
## Requirements
### Requirement: 活动终端的输出状态有界保留

系统 SHALL 为每个活动终端维护独立的前端 xterm 会话，并将该会话的滚屏上限设置为 1000 行。终端输出事件 MUST 以增量方式写入对应会话；系统 MUST NOT 在 React 终端元数据中无上限累积原始输出文本。

#### Scenario: 后台终端产生超过 1000 行输出
- **WHEN** 未被选中的活动终端产生超过 1000 行的输出
- **THEN** 该终端会话只保留最近 1000 行滚屏及当前可见行，且之前的输出不会使 React 状态或下一次终端切换的处理量持续增长

#### Scenario: 含 ANSI 控制序列的后台输出首次显示
- **WHEN** 活动终端在未显示期间收到带 ANSI 样式、光标控制或备用屏幕状态的输出，随后被用户选择
- **THEN** 系统显示该会话已解析的 xterm 状态，而不是截断并回放原始文本

### Requirement: 终端会话按生命周期释放

系统 MUST 在终端关闭或退出、所属任务结束以及前端应用卸载时释放对应终端会话、xterm 附加组件和事件监听器。已释放会话 MUST NOT 接受晚到的输出或尺寸更新。

#### Scenario: 关闭单个终端
- **WHEN** 用户关闭一个活动终端或该终端进程退出
- **THEN** 系统释放该终端的会话，并且之后同一终端 ID 的晚到输出不会重新创建或更新该会话

#### Scenario: 结束任务
- **WHEN** 用户结束一个包含多个终端的任务
- **THEN** 系统释放该任务的所有终端会话，而其他任务的终端会话保持可用

#### Scenario: 前端卸载
- **WHEN** 前端应用卸载
- **THEN** 系统释放所有仍被注册的终端会话及其监听器
