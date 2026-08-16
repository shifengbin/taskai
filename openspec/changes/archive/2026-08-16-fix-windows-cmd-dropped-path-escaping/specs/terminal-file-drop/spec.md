# terminal-file-drop 变更

## MODIFIED Requirements

### Requirement: 路径按目标终端的实际 Shell 转义
系统 SHALL 使用目标活动终端创建时的 `ShellPath` 识别 Shell 家族，而不是使用当前默认 Shell 或客户端操作系统，并将每个拖入路径格式化为该 Shell 可解析的一个字面参数。系统 MUST 支持 POSIX Shell、fish、PowerShell 和 `cmd.exe`；多个路径 MUST 以单个 ASCII 空格连接。

对 `cmd.exe`：Windows 10.0.26200 起 cmd.exe 不再处理双引号内的 caret 转义（caret 一律字面保留），因此系统 MUST 以双引号包裹原样路径，MUST NOT 对 `& | < > ^ !` 或空格做 caret 转义；包含双引号的路径 MUST 拒绝。成对 `%VAR%` 在 cmd.exe 中仍会展开，系统接受该行为为 cmd.exe 已知限制（与 Windows Terminal 拖放一致），单个不成对 `%` 必须保持字面。

#### Scenario: POSIX 或 fish Shell 接收包含特殊字符的路径
- **WHEN** `sh`、`bash`、`zsh`、`dash`、`ksh` 或 fish 终端收到包含空格、单引号、通配符或命令分隔符的路径
- **THEN** 写入的文本在对应 Shell 解析后必须只产生与原始路径完全相同的一个参数

#### Scenario: PowerShell 接收包含引号和空格的路径
- **WHEN** PowerShell 终端收到包含空格或单引号的 Windows 路径
- **THEN** 写入的文本在 PowerShell 解析后必须只产生与原始路径完全相同的一个参数

#### Scenario: cmd.exe 接收包含元字符的路径
- **WHEN** `cmd.exe` 终端收到包含空格、`&`、`|`、`<`、`>`、`^`、`!` 或单个 `%` 的 Windows 路径
- **THEN** 系统写入双引号包裹的原样路径，cmd.exe 解析后只产生与原始路径完全相同的一个参数，且不得触发元字符代表的命令语义，也不得残留任何字面 `^`

#### Scenario: cmd.exe 路径包含双引号被拒绝
- **WHEN** `cmd.exe` 终端收到包含双引号的路径
- **THEN** 系统拒绝写入并向调用方返回错误，不写入任何部分结果

#### Scenario: cmd.exe 成对百分号为已知限制
- **WHEN** `cmd.exe` 终端收到路径中恰好包含成对 `%` 且其间文本匹配已定义环境变量名（如 `%PATH%`）
- **THEN** 按下回车时 cmd.exe 自身展开该变量；系统不做转义干预（与 Windows Terminal 行为一致），该限制不影响单个 `%` 或未匹配变量名的路径保持字面

#### Scenario: 一次投放多个文件
- **WHEN** 用户向支持的 Shell 终端一次投放多个文件
- **THEN** 系统按投放路径顺序写入多个独立的已转义参数，并使用单个 ASCII 空格分隔
