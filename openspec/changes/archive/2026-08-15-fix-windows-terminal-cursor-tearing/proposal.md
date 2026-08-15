## Why

在 Windows 上，嵌入终端中运行 codex 等高频重绘的 TUI 程序时，每当字符发生变化，光标会在屏幕上乱跑（渲染出重绘的中间态）；同样的程序在系统终端中显示正常，Unix 平台的嵌入终端也没有此问题。实测根因（见 verification-baseline.md）：Windows 独有的 ConPTY 会把程序的一帧重绘拆成多个间隔约 16-30ms 的绘制批次，每个批次以 `ESC[?25l`（隐藏光标）开头、`ESC[?25h`（显示光标）结尾。前端把每个输出事件单独写入 xterm.js，xterm.js 在每个批次边界上渲染，而批次结尾的 `ESC[?25h` 使光标以"显示"状态停留在该批次的绘制中断处——这些中间态就是用户看到的"光标乱跑"。系统终端以整帧为渲染边界，因此不出现此问题。排除的假设：不是主线程 CPU 饱和（longtask 为 0，write/parse/render 采样占比 <0.5%）。

## What Changes

- 前端终端会话层把 PTY 输出事件按会话缓冲，以"输出静默期"合批：静默满约 32ms 才整体冲刷（合并同一重绘帧的全部绘制批次，渲染出帧末真实状态）；连续输出按约 64ms 截止时限有界冲刷；单会话缓冲 1MB 上限立即同步冲刷。
- 对判定为 ConPTY 绘制批次的冲刷（约 ≥64 字符且真实以 `ESC[?25h` 结尾），在写入末尾追加合成的 `ESC[?25l`，使渲染出的任何中间态处于光标隐藏区间；输出静默约 48ms 后补写 `ESC[?25h` 恢复显示（仅当流末光标本应为显示态；批间继续输出时恢复被跳过）。小冲刷（如击键回显）与不含光标序列的流式纯文本不隐藏光标。
- 终端画面变化检测（`output-change` 状态判定方式）从"每个解析块后全屏扫描一次"降为"每个合并批次扫描一次"，保留既有的逐格比较语义（字符、宽度、颜色、样式）。
- 不改变 Go 侧 ConPTY 后端、PTY 事件协议和既有跨平台终端行为；`title-change`、`http` 状态判定方式不受影响。

## Capabilities

### New Capabilities

- `terminal-output-frame-batching`: 前端把同一静默期内到达的 PTY 输出事件按会话合并为一次写入，包括静默/截止的时序边界、缓冲上限、顺序保证与中间态光标抑制。

### Modified Capabilities

- `terminal-output-status-detection`: 画面变化检测的执行时机从"每个解析块后"改为"每个合并批次解析完成后"，检测在批次末快照上执行；同一批次内被改写又被还原的瞬时变化不再单独计为一次活动，活动与静默计时的判定语义保持不变。

## Impact

- `frontend/src/terminal-session.ts`：`handleTerminalEvent` 输出路径改为按会话静默合批冲刷，冲刷时按需追加/恢复光标序列；`onWriteParsed` 里的 `captureTerminalDisplay` 调用频率随之降低。
- `frontend/src/terminal-session.test.ts`：合批与顺序、光标抑制与恢复、截止与上限的单元测试。
- `openspec/specs/terminal-output-status-detection/spec.md`：检测时机的需求更新。
- 不涉及 Wails 绑定、Go 导出方法或存储格式的变更；Unix 平台行为路径不变（合批对透传流同样适用但不改变可见行为）。
- 集成验证依赖 `wails dev` + 浏览器调试（chrome-devtools MCP），需要一个可复现的高频重绘测试命令（用 PowerShell 脚本合成 VT 重绘流，不依赖 codex 可执行文件存在）。
