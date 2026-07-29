# 任务额外信息仅使用既有信息实施计划

> **供 Codex 执行：** 必须使用 `superpowers:executing-plans`，按任务逐项实施。

**目标：** 让任务只能填写所选信息带入的参数值，并在额外信息管理中隐藏实际空分类。

**架构：** 移除 `TaskExtraInfoEditor` 对任务级参数的新增和删除状态操作，将 `TaskExtraInfoSnapshotFields` 收敛为值编辑器。额外信息管理渲染分类前按原始 `templateInfos` 过滤，不改变搜索筛选逻辑。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library。

---

### 任务 1：添加失败回归测试

**文件：**
- 修改：`frontend/src/App.test.tsx`

**步骤 1：更新任务参数测试**

选择带参数的信息后，断言可填写参数值，且没有“新增动态参数”、参数键、显示名称、参数类型和删除动态参数控件。保存时断言请求只包含该信息带入的参数。

**步骤 2：添加空分类过滤测试**

配置两个模板但只给其中一个模板创建信息。打开额外信息管理后断言有信息的分类可见，空模板的信息分类按钮不存在；在非空分类搜索无结果时断言分类仍可见。

**步骤 3：确认测试失败**

```bash
cd frontend && npm test -- --run src/App.test.tsx -t '创建任务时按名称搜索信息|隐藏没有信息的分类'
```

预期：失败，因为任务编辑器仍可新增参数，空模板分类仍被渲染。

### 任务 2：收紧任务参数编辑器

**文件：**
- 修改：`frontend/src/App.tsx:1520-1635`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：移除任务级参数操作函数和属性**

删除 `addParameter`、`removeParameter` 及其组件传参。更新 `TaskExtraInfoSnapshotFields` 签名，只保留按参数索引更新值的回调。

**步骤 2：仅渲染参数值控件**

移除参数键、显示名称和类型输入以及删除按钮。文本和复选框的值更新仍调用 `onChange`；去掉“新增动态参数”按钮，并更新辅助文案。

**步骤 3：确认任务回归测试通过**

```bash
cd frontend && npm test -- --run src/App.test.tsx -t '创建任务时按名称搜索信息'
```

预期：通过。

### 任务 3：过滤实际空分类

**文件：**
- 修改：`frontend/src/App.tsx:935-955`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：过滤分类**

在 `extraInfoTemplates.map` 前或映射中，仅为 `templateInfos.length > 0` 的模板返回分类组件。保留 `infos` 的搜索筛选和空搜索结果文本。

**步骤 2：确认分类测试通过**

```bash
cd frontend && npm test -- --run src/App.test.tsx -t '隐藏没有信息的分类'
```

预期：通过。

### 任务 4：完整验证并提交

**文件：**
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤 1：运行前端全量测试**

```bash
cd frontend && npm test -- --run
```

**步骤 2：运行 Go 全量测试**

```bash
go test ./... -count=1
```

**步骤 3：构建 Linux 应用**

```bash
./scripts/build-linux.sh
```

**步骤 4：检查并提交**

```bash
git diff --check
git add docs/plans/2026-07-29-task-extra-info-usage-only-design.md docs/plans/2026-07-29-task-extra-info-usage-only-implementation-plan.md frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "fix: 限制任务额外信息参数编辑"
```
