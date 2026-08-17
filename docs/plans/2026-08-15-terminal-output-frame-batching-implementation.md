# 终端输出合批与光标撕裂修复实施记录

## 目标

修复 Windows 嵌入终端高频重绘（如 codex 等 TUI）时光标乱跑、画面闪烁的问题：ConPTY 把一帧输出拆成多个批次（实测一帧约拆 256B+1328B 两批、批间歇约 18ms、每批以 `ESC[?25h` 结尾），前端逐事件调用 `terminal.write()` 会把批次边界的中间态渲染出来。

## 实现

全部改动位于 `frontend/src/terminal-session.ts`（配套测试 `frontend/src/terminal-session.test.ts`）：

- `TerminalSession` 按会话缓冲 PTY 输出事件：`handleTerminalEvent` 的 `output` 分支改为入队，首次入队启动约 32ms 的输出静默期定时器。
- 静默期满时按到达顺序拼接缓冲并调用一次 `terminal.write()`；连续输出超过约 64ms 截止时限时有界冲刷；单会话缓冲约 1MB 上限触发同步冲刷。
- 判定为 ConPTY 绘制批次的冲刷（约 ≥64 字符且真实以 `ESC[?25h` 结尾）在写入末尾追加合成 `ESC[?25l` 隐藏光标，输出静默约 48ms 后补写 `ESC[?25h` 恢复；批间继续输出时跳过恢复。小冲刷（击键回显）与纯文本流保持光标可见。
- `exited` 分支先冲刷该会话缓冲再写退出通知，`suppressVisualActivity` 语义不变；`dispose`/`disposeAll` 丢弃缓冲并取消未决的冲刷与光标恢复定时器。
- `onWriteParsed` 的画面比较因每次批次仅一次 `terminal.write()` 天然收敛为每批次至多一次，`captureTerminalDisplay`/`terminalDisplaysEqual` 未改动，比较语义与 `terminal-output-status-detection` 规格一致。

## 设计演进

原设计为 rAF 优先合帧；实测批间歇约 18ms 小于渲染帧窗口、合不拢同帧批次，最终改为"输出静默期 32ms"定时器合批，详见归档变更 `openspec/changes/archive/2026-08-15-fix-windows-terminal-cursor-tearing/` 下的 design.md 与 verification-baseline.md。

## 验证

- `cd frontend && npm test`（359 项全数通过）与 `npm run build`
- `wails dev` + chrome-devtools 集成测试：合成重绘脚本（`scripts/terminal-redraw-stress.ps1`，300 帧/15fps/20 行）下，光标不同位置数从修复前 90-102 收敛到 6-7 且仅剩合法位置；Performance 无长任务回归；后台会话切回画面为最新帧无丢失；普通输出、标题/输出状态判定、终端退出展示、字号/主题切换冒烟通过
- `scripts/build-windows.ps1` 编译通过

## 已知非回归问题

本机 shell `exit` 后 `exited` 事件长时间不到达（与 Go ConPTY 退出测试超时失败同源的既有后端问题），前端链路由单测覆盖。
