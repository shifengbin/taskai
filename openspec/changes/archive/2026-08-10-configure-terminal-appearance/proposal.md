## Why

当前内嵌终端的整套配色跟随工作台亮暗模式，用户无法按个人偏好设置终端，也会在亮色工作台中得到与终端使用习惯不一致的浅色终端。终端字体变更还只会影响新建会话，已有会话无法在保存设置后立即采用新字体。

## What Changes

- 新增完整的持久化终端外观配置，包含基础颜色、常规 ANSI 八色、高亮 ANSI 八色、终端字体和字号。
- 在设置页提供仅通过可视化颜色选择器操作的终端配色区域，选区背景额外提供透明度滑块、草稿预览和一键恢复当前暗色终端默认方案。
- 终端外观不再跟随工作台亮暗模式；默认配色逐项保持当前暗色终端的外观。
- 所有终端外观变更必须点击“保存”才生效；保存后立即更新已有终端，字体和字号变化会重新适配可见终端并同步 PTY 尺寸。

## Capabilities

### New Capabilities

- `terminal-appearance-settings`: 定义完整终端主题的持久化、可视化设置、默认重置和对现有会话的保存后即时应用行为。

### Modified Capabilities

- `terminal-font-selection`: 已保存字体从仅影响新建会话改为在保存后同时应用到已有会话。
- `pine-night-run-visual-system`: 终端配色从随工作台亮暗模式变化改为使用独立终端外观设置。

## Impact

- 后端：`internal/settings` 的设置模型、默认值和归一化逻辑，以及 `internal/storage` 的旧数据迁移。
- 前端：Wails 生成的设置模型、`App` 设置草稿、`TerminalSessionRegistry`、`TerminalView` 和终端设置界面。
- 验证：Go 设置/存储测试、终端会话与终端视图测试、设置对话框测试，以及 OpenSpec 规格。
