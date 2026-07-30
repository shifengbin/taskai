# 生命周期命令适用范围实施计划

> **For Codex:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**目标：** 为生命周期命令和命令链增加钩子适用范围，并限制编辑任务不能修改链选择。

**架构：** 在 `settings` 领域模型中保存命令/链的 `ApplicableHooks`，由统一规范化函数完成旧数据迁移、内置命令固定范围、链内命令覆盖校验和默认链校验。仓库与应用层复用规范化结果，前端只负责按范围筛选和只读呈现，不能绕过后端约束。

**技术栈：** Go、Wails、React、TypeScript、Vitest、OpenSpec。

---

### 任务 1：建立领域范围约束与迁移测试

**文件：**
- 修改：`internal/settings/settings_test.go`
- 修改：`internal/settings/settings.go`
- 修改：`internal/storage/repository_test.go`

**步骤：**
1. 先在 `settings_test.go` 添加失败用例：自定义命令/链缺少范围被拒绝；内置命令与初始链分别固定为 `beforeStart`/`postEnd`；链引用未覆盖所有链范围的命令被拒绝；默认链不覆盖钩子时被拒绝；无范围的旧 JSON 迁移为五钩子范围。
2. 运行 `go test ./internal/settings ./internal/storage`，确认新断言因字段与校验尚不存在而失败。
3. 在 `settings.go` 为 `LifecycleCommand`、`LifecycleCommandChain` 添加 `ApplicableHooks []LifecycleHook` JSON 字段；实现去重、合法性和非空校验；固定系统命令/默认链范围；通过命令 ID 查找验证链内每个命令覆盖链的全部范围；默认值只接受覆盖其键钩子的链。
4. 让兼容旧 JSON 的空范围在加载规范化时补为全部五钩子，而新建/保存的对象必须显式提交至少一个范围。必要时以内部迁移标记或设置加载路径区分旧快照和新的无效输入。
5. 重跑 `go test ./internal/settings ./internal/storage`，确认绿色。

### 任务 2：在仓库、服务与绑定层阻止无效选择和编辑改链

**文件：**
- 修改：`internal/storage/repository.go`
- 修改：`internal/storage/repository_test.go`
- 修改：`internal/lifecycle/service.go`
- 修改：`internal/lifecycle/service_test.go`
- 修改：`app.go`
- 修改：`app_test.go`

**步骤：**
1. 添加失败测试：保存范围不匹配链失败；创建任务选择未覆盖对应钩子的链失败；旧任务选择在规范化后只保留适用链；编辑任务无法改变既有 `LifecycleChains`。
2. 运行 `go test ./internal/storage ./internal/lifecycle .`，确认失败原因正确。
3. 更新链复制逻辑深复制 `ApplicableHooks`；在 `validateLifecycleChainSelections` 中要求所选链覆盖映射键；更新任务选择规范化以移除或拒绝不适用的引用。
4. 删除 `UpdateTaskWithExtraInfoAndLifecycleChains` 从 `lifecycle.Service` 与 `App` 的公开绑定，前端编辑路径只调用既有不含链参数的更新方法；保留任务创建的带链入口。
5. 重跑上述 Go 测试，确认范围校验、迁移和编辑保护均通过。

### 任务 3：更新前端类型、范围管理与任务表单测试

**文件：**
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/App.test.tsx`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/api.ts`

**步骤：**
1. 先在 `App.test.tsx` 添加失败交互：命令编辑能选择范围；链编辑仅显示覆盖当前链范围的命令；默认链下拉和新建任务下拉只出现当前钩子可用链；编辑任务链选择器禁用，保存不调用带链更新绑定。
2. 运行 `npm test -- --run src/App.test.tsx`，确认新断言失败。
3. 在 TypeScript 类型中补充 `applicableHooks`；提供基于 `lifecycleHooks` 的范围标签/覆盖辅助函数。命令和链编辑弹窗增加复选框；链范围变动后不自动删除命令，而是显示不兼容提示并禁用保存。系统命令范围以只读标签展示。
4. 按钩子筛选默认链和新建任务链选择；编辑任务保留链选择但将选择器设为禁用；移除 API 封装和 mock 中的更新链绑定调用。
5. 重跑 `npm test -- --run src/App.test.tsx`，确认绿色。

### 任务 4：同步 OpenSpec、README、Wails 生成代码并完成验证

**文件：**
- 修改：`openspec/changes/add-task-lifecycle-command-chains/{proposal.md,design.md,tasks.md}`
- 修改：`openspec/changes/add-task-lifecycle-command-chains/specs/task-lifecycle-command-chains/spec.md`
- 修改：`README.md`
- 自动生成：`frontend/wailsjs/go/main/{App.js,App.d.ts}`、`frontend/wailsjs/go/models.ts`

**步骤：**
1. 将范围、内置命令固定钩子、链覆盖约束、筛选与编辑任务只读规则写入中文 OpenSpec、README，并新增未勾选的增量任务。
2. 运行 `openspec validate add-task-lifecycle-command-chains --strict`，确认规格先合法。
3. 运行 `go test -race ./...` 与 `npm test`。
4. 运行 `./scripts/build-linux.sh` 生成 Wails 绑定并完成规定 Linux 构建。
5. 复跑 `openspec validate add-task-lifecycle-command-chains --strict` 和 `git diff --check`（排除生成器固有空白）；勾选增量任务。
