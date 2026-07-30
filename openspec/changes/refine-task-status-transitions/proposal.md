## Why

任务从“未执行”开始后会自动切换到“执行中”标签，但用户难以在列表中立即定位刚迁入的任务。相反，任务结束后自动跳转到“已完成”会打断用户继续处理其余执行中任务的上下文。

## What Changes

- 保留任务从“未执行”转为“执行中”后的自动标签切换，并为刚开始的任务提供一次快速、可辨识的视觉提示。
- 任务从“执行中”转为“已完成”后仅刷新任务数据和计数，保持用户当前选中的标签不变。
- 为动效触发、减少动态效果偏好及标签持久化行为补充前端测试。
- 明确与 `improve-lifecycle-command-progress` 的衔接：该变更中的标签同步规则以本变更为准。

## Capabilities

### New Capabilities

- `task-status-transition-feedback`: 定义任务跨状态标签时的视图切换和一次性视觉反馈行为。

### Modified Capabilities

<!-- 无；当前基线规格中尚未沉淀任务状态标签切换能力。 -->

## Impact

- 前端涉及 `frontend/src/App.tsx` 的任务开始、结束和生命周期事件处理，以及 `frontend/src/components/TaskTree.tsx` 的任务条目呈现。
- 不修改任务状态接口、持久化模型或后端生命周期语义；仅保留开始时既有的当前标签持久化，并移除结束时的自动标签持久化。
- 需要与活动变更 `improve-lifecycle-command-progress` 共同实现或按其实现结果调整，避免生命周期事件重新引入结束后的自动跳转。
