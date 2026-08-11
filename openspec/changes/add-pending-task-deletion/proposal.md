## Why

未执行任务可能在开始前被取消或不再需要，但目前只能保留在列表中；用户无法以与已完成任务一致且受控的方式清理这些记录。需要将现有批量删除能力扩展到未执行任务，同时继续保护工作目录和生命周期执行状态。

## What Changes

- 在“未执行”任务标签提供与“已完成”任务相同的批量选择、全选、二次确认和删除记录操作。
- 批量删除接口允许删除未执行或已完成、且没有生命周期执行记录的任务；请求仍在全部校验通过后一次性持久化。
- 删除始终只移除任务记录，不删除、移动或修改任务工作目录，也不调度、取消或重试生命周期命令链。
- 删除成功后清理前端中已删除任务关联的详情、终端、展开与选择状态。

## Capabilities

### New Capabilities

<!-- 无。 -->

### Modified Capabilities

- `completed-task-batch-deletion`: 将批量删除的可用标签与后端删除资格从仅已完成任务扩展为未执行和已完成任务。

## Impact

- 前端涉及 `frontend/src/App.tsx`、`frontend/src/components/TaskTree.tsx` 与相关测试，以复用选择模式、确认交互和本地状态清理。
- 后端涉及 `app.go`、`internal/lifecycle/service.go` 及其测试，以扩展原子化删除的状态校验，并将 Wails 方法更名为通用的 `DeleteTasks`。
- 需要更新 `openspec/specs/completed-task-batch-deletion/spec.md`，使已发布规格与行为一致。
