# Tasks

## 1. 后端：透传终端启动命令

- [x] 1.1 在 `internal/terminal/types.go` 的 `Info` 结构增加 `Command string` 字段（JSON 标签 `command,omitempty`）
- [x] 1.2 在 `internal/terminal/manager.go` 的 `createWithEnvironmentBuilder` 构造 `Info` 时，从已计算的启动命令（`request.Command`，缺失时为 `request.ShellPath`）带出 `Command` 字段
- [x] 1.3 更新 `internal/terminal` 相关测试，断言命令终端与纯 shell 终端的 `Info.Command` 取值正确

## 2. 后端：快捷键生效程序字段与校验

- [x] 2.1 在 `internal/settings/settings.go` 的 `TerminalShortcut` 增加 `IncludePrograms []string` 字段（JSON 标签 `includePrograms,omitempty`）
- [x] 2.2 在 `NormalizeTerminalShortcuts` 中归一化 `IncludePrograms`：逐项去首尾空白、丢弃空白项、按忽略大小写去重
- [x] 2.3 在 `internal/settings/terminal_shortcut_test.go` 增加用例：含/缺失 `IncludePrograms` 的归一化、去重、持久化往返

## 3. 前端：类型与匹配判定

- [x] 3.1 在 `frontend/src/types.ts` 为 `TerminalShortcut` 增加 `includePrograms?: string[]`，为 `TerminalRecord` 增加 `command?: string`
- [x] 3.2 在 `frontend/src/terminal-shortcuts.ts` 新增判定函数（如 `terminalShortcutApplies(shortcut, command)`）：空列表或缺失视为生效；非空时按 basename + 去 Windows 扩展名（`.exe`/`.com`）+ 忽略大小写后精确匹配，任一列表项相等即生效
- [x] 3.3 在 `frontend/src/terminal-shortcuts.test.ts` 增加用例：`codex`/`codex.exe`/`C:\tools\codex.exe`/`/usr/local/bin/codex` 归一化命中、空列表=全部生效、`pwsh` 与 `codex` 不互匹配、`command` 缺失时不匹配非空列表

## 4. 前端：键盘闸门接入范围判定

- [x] 4.1 在 `frontend/src/components/TerminalView.tsx` 的 `customKeyEventHandler` 中，命中快捷键后、`writeInput` 之前调用范围判定；范围外 `return true` 透传原始按键，范围内照旧 `preventDefault` + `writeInput` + `return false`
- [x] 4.2 将 `terminal.command` 纳入该 `useEffect` 依赖数组（或通过 ref 读取），确保判定读到当前终端的启动命令
- [x] 4.3 在 `frontend/src/components/TerminalView.test.tsx` 增加用例：生效范围外透传、生效范围内执行两条路径

## 5. 前端：终端记录传递 command

- [x] 5.1 在终端记录构造路径（任务菜单命令返回的 `TaskMenuCommandResult`、生命周期命令创建、终端列表加载）将后端 `Info.Command`/`command` 复制到 `TerminalRecord.command`
- [x] 5.2 更新 `frontend/src` 相关 api/state 测试，断言 `command` 正确透传到前端终端记录

## 6. 前端：设置 UI 编辑生效程序

- [x] 6.1 在 `frontend/src/components/TerminalShortcutSettings.tsx` 的 `ShortcutCard` 增加"生效程序"列表控件，支持逐项新增与删除，并标注"留空=全部终端生效"
- [x] 6.2 保持"新增快捷键"默认不带 `includePrograms`（等价全部终端生效）
- [x] 6.3 更新设置组件测试，覆盖生效程序的新增、删除与保存草稿/取消

## 7. 绑定与构建验证

- [x] 7.1 重新生成 `frontend/wailsjs/` 绑定，使 `Info.Command` 反映到生成的 TS 类型
- [x] 7.2 运行 `go build ./...` 与前端测试/构建（`npm test`、`npm run build`），确保编译与全部测试通过

## 8. 文档

- [x] 8.1 更新项目文档中终端快捷键相关说明，描述"生效程序"包含语义、归一化规则与"shell 内手动运行不生效"的边界
