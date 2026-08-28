## Why

终端输出合批为抑制重绘中间态而追加 `ESC[?25l`，会把 xterm 的光标可见性永久改为隐藏。恢复定时器遇到后续待处理输出时会提前结束，普通输出或左右键输入又不会自动显示光标，导致 Windows 和 Linux 上均可能出现光标消失、闪烁频率异常或移动后不显示。当前测试桩只记录写入参数，未覆盖 xterm 的真实光标状态。

## What Changes

- 保留按会话进行的输出静默合批、截止冲刷、顺序保证和缓冲上限。
- 移除合成 `ESC[?25l` / `ESC[?25h` 及其恢复定时器，不再通过应用层写入修改终端程序的光标可见性、闪烁设置或光标样式。
- 使用 xterm 支持的同步输出模式抑制合批期间的中间态渲染；同步状态跨越截止冲刷，在输出静默、退出、分离和销毁时可靠关闭并渲染最终画面。
- 让 Windows ConPTY、Unix PTY 以及使用相同 VT 控制序列的终端程序遵循同一套渲染同步行为，不按操作系统猜测输出格式。
- 增加真实 xterm/浏览器渲染验证，覆盖普通输出、连续重绘、左右键输入、光标闪烁和会话生命周期；保留输出顺序、后台会话切回及状态检测行为。

## Capabilities

### New Capabilities

### Modified Capabilities

- `terminal-output-frame-batching`: 修改高频输出的渲染抑制契约，改为使用不改变终端光标状态的同步输出模式，并要求在后续普通输出和用户输入后光标可见性、闪烁设置及程序自身控制序列保持正确。

## Impact

- `frontend/src/terminal-session.ts`：调整输出冲刷状态机，移除光标隐藏/恢复序列，增加同步输出模式的会话级生命周期管理。
- `frontend/src/terminal-session.test.ts`：补充同步输出状态、连续截止冲刷、普通输出和销毁清理的单元测试，并增加真实 xterm 光标状态覆盖。
- 前端集成验证：使用 `wails dev` 和浏览器调试工具检查 `.xterm-cursor` 的可见性、闪烁节奏及左右键后的最终位置。
- `openspec/specs/terminal-output-frame-batching/spec.md`：同步新的跨平台渲染同步与光标状态保持要求。
- 不修改 Go 侧 PTY/ConPTY 后端、事件协议、Wails 绑定或持久化数据格式。
