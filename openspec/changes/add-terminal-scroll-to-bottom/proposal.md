## Why

用户向上滚动查看终端历史后，当前界面缺少明确且快速的方式回到最新输出；当终端持续产生内容时，用户需要手动拖动滚动条或连续滚动，操作成本较高。

## What Changes

- 当当前终端视口不在滚屏底部时，在终端内容区显示“回到底部”浮动按钮。
- 用户点击按钮后，将当前 xterm 会话滚动到最新输出并恢复终端输入焦点。
- 用户通过滚轮、触控板、滚动条或程序化滚动回到底部后，自动隐藏按钮。
- 切换终端、调整终端尺寸或查看异常退出的只读快照时，根据对应会话的实际滚动位置同步按钮状态。
- 将滚动位置监听与终端实时状态的画面变化判定隔离，避免用户查看历史影响任务状态上报。

## Capabilities

### New Capabilities

- `terminal-scroll-to-bottom`: 定义终端离开底部时的提示、回到底部操作、会话切换行为和监听生命周期。

### Modified Capabilities

无。

## Impact

- 前端终端视图：`frontend/src/components/TerminalView.tsx` 增加本地可见状态和浮动按钮。
- 前端终端会话：`frontend/src/terminal-session.ts` 封装 xterm 滚动位置监听与 `scrollToBottom()` 调用。
- 测试：扩展 `frontend/src/components/TerminalView.test.tsx` 和 `frontend/src/terminal-session.test.ts` 的 xterm 测试桩与行为断言。
- 不修改 Go/Wails API、PTY 会话协议、终端输出保留上限、持久化格式或第三方依赖。
