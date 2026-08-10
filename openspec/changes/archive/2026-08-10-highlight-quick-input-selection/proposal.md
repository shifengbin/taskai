## Why

快捷输入选择器已支持用上下方向键切换结果，但键盘焦点保留在搜索框，当前选项只通过无障碍属性标记，界面没有相应的视觉反馈。用户无法确认当前选中了哪一项，容易在按 Enter 时插入错误内容。

## What Changes

- 为终端快捷输入选择器的当前键盘选中项提供持续可见的选中样式。
- 保持悬停和键盘选择同步，并使搜索、Enter 插入、Escape 关闭及焦点恢复行为不变。
- 为视觉选中状态补充前端测试。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `terminal-quick-inputs`: 明确方向键选中的结果必须有可区分的视觉样式。

## Impact

- 影响 `frontend/src/components/TerminalView.tsx` 中快捷输入选择器的结果项呈现。
- 影响 `frontend/src/components/TerminalView.test.tsx` 的键盘导航测试。
- 不改变快捷输入的持久化、终端写入、快捷键或任何后端接口。
