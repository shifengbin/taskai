## Why

终端右键粘贴目前绕过 xterm 的模拟粘贴机制，直接将剪贴板文本写入 PTY。多行文本中的换行因此可能被 Shell 当作回车，变成多条命令；快捷输入已经具备正确的粘贴语义，应让两种入口保持一致。

## What Changes

- 右键粘贴通过当前 xterm 会话的模拟粘贴 API 写入完整剪贴板文本。
- 不追加 Enter、换行或其他命令执行字符，也不改变键盘输入和文件拖放路径。
- 终端未启用 bracketed paste 时的行为与快捷输入保持一致，不额外承诺第三方程序的多行处理方式。
- 增加前端会话与终端视图测试，覆盖多行内容、关闭终端和鼠标剪贴板禁用策略。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `terminal-quick-inputs`: 明确右键剪贴板粘贴与快捷输入共用模拟粘贴语义，并保留多行及执行字符边界。

## Impact

- 影响 `frontend/src/components/TerminalView.tsx` 与 `frontend/src/terminal-session.ts` 的输入路由。
- 更新相关 Vitest 测试和 `openspec/specs/terminal-quick-inputs/spec.md`。
- 不新增后端 API，不改变 PTY、终端创建或持久化数据结构。
