## 1. 诊断确认（实现前基线）

- [x] 1.1 编写合成重绘流 PowerShell 测试脚本（循环：光标归位 `ESC[H` + 重写多行 + 光标定位，每秒约 10–20 帧），保存到任务工作目录外可复用位置，并在 `wails dev` 嵌入终端中确认它能复现"光标乱跑"
- [x] 1.2 用 chrome-devtools Performance trace 录制脚本运行 10 秒，确认主线程长任务集中在逐写入块的全屏扫描（`captureTerminalDisplay`）与解析交错上，记录修复前基线数据（实测无长任务、CPU 占用 <0.5%，撕裂主因是逐事件渲染出 ConPTY 批次边界的中间态；批次结构已抓包确认：一帧拆约 256B+1328B 两批、批间歇约 18ms、每批以 `ESC[?25h` 结尾。记录于 verification-baseline.md 并修正设计假设）

## 2. 前端合帧实现

- [x] 2.1 在 `TerminalSessionRegistry` 的 `TerminalSession` 上增加按会话输出缓冲：`handleTerminalEvent` 的 `output` 分支改为入队，首次入队调度一次刷新（实现演进：原计划 rAF 优先，实测批间歇约 18ms 大于渲染帧窗口、合不拢同帧批次，最终改为按"输出静默期 32ms"定时器合批）
- [x] 2.2 实现刷新回调：按到达顺序拼接队列并调用一次 `terminal.write()`；同批后续事件并入同一次冲刷；不同会话互不阻塞
- [x] 2.3 实现兜底与上限：连续输出超过 64ms 截止时限有界冲刷，单会话缓冲 1MB 上限触发同步冲刷（窗口隐藏时定时器照常触发，无需 rAF 兜底）
- [x] 2.4 处理 `exited` 分支：退出前先冲刷该会话缓冲（保持光标原状），退出通知（`terminalExitSnapshotNotice`）在冲刷后写入且 `suppressVisualActivity` 语义保持
- [x] 2.5 处理 `dispose`/`disposeAll`：销毁会话时丢弃缓冲并取消未决的冲刷/光标恢复定时器，不产生悬挂回调
- [x] 2.6 （实现中新增）中间态光标抑制：判定为 ConPTY 绘制批次的冲刷（≥64 字符且真实以 `ESC[?25h` 结尾）在写入末尾追加合成 `ESC[?25l`，输出静默 48ms 后补写 `ESC[?25h` 恢复（批间继续输出时跳过）；小冲刷与不含光标序列的流式纯文本保持光标可见

## 3. 画面检测收敛

- [x] 3.1 验证合并写入后 `onWriteParsed` 的画面比较收敛为每批次至多一次（每批次仅一次 `terminal.write`，xterm.js 对每次 `write()` 解析完成触发一次 `onWriteParsed`，天然收敛，无需去重标记）
- [x] 3.2 确认比较语义未变：字符、宽度、前景/背景色、样式、`baseY` 起算的活动终端页范围均与现状一致（对照 `terminal-output-status-detection` 规格）（`captureTerminalDisplay`/`terminalDisplaysEqual` 未改动，现有 359 项测试全数通过）

## 4. 单元测试

- [x] 4.1 同一静默期多次输出事件合并为一次 `terminal.write`（断言 write 调用次数与拼接顺序）
- [x] 4.2 跨静默期输出分批写入且顺序保持；exited 事件先冲刷缓冲再写退出通知
- [x] 4.3 静默期满定时器冲刷；缓冲超上限立即同步冲刷；连续输出超过截止时限按截止冲刷
- [x] 4.4 会话销毁后无悬挂回调；`output-change` 检测在合并批次末比较（含"同批改写又还原不计活动"场景）
- [x] 4.5 光标抑制与恢复：绘制批次追加合成隐藏序列、静默后恢复、批间继续输出跳过恢复、小冲刷不隐藏、纯文本流不隐藏
- [x] 4.6 运行 `cd frontend && npm test` 与 `npm run build` 全量通过（359 项测试通过，构建成功）

## 5. 集成验证

- [x] 5.1 `wails dev` 启动应用，从输出获取调试地址，用浏览器工具打开应用，在任务终端运行合成重绘脚本：光标不同位置数从 90-102 收敛到 6-7 且仅剩合法位置（键入回显位、帧末停靠位、结束提示位），重绘期间光标处于隐藏区间、帧间隙出现在停靠位（数据见 verification-baseline.md 修复后表格）
- [x] 5.2 Performance 对比：修复前后 longtask 均为 0，无性能回归；每批次至多一次画面比较（决策 3 + 单测）；后台会话（重绘期间切到其他终端约 8 秒再切回）画面为最新帧（18 行全部第 300 帧）、DONE 完整、无丢失
- [x] 5.3 冒烟测试：普通 shell 输出、`title-change` 数据源（OSC 0/2 标题更新终端名）、`output-change` 状态（working→idle）、字号/主题切换并还原均行为不变；shell `exit` 后 exited 事件在本机长时间不到达（旁路监听证实，与本机 Go ConPTY 退出测试超时失败同源的既有后端问题，非本变更回归，前端链路由单测覆盖）；验证完成后关闭 `wails dev` 调试进程

## 6. 收尾

- [ ] 6.1 按开发流程合并 worktree 分支、编译验证、同步 `openspec/specs/` 与 `docs/plans/`、提交 git 变更、移除已合并的 worktree
