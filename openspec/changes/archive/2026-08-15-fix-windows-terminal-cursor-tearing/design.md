## Context

症状与根因（见 proposal 与 verification-baseline.md 的实测修正）：Windows 上 ConPTY 把 codex 等程序的一帧 VT 重绘拆成多个间隔约 16-30ms 的绘制批次（实测一帧约 1584B 拆为约 256B + 1328B 两批），每个批次以 `ESC[?25l` 开头、`ESC[?25h` 结尾。前端对每个输出事件单独调用 `terminal.write()`（`frontend/src/terminal-session.ts` 的 `handleTerminalEvent`），xterm.js 在每个批次边界渲染；批次结尾的 `ESC[?25h` 使光标以显示态停在绘制中断处，这些中间态即"光标乱跑"。已排除主线程 CPU 饱和（longtask 为 0，write/parse/render 的 CPU 采样占比 <0.5%）。系统终端以整帧为渲染边界，因此正常。

相关既有约束：

- `terminal-output-status-detection` 规格要求画面比较包含字符内容、字符宽度、前景色、背景色和可见文本样式，且以 `baseY` 起算的活动终端页为比较范围。本设计不得弱化比较语义，只调整比较时机。
- `terminal-session.ts` 同时服务被选中和后台（未挂载 DOM）的会话；合批逻辑必须对后台会话同样生效。
- xterm.js 的 `write()` 本身是异步解析、渲染内部按渲染帧去抖；问题不在解析分片，而在"每个批次末尾光标为显示态"的中间态被渲染出来。

## Goals / Non-Goals

**Goals:**

- 消除 Windows 嵌入终端中高频重绘 TUI 的光标乱跑：可见画面只呈现重绘帧的末态，重绘过程中光标不可见地跳到中间位置。
- 降低 `output-change` 判定方式下的主线程开销：全屏扫描频率从"每个解析块"降到"每个合批批次"。
- 保证输出顺序、不丢数据：合批只做合并，不做重排或丢弃；缓冲有上限与截止时限。
- Unix 平台行为不回退：合批对透传流同样适用（表现为合并写入，无可见行为差异）。

**Non-Goals:**

- 不修改 Go 侧 ConPTY/PTY 后端、事件协议或 Wails 绑定。
- 不更换 xterm.js 渲染器、不引入 Web Worker 解析。
- 不解决右键菜单创建终端时初始尺寸写死 `100×32`（`App.tsx` 的 `runTaskMenuCommand`）导致的首次 resize 闪跳；该问题记录为后续独立改进。
- 不改变 `title-change`、`http` 状态判定方式的行为。

## Decisions

### 决策 1：按会话以"输出静默期"合批，定时器驱动冲刷

在每个 `TerminalSession` 上增加输出缓冲队列。`handleTerminalEvent` 收到 `output` 事件时只入队并重置静默计时；静默满约 32ms（`terminalOutputQuietFlushMs`）才把队列按到达顺序拼接为一次 `terminal.write()`。ConPTY 的批间歇期约 16-30ms、帧间隔（如 15fps）约 66ms，32ms 静默窗能吞下同帧的批间歇、又不会跨越帧边界。

- 为什么不是 rAF 合帧（原方案，实测无效）：批间歇约 18ms 大于渲染帧窗口约 16ms，rAF 边界与批次边界不对齐，合不拢同一帧的批次；且窗口隐藏时 rAF 停止。以"输出静默"为节拍与 ConPTY 的批次节奏天然对齐。
- 连续输出不能无限推迟：自首批入队起约 64ms（`terminalOutputMaxFlushDelayMs`）截止，到期按剩余时间冲刷，保证延迟有界。
- 单会话缓冲超过 1MB（`terminalOutputBufferLimit`）立即同步冲刷，防内存堆积。
- 为什么按会话各自缓冲：会话间互不阻塞，一个终端的大输出不会推迟其他终端的冲刷。
- `exited` 事件先冲刷该会话缓冲（保持光标原状），退出通知（`terminalExitSnapshotNotice`）在冲刷后写入；`dispose` 丢弃缓冲并取消未决定时器。

### 决策 2：中间态光标抑制——合成 `ESC[?25l` / 延迟恢复 `ESC[?25h`

静默合批覆盖大多数场景，但截止时限冲刷（连续输出 >64ms）仍会写出"半帧"内容，且批次结尾的 `ESC[?25h` 会让中间态光标可见。因此对判定为 ConPTY 绘制批次的冲刷追加处理：

