## MODIFIED Requirements

### Requirement: 使用模拟粘贴写入当前终端
系统 MUST 使用当前 xterm 会话的模拟粘贴机制向所选活动终端写入快捷输入完整内容，并在默认 TaskAI 鼠标剪贴板策略的终端中使用相同机制写入右键读取的非空系统剪贴板内容。系统 MUST NOT 在内容前后额外添加换行、Enter 或命令执行字符。系统 MUST NOT 将内容写入其他终端；目标会话已关闭、缺失或写入失败时，系统 MUST 报告快捷输入错误或忽略右键粘贴，且不得把内容路由到其他会话。

多行内容在目标 Shell 或终端程序启用 bracketed paste 时 MUST 以该终端协议的粘贴语义传递。系统 MUST NOT 承诺未支持 bracketed paste 的第三方程序不会把内容中的换行当作提交。

#### Scenario: 插入单行内容不追加执行字符
- **WHEN** 用户选择内容为 `git status` 的快捷输入
- **THEN** 系统仅通过模拟粘贴写入 `git status`，且不追加换行或其他执行字符

#### Scenario: 插入多行内容
- **WHEN** 用户选择包含多行文本的快捷输入，且目标程序已启用 bracketed paste
- **THEN** 系统通过 xterm 模拟粘贴将完整文本传给当前终端，使目标程序按粘贴内容处理而非由 TaskAI 额外提交命令

#### Scenario: 右键粘贴多行内容
- **WHEN** 用户在使用默认 TaskAI 鼠标剪贴板策略的活动终端中右键，系统剪贴板包含多行文本且目标程序已启用 bracketed paste
- **THEN** 系统通过 xterm 模拟粘贴将完整剪贴板文本传给该终端，不追加 Enter 或其他执行字符

#### Scenario: 右键读取期间目标终端关闭
- **WHEN** 用户右键后系统正在读取剪贴板，目标终端在读取完成前关闭
- **THEN** 系统不向任何 PTY 写入剪贴板文本，也不将其路由到其他终端

#### Scenario: 目标终端已关闭
- **WHEN** 用户打开选择器后目标终端在选择前关闭
- **THEN** 系统报告该终端不可用，且不得将快捷输入写入其他终端
