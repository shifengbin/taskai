## Why

当前快门波普视觉系统的硬投影在任务项、按钮和覆盖层上过于突出，影响界面的信息密度与视觉轻量感。保持粗描边和交互抬起反馈的同时，需收敛投影偏移以获得更克制的层级。

## What Changes

- 将 Snap 的小、中、大三档硬投影偏移统一缩小 1px。
- 保持硬投影颜色、零模糊特性、描边、圆角和悬停抬起行为不变。
- 使任务项、按钮、图标按钮、表单控件及弹出层自动采用更小的投影，无需逐项覆盖样式。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `pine-night-run-visual-system`: 收紧工作台与控件的快门波普硬投影尺寸，同时保留一致的视觉语言和交互反馈。

## Impact

- 影响 `frontend/tailwind.config.cjs` 中的 `shadow-snap-sm`、`shadow-snap` 与 `shadow-snap-lg` 令牌。
- 所有引用这些令牌的前端组件将自动使用更小的投影；不涉及 API、数据、依赖或业务逻辑变更。
