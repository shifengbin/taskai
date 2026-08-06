## Context

终端视图用 xterm.js 渲染、用 ConPTY（Windows）/ PTY（Unix）承载 Shell。尺寸链路为：

```
ResizeObserver ──► fit() ──► xterm.resize(cols,rows)   [瞬时]
                         └─► onResize(cols,rows) ──IPC──► conpty.Resize   [异步、有延迟]
```

xterm 网格在 `fit()` 后瞬时切换，PTY 尺寸要等一次 IPC + ConPTY 重排才跟上。这段“分歧窗口”内 Shell 仍按旧列数折行，且已折行内容不会被重排 → 结构化输出错位。当前实现放大了该窗口：

- `App.tsx:createTerminal` 固定以 `100×32` 创建 ConPTY，与真实容器不符；首批输出在首次 `fit()` 生效前已按 100 列折行。
- `TerminalView.tsx` 的 `ResizeObserver` 未防抖，每个中间帧都 `fit()` 并把抖动列数（Windows 显示缩放下 N/N±1 抖动）同步给 ConPTY。
- `App.tsx` 的 `onResize` 为 fire-and-forget，无合并/去重。

后端 `Manager.Create*`/`Resize` 已接受 `columns, rows`，无需 API 变更；`CreateTerminal(taskID, columns, rows)` 绑定直接可用。Unix 后端 `TIOCSWINSZ` 延迟极低，本问题主要在 Windows。

## Goals / Non-Goals

**Goals:**
- 新终端 PTY 以真实容器尺寸初始化，Shell 首批输出按真实列数折行。
- 容器尺寸变化被合并，PTY 仅收到稳定后的最终尺寸，不收抖动中间帧。
- 尺寸变化期间 xterm 网格与 PTY 保持一致，不出现按旧/中间尺寸折行或光标错位的输出。

**Non-Goals:**
- 消除 ConPTY 单次 Resize 的固有延迟本身（平台限制，不可去除）；但其对渲染的影响已由 D4 关闭（同步期间缓冲输出，不再错位）。
- 回溯重排已折行的结构化输出（Shell 不支持重排）。
- 改动 Unix 后端 resize 路径（延迟已足够低，不受影响）。
- 重构会话生命周期/滚屏保留（由 `terminal-output-retention` 覆盖）。

## Decisions

### D1：新终端以真实容器尺寸创建（替换 `100×32`）

- **D1a（首选，无后端变更）—— 复用共享面板几何**：内容区面板对所有终端共享同一尺寸。`TerminalSessionRegistry` 在每次 `fit()` 成功后缓存最近一次的 `(cols, rows)`；`createTerminal` 读取该缓存值传入 `CreateTerminal(taskID, cols, rows)`。对“尚无任何会话”的首个终端，用内容区容器的 DOM 尺寸 + xterm 当前字体度量估算，或退化为默认值并在挂载 `fit()` 后立即同步（仅每会话首次、无输出交互）。
- **D1b（备选，最干净不变量，需小幅后端调整）—— 挂载后再 spawn**：先把终端记录加入 UI 并挂载 `TerminalView`、`fit()` 测得真实尺寸，再以该尺寸 spawn PTY。需前端拥有/生成终端 ID（新增接受 ID 的绑定变体，或拆分为 reserve + start）。

**取舍**：D1a 不改后端、覆盖常见情况（绝大多数终端在已有会话后创建）；D1b 给出绝对干净的首尺寸不变量但引入生命周期拆分。先实现 D1a；若首个终端边界不可接受再升级 D1b。

*考虑过的替代方案*：保留 `100×32` 并在挂载后立即 `resize` —— 治标不治本，首批输出仍可能在 resize 生效前按 100 列折行。否决。

### D2：合并容器尺寸变化（防抖网格 + PTY 一起）

用 trailing 防抖（建议 ~100ms）包裹 `ResizeObserver` 回调，**网格与 PTY 同步一起防抖**：拖拽期间终端保持上一次稳定尺寸（网格与 PTY 一致），尺寸稳定后再一次性 `fit()` + 同步 PTY + `refresh`。

- **为何一起防抖**：若只防抖 PTY、网格实时跟随，会重新打开“网格已变/PTY 未变”的分歧窗口，正是本 bug 根因。两者一起防抖才能保证任意时刻网格与 PTY 一致。
- **去重**：仅当 `cols/rows` 实际变化时才触发 PTY 同步，减少无谓 IPC。
- **挂载/切换路径不变**：首次挂载与切换终端时的 `attach → fit → refresh` 仍立即执行（见 `terminal-switch-rendering` 既有要求），防抖只作用于“持续的 ResizeObserver 流”。

*考虑过的替代方案*：rAF 节流而非时间防抖 —— rAF 仍会在每帧产出抖动列数并同步给 ConPTY，不能消除 N/N±1 抖动。否决，改用时间防抖取稳定值。

### D3：单次 fit 内保持紧凑时序

