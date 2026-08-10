## Why

终端快捷键会把固定按键序列（如 `Shift+Enter` → `\` + 回车）发送到当前聚焦终端。但在 codex 这类交互式 TUI 程序里，原始组合键本身另有用途（如换行），快捷键拦截会造成冲突。当前没有手段把某个快捷键的作用范围收窄到特定程序：用户无法让一条快捷键只在 shell 里生效、在 TUI 程序里自动让位（无需逐一枚举 codex、vim、less 等无界增长的程序黑名单）。

## What Changes

- 为终端快捷键新增可选的"生效程序"列表，采用 **include（包含）语义**：留空时维持现状（所有终端生效）；填入程序名后，快捷键仅在该终端由匹配的启动命令创建时才拦截执行，否则把原始按键透传给终端。
- 程序判断依据是 **TaskAI 创建该终端时使用的启动命令**（任务菜单命令项、生命周期命令链创建的命令终端），**不**跟踪用户在 shell 内手动运行的子程序——这类终端按其启动命令（shell）判断。
- 匹配按启动命令的文件名（去目录、去 Windows 扩展名、忽略大小写）做归一化后精确匹配；列表中任一命中即视为在该终端生效。（子串匹配因误伤问题被排除，见 design.md。）
- 将终端启动命令从后端会话记录透传到前端终端记录，供快捷键做范围判断。
- 在设置编辑器中为每条快捷键提供"生效程序"列表的新增、编辑与删除。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `terminal-shortcut-inputs`: 新增"按启动命令收窄快捷键生效范围"的需求——快捷键在生效范围外时 MUST 透传原始按键（保持 xterm 原有行为），在范围内时拦截执行的行为不变；并新增对该列表的持久化、校验与编辑要求。

## Impact

- Go：`internal/terminal/types.go` 的 `Info` 增加启动命令字段；`internal/terminal/manager.go` 已在 `managedSession.command` 记录启动命令，需在构造返回 `Info` 时带出；`internal/settings/settings.go` 的 `TerminalShortcut` 增加 `IncludePrograms` 字段及归一化、校验（拒绝未知修饰、去重、去空白，接受空列表）。
- 前端：`types.ts` 的 `TerminalShortcut` 增加 `includePrograms?`、`TerminalRecord` 增加 `command?`；`terminal-shortcuts.ts` 增加启动命令与生效列表的匹配判断；`TerminalView.tsx` 的键盘事件闸门接入范围判断（范围外 `return true` 透传）；`TerminalShortcutSettings.tsx` 增加"生效程序"编辑控件；终端记录构造路径传递 `command`。
- Wails 绑定（`frontend/wailsjs/`）需重新生成。
- 兼容性：所有新增字段均为可选；缺少字段的既有设置按"全部终端生效"加载、既有终端记录按 `command` 缺失处理（等价于无法匹配任何非空生效列表，但空列表=全部，故行为不变），无需数据迁移。
