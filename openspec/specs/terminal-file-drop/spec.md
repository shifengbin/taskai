# terminal-file-drop Specification

## Purpose
定义桌面应用中终端内容区域接收本地文件拖放的行为：系统依据目标终端创建时的 Shell 逐个转义绝对路径，再作为不自动执行的输入写入现有 PTY；终端外投放不会导航或写入数据。
## Requirements
### Requirement: 终端内容区接收原生文件拖放
系统 SHALL 启用 Wails 原生文件拖放，并且只在当前活动终端的内容区域作为 Wails drop target 时处理文件路径。投放到终端区域以外不得向任何 PTY 写入数据。

#### Scenario: 向当前终端拖入文件
- **WHEN** 用户将一个或多个本地文件投放到当前活动终端的内容区域
- **THEN** 系统接收 Wails 提供的绝对路径，并将该投放路由到该终端会话

#### Scenario: 向终端区域外拖入文件
- **WHEN** 用户将本地文件投放到任务列表、设置界面或没有活动终端的区域
- **THEN** 系统不得向任何终端写入路径文本，并阻止 WebView 打开该文件或导航离开应用界面

#### Scenario: Linux 投放由原生层完成
- **WHEN** Linux 上的 Wails 2.12 接收本地文件投放
- **THEN** 原生 GTK 回调必须保留 Wails 的路径收集和转发，并在转发绝对路径后显式完成投放，使 WebKit 不得以默认行为打开该文件或替换应用界面

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

### Requirement: 拖放只插入完整的非执行输入
系统 SHALL 在所有路径成功格式化且目标终端仍活动后，一次写入完整路径文本。系统 MUST 不追加回车、换行或命令分隔符，也不得自动执行终端中的任何命令。

#### Scenario: 终端已有待编辑命令
- **WHEN** 用户在终端提示符中已有待编辑的命令并投放文件
- **THEN** 系统只在光标位置插入已转义路径文本，且命令保持待执行状态

#### Scenario: Shell 未被支持或终端已关闭
- **WHEN** 目标终端的 Shell 未被识别、拖入路径为空或目标终端在写入前已退出
- **THEN** 系统不得写入原始路径或部分格式化结果，并向前端调用返回错误