维持 `fit()` 的“先 `fitAddon.fit()` 再同步读 `terminal.cols/rows`”的同步顺序（已足够紧凑），但拆分职责：`fit()` 只同步适配网格 + 去重，返回待同步尺寸；PTY 同步交由 D4 的 `syncPty` 异步完成。单次 Resize 的固有 IPC 延迟无法消除，但其渲染影响由 D4 关闭。

### D4：PTY 同步期间缓冲输出（关闭分歧窗口 —— 真正的根因修复）

**触发**：D2 防抖 + D1 真实尺寸落地后，用户实机仍报告“放大窗口时附近几行错位且自愈”。证据（自愈、仅附近几行、放大后、200% 缩放）确认根因不是 DPI 抖动，而是**分歧窗口内到达的输出被按新网格渲染**：`fit()` 瞬时把 xterm 网格切到新列数，而 PTY 要等一次 IPC + ConPTY 重排才跟上；窗口内 Shell/历史回放按旧宽度发出的内容写进新网格 → 错位，随后正确宽度的输出覆盖 → 自愈。防抖只降低了窗口出现频率，并未在“稳定那一刻”使网格与 PTY 原子一致。

**修复**（前端、无后端变更）：把“适配网格”与“同步 PTY”解耦，并在两者之间缓冲输出。

- `fit(session)`：同步 `fitAddon.fit()` + 去重，返回待同步的 `{columns, rows}`（或 `undefined`）。
- `syncPty(session, dims, onResize)`：置 `session.resizeInFlight = true` → `await onResize(cols, rows)`（IPC 完成、ConPTY 生效）→ `finally` 清标志 + `flushOutput`。
- `handleTerminalEvent`：输出到达时若 `resizeInFlight`，则压入 `session.outputQueue`，否则直接 `terminal.write`。
- `fitAndRefresh`/`attach`：同步 `fit()` + `refreshVisibleRows()`（网格立即可见、不被异步延迟），再 `void syncPty(...)` 异步对齐 PTY。
- `App.tsx` 的 `onResize` 由 `void api.resizeTerminal(...)`（丢弃 Promise）改为返回该 Promise（`.catch` 兜底），使 `syncPty` 可 `await` 到 ConPTY 真正生效。

**为何网格先、PTY 后仍正确**：用户复现为“先输入、再放大”，拖拽期间 Shell 空闲、无行编辑输出；窗口内唯一可能错位的是历史回放/Shell 对 resize 的重绘——这些在 `resizeInFlight` 期间被缓冲，待 PTY 生效后按新网格写入即正确。xterm 自身的 reflow 是正确且期望的行为（加宽时合并软折行），不属错位。

**取舍/残量**：若恰好在 IPC 返回后、Shell 处理 resize 事件前的纳秒窗口内有行编辑输出，理论仍可能瞬态错位；该窗口远小于原 IPC 往返窗口，且非用户复现路径，列为可接受残量。D2 的防抖仍保留以避免把抖动列数逐帧灌给 ConPTY。

## Risks / Trade-offs

- **[拖拽期间终端不实时跟随尺寸]** → 取舍：以正确性换流畅度。缓解：trailing 间隔取短（~100ms），用户松手即刷新；多数用户为“拖-停”式调整。
- **[D1a 首个终端无缓存几何]** → 退化为估算/默认，可能每会话首次仍有一次轻微 resize。缓解：用容器 DOM + 字体度量估算，或接受一次性（无并发输出）。
- **[ConPTY 固有延迟不可去除]** → 单次稳定 Resize 后曾有极短收敛窗口，窗口内输出会按新网格渲染旧宽度内容而瞬态错位。**已由 D4 关闭**：`syncPty` 期间缓冲输出，待 ConPTY 生效后按序写入。残量仅为 IPC 返回后、Shell 处理 resize 事件前的纳秒窗口（非用户复现路径，可接受）。
- **[回归既有切换渲染修复]** → 防抖若误伤挂载/切换路径会导致切换后空白。缓解：防抖仅作用于持续 ResizeObserver 流；挂载/切换保持立即 `fit+refresh`，并补充测试覆盖（见 tasks）。

## Migration Plan

- 主要为前端改动（`App.tsx`、`TerminalView.tsx`、`terminal-session.ts`）；后端 `Create*`/`Resize` 签名不变。若采用 D1b 则新增一个接受 ID 的绑定变体。
- 随下一版发布，无数据迁移。回滚即还原前端三处改动。
- 防抖参数（间隔）作为前端常量，便于按手感调整，无需发版机制。

## Open Questions

- trailing 防抖间隔的取舍值（建议 ~100ms，需按 Windows 实机手感校准）。
- 首个终端边界：接受“估算/默认 + 立即 resize”，还是必须新增字体度量探针以保证首终端也精确？
- 是否在后端 `Manager.Resize` 增加相同尺寸去重作为纵深防御（当前仅前端去重）？