- 判定条件：合并数据 ≥64 字符（`terminalCursorSuppressMinBytes`）且真实出现 `ESC[?25h` 且其位置不早于最后的 `ESC[?25l`（即批次以显示态结尾）。不含光标序列的流式纯文本（如构建日志）和小的击键回显不满足条件，光标保持可见。
- 冲刷时在写入末尾追加合成的 `ESC[?25l`：渲染出的中间态全部处于光标隐藏区间。
- 同时调度约 48ms（`terminalCursorRestoreDelayMs`）的恢复定时器：期间若又有输出入队（批间继续输出）则恢复被跳过——光标持续隐藏；输出真正静默后补写 `ESC[?25h` 恢复显示。48ms 大于批间歇（可达 30ms）而小于帧间隔（约 66ms@15fps），避免批间提前恢复造成短暂闪现，帧末光标照常出现。
- 原样写入的批次若自带光标序列，撤销先前挂起的合成恢复（流本身已决定光标态）。

### 决策 3：画面变化检测改为"每批次一次"，比较语义不变

`onWriteParsed` 回调仍由 xterm.js 触发，但决策 1 把一批数据合并为一次 `write()`，xterm 的 `onWriteParsed` 自然收敛到每批次一次；`suppressVisualActivity` 语义保留（退出通知所在批次不触发活动）。

- 语义影响：同一批次内"改写又被还原"的瞬时变化不再单独计为一次活动；活动判定以批次末快照为准。这是对 `terminal-output-status-detection` 需求时序的唯一修改，静默计时（1.5 秒）相对批次时长不受影响。
- 比较内容（字符、宽度、前景/背景色、样式）与比较范围（`baseY` 起算的活动终端页）完全不变。

### 决策 4：验证手段——合成重绘流脚本 + 光标位置采样，不依赖 codex

集成测试与人工复现使用 PowerShell 脚本（`scripts/terminal-redraw-stress.ps1`，ASCII 输出避开 PS 5.1 控制台 GBK 编码问题）在嵌入终端中合成高频重绘流（每帧 `ESC[H` 归位 + 20 行逐行定位整行重写 + 帧末光标停靠固定位置）。度量方式：headless 浏览器 rAF 循环采样 `.xterm-cursor` 的 getBoundingClientRect（经 getComputedStyle 过滤不可见样本），统计光标不同位置数与停靠位占用比，辅以 PerformanceObserver longtask。修复目标：光标不同位置数从约 90-102 收敛到个位数、且仅剩合法位置（键入回显位、帧末停靠位、结束后提示位）。

## Risks / Trade-offs

- [合批引入至多 32ms 输出延迟] → 远低于人眼对滚动输出的感知阈值；连续输出场景由 64ms 截止兜底。
- [持续重绘期间光标不可见（恢复被跳过）] → 与系统终端行为一致（整帧渲染时用户看到的也是帧末位置）；帧间隙 >48ms 时光标照常出现。
- [极端大输出下缓冲占用内存] → 1MB 上限触发同步刷新；上限值在单元测试中覆盖。
- [后台终端输出不再即时写入 xterm，切换回来时是否有差异] → 后台会话本就不渲染，合批只减少解析次数，切换回来显示的是最新状态，无行为差异（集成测试验证）。
- [`output-change` 判定的活动计数略微减少（批次内瞬时变化合并）] → 规格增量明确该时序语义，现有场景语义（持续变化推迟静默、1.5 秒静默）不变。
- [合成光标序列与程序自身的光标控制交错] → 仅在批次以 `?25h` 结尾时追加隐藏；恢复仅在无后续输出且终端未关闭时执行；原样批次自带光标序列时撤销挂起恢复。单元测试覆盖这些边界。

## Migration Plan

纯前端行为变更，无数据迁移。合批逻辑集中在 `terminal-session.ts`，随正常发版交付；出问题时的回滚策略是还原该文件的单点改动，不涉及协议或存储。

## Open Questions

- 静默 32ms / 截止 64ms / 恢复 48ms / 阈值 64 字符是否需要在设置中暴露？当前倾向不暴露（内部实现细节）。
- 未来是否值得在 Go 侧对 sub-frame 事件做软合并以降低 Wails 事件桥开销？本变更不做，待性能数据说话。
- 本机 ConPTY 退出检测存在既有问题（Go 测试 `TestWindowsBackendStartsCommandProcessorInRequestedDirectory` 等超时失败、shell `exit` 后 exited 事件长时间不到达前端），与本变更无关，待单独排查。
