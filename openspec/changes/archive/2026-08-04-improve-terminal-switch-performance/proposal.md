## Why

活动终端运行一段时间并产生大量输出后，切换终端会明显卡顿。前端目前无上限地累积原始输出，并在每次切换时销毁 xterm、重新解析全部历史；处理量随历史内容增长，而用户只需要保留最近 1000 行滚屏。

## What Changes

- 为每个活动终端保留可复用的 xterm 会话状态，并将其滚屏上限固定为 1000 行。
- 将终端输出直接增量写入对应会话，不再在 React 状态中无上限累积或在渲染时比较完整输出字符串。
- 切换终端时复用已有 xterm 状态并恢复到当前可见容器，而非回放全部原始输出；继续执行尺寸适配、可见行刷新和自动聚焦。
- 在终端关闭、任务结束和应用卸载时释放对应会话及其监听器，避免缓存泄漏。
- 补充大输出量、多终端切换、1000 行滚屏边界及资源清理的前端回归测试。

## Capabilities

### New Capabilities

- `terminal-output-retention`: 将每个活动终端的输出状态限制为 xterm 最近 1000 行滚屏，并在终端生命周期结束时释放状态。

### Modified Capabilities

- `terminal-switch-rendering`: 终端切换从重新回放完整历史输出改为复用已有终端状态，同时保持完整可见区域绘制和交互连续性。

## Impact

- 影响前端终端会话所有权、事件分发及 `TerminalView` 挂载生命周期。
- 影响 `frontend/src/App.tsx`、`frontend/src/state.ts`、`frontend/src/components/TerminalView.tsx` 及其测试。
- 不改变 Go 终端管理器、Wails 事件协议、PTY 协议、持久化数据或 HTTP API。
