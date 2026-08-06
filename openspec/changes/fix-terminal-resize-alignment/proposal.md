## Why

终端输出在 Windows 上偶发折行/光标错位（大部分时间正常）。根因是 xterm 网格与 ConPTY/Shell 的尺寸**短暂不一致**：xterm 网格在 `fit()` 后瞬时切换到新列数，而 PTY 尺寸要经过一次异步 IPC + ConPTY 重排才跟上。这段“分歧窗口”内 Shell 仍按旧列数折行输出，且已折行的内容不会被重排，于是结构化输出（TUI、表格、进度条、光标定位）留下错位，直到滚出可视区。

Windows 上尤其明显：ConPTY 的 `Resize` 延迟远高于 Unix `TIOCSWINSZ`，分歧窗口更宽；显示缩放（125%/150%）使 `FitAddon` 在拖拽过程中算出的列数在 N / N±1 之间抖动，少量鼠标移动就产生大量“中间尺寸”重排请求，反复触发该竞态。两个具体入口放大了问题：(1) 终端创建时固定以 `100×32` 启动 ConPTY（`App.tsx` `createTerminal`），与真实容器尺寸不符，首批输出在首次 `fit()` 生效前就已按 100 列折行；(2) `ResizeObserver` 未做防抖，每个中间帧都把抖动列数同步给 ConPTY。

## What Changes

- 终端创建时以**真实容器尺寸**初始化 PTY，而非固定的 `100×32`，消除“首次输出按猜测尺寸折行”的窗口。
- 对容器尺寸变化做**合并/防抖**，仅在尺寸稳定后把最终列/行同步给 PTY，避免把抖动的中间列数灌给 ConPTY。
- 收紧 xterm 网格与 PTY 尺寸的**一致性时序**：尺寸变化时优先保证 PTY 与网格尽快对齐，杜绝输出交互期间可观测的折行/光标错位。

## Capabilities

### New Capabilities
<!-- 无新增能力：该不变量已部分由现有能力承载，本次为强化既有要求。 -->

### Modified Capabilities
- `terminal-switch-rendering`: 强化“容器尺寸变化后保持完整终端绘制”要求——明确 PTY 尺寸须与 xterm 网格在输出交互期间保持一致（非仅最终一致），并新增“终端以真实容器尺寸创建”与“尺寸变化期间并发输出不错位”两类场景。

## Impact

- `frontend/src/App.tsx`：`createTerminal` 不再硬编码 `100×32`，改为创建前测量目标容器尺寸后传入；`onResize` 的同步策略调整（合并/防抖）。
- `frontend/src/components/TerminalView.tsx`：`ResizeObserver` 回调由“每帧直发”改为合并到稳定值后再 `fitAndRefresh`。
- `frontend/src/terminal-session.ts`：`fit`/`fitAndRefresh` 与尺寸同步的调用时序对齐到新要求。
- `internal/terminal/manager.go` / `backend_windows.go`：`Create*`/`Resize` 已接受 `columns, rows`，**无需后端 API 变更**；仅在调用侧修正传参与时序。
- 无破坏性改动；不影响 Unix 后端既有行为。
