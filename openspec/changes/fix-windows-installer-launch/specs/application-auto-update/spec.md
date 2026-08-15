# application-auto-update Delta

## MODIFIED Requirements

### Requirement: 安装遵守平台启动和现有退出语义

系统 SHALL 使用当前平台的安装方式打开已校验安装包：Windows 通过 `ShellExecute` 的 `open` 动词启动 NSIS 安装程序（以触发 UAC 提升）且不额外显示控制台窗口，macOS 通过系统 `open` 打开 DMG，Linux 通过 `xdg-open` 打开 DEB。系统 MUST 先成功启动安装程序，之后才能调用现有 `PrepareQuit` 关闭终端并退出 TaskAI；启动失败时 MUST 保持应用运行并提供重试安装和手动下载。

#### Scenario: 无执行中任务时安装

- **WHEN** 用户确认安装且没有执行中任务
- **THEN** 系统先启动当前平台安装程序，成功后关闭终端并退出 TaskAI

#### Scenario: 有执行中任务时确认安装

- **WHEN** 用户确认安装但仍有执行中任务
- **THEN** 系统显示现有退出确认语义，说明只关闭终端、不改变任务状态或删除工作目录，并提供“关闭终端并安装”动作

#### Scenario: 取消执行中任务确认

- **WHEN** 用户在执行中任务安装确认中选择取消
- **THEN** 系统不启动安装程序、不关闭终端、不退出，并保持已下载状态

#### Scenario: 安装程序启动失败

- **WHEN** 平台安装程序或系统打开命令无法启动
- **THEN** 系统不调用 `PrepareQuit`、保持 TaskAI 运行，并提供重试安装和手动下载

#### Scenario: Windows 安装不弹出控制台窗口

- **WHEN** Windows 用户确认启动已下载的 NSIS 安装程序（其 UAC 清单要求管理员权限）
- **THEN** 系统通过 `ShellExecute` 的 `open` 动词启动安装程序并触发系统 UAC 提升，不使用 `CreateProcess`，且不额外弹出控制台窗口
