# 任务标签记忆与排序实施计划

> **供 Codex 使用：** 按 TDD 顺序逐项实施并验证。

**目标：** 持久化任务状态标签选择，并支持任务双击展开终端和同状态内的拖动排序。

**架构：** 当前标签保存于 `Settings`，任务排序复用 JSON 中既有任务数组；后端校验并重排相同状态的任务，前端任务树使用原生拖放调用新绑定。

**技术栈：** Go、Wails、React、Material UI、Vitest、Go `testing`。

---

### 任务 1：持久化当前标签

**文件：**
- 修改：`internal/settings/settings.go`
- 修改：`internal/settings/settings_test.go`
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 编写失败测试，验证缺失标签时回退“未执行”，以及用户切换标签后保存完整设置。
2. 为设置增加并校验 `activeTaskStatus`，加载设置后恢复标签。
3. 让手动切换和开始、结束任务都经过同一个标签持久化函数。
4. 运行对应 Go 与前端测试，确认通过。

### 任务 2：持久化同状态任务顺序

**文件：**
- 修改：`internal/application/contracts.go`
- 修改：`internal/lifecycle/service.go`
- 修改：`internal/lifecycle/service_test.go`
- 修改：`app.go`
- 修改：`frontend/src/api.ts`

**步骤：**
1. 编写失败测试，验证同状态任务按指定 ID 顺序持久化且其他状态任务不变。
2. 增加重排绑定，并拒绝缺失、重复或跨状态 ID 列表。
3. 运行生命周期服务与应用测试，确认通过。

### 任务 3：实现任务树双击与拖动

**文件：**
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/components/TaskTree.test.tsx`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 编写失败测试，验证双击任务切换终端子节点，以及拖放任务触发排序回调。
2. 为任务条目加入非按钮区域的双击展开和原生拖放反馈。
3. 应用层调用重排接口、更新返回列表，并在失败时重新加载列表。
4. 运行前端全量测试，确认通过。

### 任务 4：规格与交付验证

**文件：**
- 修改：`openspec/changes/add-task-terminal-workspace/specs/task-tree-interface/spec.md`
- 修改：`openspec/changes/add-task-terminal-workspace/specs/workspace-settings/spec.md`
- 修改：`openspec/changes/add-task-terminal-workspace/tasks.md`
- 修改：`README.md`

**步骤：**
1. 用中文补充标签恢复、双击展开和同状态拖动排序场景。
2. 运行 `go test -race ./...`、`cd frontend && npm test -- --run`、`npm run build` 与 `openspec validate add-task-terminal-workspace --strict`。
3. 运行 `./scripts/build-linux.sh` 并确认生成 `build/bin/taskai`。
