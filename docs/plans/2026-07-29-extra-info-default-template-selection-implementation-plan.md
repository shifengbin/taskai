# 额外信息默认模板选择实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 让分类模板默认收起且可直接创建，并让新增信息在单模板或当前会话上次成功新增模板存在时自动进入对应模板。

**架构：** 在 `App` 中为模板折叠状态和最后成功新增信息的模板 ID 建立仅内存状态。`openExtraInfoEditor` 通过统一的模板解析逻辑创建新增草稿；编辑已有信息、取消编辑和仅切换下拉框不更新该状态。分类模板折叠面板与新增按钮使用同级布局，使按钮不依赖折叠内容的可见性。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library、Go、Wails。

---

### 任务 1：新增默认模板选择与折叠按钮的失败测试

**文件：**
- 修改：`frontend/src/App.test.tsx`
- 参考：`frontend/src/App.tsx` 中 `openExtraInfoEditor`、`saveExtraInfo` 和额外信息管理 `Dialog`

**步骤 1：编写失败测试**

在既有额外信息测试附近增加交互测试：

```tsx
expect(screen.getByRole('button', {name: '分类模板'})).toHaveAttribute('aria-expanded', 'false')
await user.click(screen.getByRole('button', {name: '新增模板'}))
expect(screen.getByRole('dialog', {name: '新增模板'})).toBeInTheDocument()

await user.click(screen.getByRole('button', {name: '新增信息'}))
expect(screen.queryByRole('combobox', {name: '选择模板'})).not.toBeInTheDocument()
expect(screen.getByRole('textbox', {name: '项目名称'})).toBeInTheDocument()
```

为多模板场景模拟一次成功新增信息：首次选择 Git 模板、填写名称并保存；再次点击新增信息时断言直接显示 Git 固定字段且不再显示模板选择框。额外覆盖上次模板删除后回退为手动选择。

**步骤 2：运行测试并确认失败**

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：失败，分类模板当前为展开状态，或新增信息仍显示模板选择框。

### 任务 2：实现模板区域布局和会话内默认选择

**文件：**
- 修改：`frontend/src/App.tsx`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：实现最小状态与解析逻辑**

1. 添加 `templateSectionExpanded`，初始值设为 `false`，将分类模板 `Accordion` 改为受控展开状态。
2. 添加 `lastCreatedExtraInfoTemplateID`，初始值为空字符串；仅在 `saveExtraInfo` 成功保存新信息后写入 `saved.templateId`。
3. 在 `openExtraInfoEditor` 的新增分支按以下顺序解析模板：唯一模板、仍存在的最后新增模板、无默认模板。前两种直接调用现有草稿创建逻辑，第三种保留模板选择框。
4. 编辑已有信息、`closeExtraInfoEditor`、模板下拉框切换不修改最后新增模板状态；删除模板后不主动写状态，解析时通过当前模板列表验证其有效性。

**步骤 2：实现模板区域布局**

将 `Accordion` 和“新增模板”按钮放入同级的自适应横向容器。按钮保留 `contained` 和 `small` 尺寸，并不再放在 `AccordionDetails` 内部；分类模板收起时仍可点击创建。

示意：

```tsx
<Box sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', gap: 1}}>
  <Accordion expanded={templateSectionExpanded} onChange={(_, next) => setTemplateSectionExpanded(next)}>
    {/* 分类模板列表 */}
  </Accordion>
  <Button size="small" variant="contained">新增模板</Button>
</Box>
```

**步骤 3：运行目标测试并确认通过**

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：默认收起、折叠外新增模板、单模板预选、上次成功新增模板预选与删除后回退测试均通过，既有额外信息测试继续通过。

**步骤 4：提交实现**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: 优化额外信息默认模板选择"
```

### 任务 3：完整验证

**文件：**
- 验证：`frontend/src/App.test.tsx`、前端测试套件、Go 代码和 Linux 构建产物

**步骤 1：运行质量检查和前端全量测试**

```bash
git diff --check
cd frontend && npm test -- --run
```

预期：无空白错误，前端测试全部通过。

**步骤 2：运行 Go 测试和 Linux 构建**

```bash
go test ./... -count=1
./scripts/build-linux.sh
```

预期：Go 测试与 Linux 构建均通过，并生成 `build/bin/taskai`。

**步骤 3：检查生成文件和工作区**

```bash
git status --short
git diff --check
```

预期：仅保留预期改动；构建器产生的纯格式生成文件改动应在不影响内容的前提下清理。
