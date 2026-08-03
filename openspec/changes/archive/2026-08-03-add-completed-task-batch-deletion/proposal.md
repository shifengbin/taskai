## Why

已完成任务会持续保留在任务列表中，用户只能逐项查看，无法高效清理不再需要的历史记录。需要提供受控的批量删除入口，同时确保这项记录管理操作不会意外影响任务工作目录或生命周期自动化。

## What Changes

- 在“已完成”任务标签中增加选择模式，允许逐项勾选、全选当前可删除任务以及退出选择。
- 为已选任务提供批量删除操作和不可撤销的二次确认；确认后仅从任务数据中移除这些记录。
- 新增后端批量删除接口，在一次持久化操作中校验并删除任务；仅允许删除状态为已完成、且没有生命周期命令链正在执行或失败待重试的任务。
- 删除成功后清理前端中对应的任务选择、展开和选择模式状态，并刷新任务计数。
- 明确批量删除不删除工作目录、不执行或重试生命周期命令，也不影响未执行或执行中的任务。

## Capabilities

### New Capabilities

- `completed-task-batch-deletion`: 在已完成任务中进行受控选择、确认和原子化批量删除记录的能力。

### Modified Capabilities

<!-- 无。 -->

## Impact

- 前端涉及 `frontend/src/App.tsx`、`frontend/src/components/TaskTree.tsx`、`frontend/src/api.ts` 及其测试，以管理选择模式、确认对话框和本地状态清理。
- Wails 绑定、`app.go`、`internal/application/contracts.go` 与 `internal/lifecycle/service.go` 需要新增批量删除已完成任务记录的接口和校验。
- 需要更新 Wails 生成的前端绑定，并补充 Go 服务、应用绑定和前端组件/应用测试。
