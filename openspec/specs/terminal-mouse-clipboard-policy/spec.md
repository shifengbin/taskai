# terminal-mouse-clipboard-policy Specification

## Purpose
定义用户为单个显示终端的自定义菜单命令关闭 TaskAI 鼠标剪贴板处理的行为，使终端程序能够自行处理选区和右键操作，同时保持其他终端的既有复制、粘贴体验。

## Requirements

### Requirement: 菜单终端命令可配置鼠标剪贴板策略

系统 SHALL 为每个用户创建且显示终端的任务菜单命令保存 `disableTaskAIMouseClipboard` 设置。新建命令和缺失该字段的既有配置 MUST 默认为 `false`。非终端命令与系统固定菜单项 MUST NOT 启用该设置。

#### Scenario: 用户为 Claude 菜单命令开启设置

- **WHEN** 用户在菜单管理中编辑一个显示终端的自定义命令，并勾选“禁用 TaskAI 鼠标复制与右键粘贴”后保存设置
- **THEN** 系统持久化该命令的 `disableTaskAIMouseClipboard: true`

#### Scenario: 用户关闭显示终端

- **WHEN** 用户关闭某个自定义命令的“显示终端”设置
- **THEN** 系统隐藏鼠标剪贴板设置，并将其保存值归一化为 `false`

#### Scenario: 读取旧配置或系统固定菜单项

- **WHEN** 系统读取没有该字段的既有菜单配置，或归一化一个系统固定菜单项
- **THEN** 该设置为 `false`，并保留 TaskAI 的默认鼠标剪贴板行为

### Requirement: 终端在创建时快照菜单策略

系统 SHALL 在执行显示终端的自定义菜单命令时，将该命令当前的鼠标剪贴板设置写入新终端的信息记录。已经创建的终端 MUST NOT 因后续编辑菜单配置而改变行为。

#### Scenario: 已配置的命令创建终端

- **WHEN** 用户执行一个 `disableTaskAIMouseClipboard: true` 的显示终端菜单命令
- **THEN** 返回给前端的终端记录带有 `disableTaskAIMouseClipboard: true`

#### Scenario: 后续编辑菜单命令

- **WHEN** 用户在某终端启动后修改同一菜单命令的设置
- **THEN** 运行中的终端保留创建时的策略，之后新建的终端使用保存后的新策略

#### Scenario: 其他终端创建入口

- **WHEN** 用户创建普通终端、直接命令终端或执行未启用该设置的菜单命令
- **THEN** 新终端的 `disableTaskAIMouseClipboard` 为 `false`

### Requirement: 已禁用的终端不拦截鼠标剪贴板操作

终端记录的 `disableTaskAIMouseClipboard` 为 `true` 时，TaskAI MUST NOT 因 xterm 选区调用系统剪贴板写入，也 MUST NOT 阻止右键事件、读取系统剪贴板或将剪贴板内容写入终端。键盘输入、终端输出和文件拖放 MUST 保持原有行为。

#### Scenario: 禁用终端中的文本选区

- **WHEN** 用户在已禁用的终端中用鼠标选择文本
- **THEN** TaskAI 不调用系统剪贴板写入，终端程序可按自身鼠标协议处理该操作

#### Scenario: 禁用终端中的右键操作

- **WHEN** 用户在已禁用的终端中点击右键
- **THEN** TaskAI 不取消该事件，不读取此前系统剪贴板内容，也不向 PTY 注入任何文本

#### Scenario: 默认终端中的鼠标操作

- **WHEN** 用户在 `disableTaskAIMouseClipboard` 缺失或为 `false` 的终端中选择文本或点击右键
- **THEN** TaskAI 保留现有的选区自动复制和右键粘贴行为
