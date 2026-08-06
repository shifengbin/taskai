# Tasks

## 1. 前端：新终端以真实容器尺寸创建（D1a）

- [x] 1.1 在 `terminal-session.ts` 的 `TerminalSessionRegistry` 中缓存最近一次 `fit()` 成功后的 `(cols, rows)`，并提供读取接口（如 `lastDimensions()`）
- [x] 1.2 在 `App.tsx:createTerminal` 中以缓存尺寸替换硬编码的 `100×32`：有缓存则用缓存，无缓存（首个终端）用内容区容器 DOM + xterm 字体度量估算
- [x] 1.3 处理首个终端边界：估算不可得时退化为默认值，并确保挂载 `fit()` 后立即同步（仅每会话首次、且无并发输出）
- [x] 1.4 校验 `CreateTerminal(taskID, columns, rows)` 绑定按真实尺寸传入，后端 `Manager.Create` 无需改动

## 2. 前端：合并容器尺寸变化（D2 防抖）

- [x] 2.1 新增 trailing 防抖工具（间隔作为常量，建议 ~100ms，便于按手感调整）
- [x] 2.2 在 `TerminalView.tsx` 用防抖包裹 `ResizeObserver` 的 `fitAndRefresh`，网格与 PTY 同步一起防抖
- [x] 2.3 在 `terminal-session.ts` 的 `fit()` 中去重：仅当 `cols/rows` 实际变化时才调用 `onResize`（PTY 同步）
- [x] 2.4 确保挂载与切换终端路径（`attach → fit → refresh`）保持立即执行，不被防抖延迟（仅持续 ResizeObserver 流被防抖）

## 3. 收紧单次 fit 时序（D3）

- [x] 3.1 核对 `fit()` 内“先 `fitAddon.fit()`、再同步读 `terminal.cols/rows` 调 `onResize`”的顺序保持紧凑
- [x] 3.2 确认防抖后单次稳定 Resize 的 `fit + onResize + refreshVisibleRows` 仍一次性完成

## 4. 测试（遵循前端 jsdom 测试纪律）

- [x] 4.1 为 `TerminalSessionRegistry.lastDimensions()` 与缓存更新补充单测
- [x] 4.2 测试 `createTerminal` 在有/无缓存时传入的尺寸正确（不再为 `100×32`）
- [x] 4.3 测试 `ResizeObserver` 连续回调被合并为最终稳定尺寸，PTY 同步以稳定值调用、而非逐帧
- [x] 4.4 测试相同 `cols/rows` 不触发重复 PTY 同步（去重）
- [x] 4.5 测试挂载/切换终端仍立即 `fit + refresh`，未被防抖延迟（防回归 `terminal-switch-rendering`）

## 5. Windows 实机验证与收尾

- [ ] 5.1 在 Windows 显示缩放（125%/150%/200%）下验证：拖拽窗口/分隔条期间及新建终端时不再出现折行/光标错位
- [ ] 5.2 校准防抖间隔常量至手感可接受
- [x] 5.3 复核既有切换渲染与输出保留行为未回归（前端测试套件 183/183 通过，含切换复用/输出保留用例）
- [ ] 5.4 可选硬化：在 `Manager.Resize` 增加相同尺寸去重作为纵深防御（见 design Open Questions）

## 6. 网格/PTY 原子性：PTY 同步期间缓冲输出（D4 —— 真正的根因修复）

> 触发：D1+D2 落地后用户实机仍报告“放大窗口时附近几行错位且自愈”。证据确认根因是同步网格与异步 PTY 之间的分歧窗口内输出被错位渲染，而非 DPI 抖动。防抖只降频未消除该窗口。

- [x] 6.1 拆分 `fit()` 职责：仅同步适配网格 + 去重，返回待同步 `{columns, rows}`（或 `undefined`）；不再在其中直接调 `onResize`
- [x] 6.2 新增 `syncPty(session, dims, onResize)`：置 `resizeInFlight` → `await onResize` → `finally` 清标志 + `flushOutput`
- [x] 6.3 `handleTerminalEvent` 在 `resizeInFlight` 期间把输出压入 `session.outputQueue`，否则直接 `terminal.write`
- [x] 6.4 `fitAndRefresh`/`attach` 保持同步 `fit()` + `refreshVisibleRows()`（网格立即可见），再 `void syncPty(...)` 异步对齐 PTY
- [x] 6.5 `App.tsx` 的 `onResize` 由 `void api.resizeTerminal(...)` 改为返回该 Promise（`.catch` 兜底），使 `syncPty` 可 await 到 ConPTY 生效
- [x] 6.6 新增回归测试：PTY 同步挂起期间到达的输出先缓冲、不立即写入，同步完成后按序写入（TDD：先红后绿）
- [x] 6.7 全量前端测试通过（183/183）、`tsc --noEmit` 通过、Windows 构建产物生成

