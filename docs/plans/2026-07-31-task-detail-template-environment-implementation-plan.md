# 任务详情模板环境变量实施计划

> **执行说明：** 按顺序实施并在每个步骤后运行对应验证。

**目标：** 在任务详情中展示当前启用模板、字段值和模板字段是否生成生命周期环境变量。

**实现方式：** 将当前启用模板传入 `TaskDetail`，复用前端已有的字段值解析逻辑，按模板字段顺序渲染字段值及环境变量状态。只在 `injectEnvironment` 为真时展示 `TASKAI_<字段键大写>=<当前值>`，并标明仅传入自定义生命周期 Shell 命令。

**技术栈：** React、TypeScript、Material UI、Vitest。

---

### 任务 1：补充失败的详情页交互测试

**文件：**

- 修改：`frontend/src/App.test.tsx`

1. 构造当前启用模板，包含字符串、布尔和未注入字段，并设置任务的模板字段值。
2. 断言详情显示模板名称、字段名称和值。
3. 断言注入字段显示 `TASKAI_<字段键大写>=<值>` 与生命周期 Shell 作用范围；未注入字段显示“不生成环境变量”。
4. 使用 `npm test -- App.test.tsx -t '任务详情展示模板字段和环境变量'` 运行测试，确认新增断言在实现前失败。

### 任务 2：渲染模板详情和环境变量状态

**文件：**

- 修改：`frontend/src/App.tsx`

1. 将当前启用模板传入 `TaskDetail`。
2. 使用 `resolveTaskTemplateValues` 取得字段的当前解析值。
3. 在额外信息和系统环境变量区之间渲染“任务模板”区；未启用模板时显示空状态。
4. 为每个字段显示字段键、当前值和环境变量状态；布尔值统一为 `true` 或 `false`。
5. 再次运行任务 1 的测试，确认通过。

### 任务 3：更新 OpenSpec 与完整验证

**文件：**

- 修改：`openspec/changes/enhance-task-detail-command-chain-help/design.md`
- 修改：`openspec/changes/enhance-task-detail-command-chain-help/specs/task-detail-command-context/spec.md`
- 修改：`openspec/changes/enhance-task-detail-command-chain-help/tasks.md`

1. 增加模板展示与模板环境变量范围的设计、需求场景和任务项。
2. 运行 `npm test`、`go test ./...` 和 `scripts/build-linux.sh`。
3. 运行 `openspec validate enhance-task-detail-command-chain-help --strict`。

本计划不会自动创建 Git 提交；完成后保留当前 worktree 供人工审阅。
