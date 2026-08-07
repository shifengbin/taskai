## Why

右侧终端标题栏与左侧终端项重复提供状态与关闭操作，压缩标题可用空间，也让同一终端在两个位置承担相同的管理职责。右侧应聚焦当前终端的快捷输入，状态查看和关闭操作统一保留在左侧列表。

## What Changes

- 移除右侧活动终端标题栏的实时状态点和“关闭终端”按钮。
- 保留右侧标题栏的终端图标、可裁剪标题和快捷输入入口；快捷输入入口在活动终端标题栏中始终显示，不依赖鼠标悬浮或键盘焦点。
- 保持左侧终端项的实时状态点和关闭入口，以及其现有的悬浮、焦点和无悬停设备显示规则。
- 保持终端关闭行为、快捷输入快捷键、会话生命周期、无障碍标签和所有后端接口不变。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `pine-night-run-visual-system`: 调整右侧终端标题栏的操作与状态可见性，使其不再重复左侧终端项的状态和关闭功能。

## Impact

- 受影响前端组件：`frontend/src/components/TerminalView.tsx` 和作用域化样式。
- 受影响前端测试：`frontend/src/components/TerminalView.test.tsx`。
- 不涉及 Go 后端、Wails 绑定、PTY 会话、持久化数据、API 或第三方依赖。
