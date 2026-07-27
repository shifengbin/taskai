# 终端右键粘贴设计

## 目标

用户在嵌入式终端内容区域按鼠标右键时，直接将系统剪贴板中的非空文本发送到该终端。

## 方案

在 `TerminalView` 的终端内容容器上处理 `contextmenu` 事件：

- 调用 `preventDefault()`，不显示浏览器或 xterm 默认右键菜单。
- 调用 Wails Runtime 的 `ClipboardGetText()` 读取系统剪贴板。
- 读取到非空文本时，复用现有 `onWrite` 回调写入 PTY。
- 剪贴板为空或读取失败时静默忽略，保持终端交互不中断。

## 取舍

- 直接右键粘贴比自定义右键菜单少一次操作。
- 使用 Wails Runtime 而非浏览器 Clipboard API，避免桌面环境中的权限差异。
- 不改变键盘粘贴逻辑，也不影响已有的选区自动复制。

## 验证

在 `TerminalView` 测试中模拟 `ClipboardGetText`，验证右键事件被阻止、非空剪贴板文本会调用 `onWrite`，空内容不会调用。
