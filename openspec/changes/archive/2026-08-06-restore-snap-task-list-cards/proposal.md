## Why

任务列表当前只保留了任务色左边框和较小的硬投影，未能体现 Snap - 快门波普参考中的完整卡片层级；终端子项也因此显得偏弱。此前收紧全局阴影令牌的方向进一步拉大了这一差距。

## What Changes

- 恢复 Snap 共享硬投影令牌的原始 2px、3px、4px 偏移层级。
- 将顶层任务项渲染为 Snap 参考中的完整硬描边卡片，保留用户定义任务颜色的低透明背景与颜色标识。
- 为任务项提供可见的钴蓝选中态、1px 悬停抬起和更深的硬投影，不新增状态尾标。
- 将任务下的终端子项对齐为具有次级表面、硬描边、硬投影与相同悬停反馈的紧凑卡片。
- 将顶层任务卡片固定为两行 60px 高度，始终显示标题与描述；缺失描述时使用“暂无描述”。
- 保留既有悬停显示操作、拖拽、生命周期进度、键盘操作和减少动态效果行为；开始任务反馈不得覆盖卡片的常态硬投影。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `pine-night-run-visual-system`: 恢复 Snap 的硬投影层级，并明确任务项和终端子项的卡片、选择与交互反馈要求。

## Impact

- 修改 `frontend/tailwind.config.cjs`、`frontend/src/App.css`、`frontend/src/style.css` 和 `frontend/src/components/TaskTree.tsx`。
- 扩展 `TaskTree`、样式令牌相关的前端测试。
- 不涉及后端接口、任务数据、终端会话或依赖变更。
