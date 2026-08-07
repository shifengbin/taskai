## Why

PTY 的单次读取可以在 UTF-8 多字节字符中间结束。当前终端管理器把每个读取片段立即作为字符串事件发送，导致跨片段的中文或 emoji 在 Wails 的 JSON 序列化中变成替换字符，用户偶尔会看到少量菱形问号。

## What Changes

- 在每个活动终端会话中保留尚未构成完整 UTF-8 字符的输出尾字节。
- 将后续 PTY 输出与该尾字节拼接后再发布，保证发送给前端的普通终端输出只包含完整的 UTF-8 字符，且顺序不变。
- 为中文和 emoji 跨读取边界的情况添加回归测试，并覆盖终端退出时残留字节的处理。
- 保持终端事件名称、前端事件数据类型、终端输入、状态判定和 Unix/Windows PTY 实现不变。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `terminal-output-retention`: 终端的增量输出在发送到前端前必须保持完整的 UTF-8 字符边界。

## Impact

- 受影响代码：`internal/terminal/manager.go` 及其测试。
- 受影响系统：Unix PTY 和 Windows ConPTY 共用的终端输出读取循环。
- 不改变 Wails 绑定、前端 `TerminalEvent` 契约或第三方依赖。
