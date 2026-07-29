# 新建信息的模板参数只读预览实施计划

> **供 Codex 执行：** 必须使用 `superpowers:executing-plans`，按任务逐项实施。

**目标：** 在创建或编辑信息时只读展示当前模板的动态参数，同时保持信息级动态参数独立可编辑。

**架构：** 在 `App` 中根据 `extraInfoDraft.templateId` 从已加载模板列表解析模板参数，单独渲染只读预览区。预览不进入信息草稿，因此 `SaveExtraInfo` 与创建任务时的参数合并无需调整。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library。

---

### 任务 1：添加模板参数预览的失败测试

**文件：**
- 修改：`frontend/src/App.test.tsx`

**步骤 1：编写失败用例**

创建带文本参数与复选框参数的模板，打开“新增信息”并选择该模板。断言只读预览包含键、显示名称、默认值和文本参数的必填状态；断言没有对应的文本框、复选框或删除按钮。

**步骤 2：确认测试失败**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx -t '新建信息只读预览模板动态参数'
```

预期：失败，因为编辑器目前只渲染 `extraInfoDraft.parameters`，而模板参数不在草稿中。

### 任务 2：渲染只读模板参数预览

**文件：**
- 修改：`frontend/src/App.tsx:962-1000`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：解析当前模板**

在额外信息编辑器中依据 `extraInfoDraft.templateId` 查找 `extraInfoTemplates` 的模板；未选模板或模板缺失时不渲染预览。

**步骤 2：增加只读预览区**

在可编辑“信息参数”区域前渲染“模板参数”标题和每条参数的键、显示名称、默认值、文本参数必填状态。复选框参数显示 `true`/`false` 默认值，但不显示必填状态。预览仅使用 `Typography`、`Chip` 或只读容器，不引入编辑事件、删除按钮或草稿状态更新。

**步骤 3：保留信息级参数编辑行为**

保留“新增动态参数”按钮及其现有 `extraInfoDraft.parameters` 修改逻辑。保存信息时不将模板参数加入 `SaveExtraInfo` 的参数数组。

**步骤 4：确认测试通过**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx -t '新建信息只读预览模板动态参数'
```

预期：通过。

### 任务 3：执行完整验证并提交

**文件：**
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤 1：运行前端全量测试**

```bash
cd frontend && npm test -- --run
```

**步骤 2：运行 Go 测试**

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
git add docs/plans/2026-07-29-extra-info-template-parameter-preview-design.md docs/plans/2026-07-29-extra-info-template-parameter-preview-implementation-plan.md frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: 预览信息模板动态参数"
```
