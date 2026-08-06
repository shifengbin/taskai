## Why

任务列表中的每个任务项当前默认带有偏移硬阴影，多个条目连续排列时会产生过重的层叠感，降低列表扫描时的清晰度。保留悬浮阴影可以继续提供明确的交互反馈，因此只调整默认展示状态。

## What Changes

- 移除未选中任务项默认状态的外部硬阴影。
- 保留任务色内嵌色条、边框、背景、尺寸和其他布局样式。
- 保留任务项鼠标悬浮时的上移和外部阴影反馈。
- 保留选中任务项的钴蓝选择阴影，以及选中任务悬浮时的增强阴影。
- 更新任务树样式测试和快门波普视觉规格，明确默认态与悬浮态的差异。

## Capabilities

### New Capabilities


### Modified Capabilities

- `pine-night-run-visual-system`: 调整任务树任务项默认态的阴影要求，保留悬浮和选中状态的阴影反馈。

## Impact

- 前端样式：`frontend/src/App.css` 中的 `.taskai-task-row` 及其状态规则。
- 前端测试：`frontend/src/components/TaskTree.test.tsx` 中任务项视觉断言。
- OpenSpec 视觉规格：更新任务项默认态与悬浮态的行为描述。
- 不涉及 Go 后端、API、持久化数据或运行时依赖。
