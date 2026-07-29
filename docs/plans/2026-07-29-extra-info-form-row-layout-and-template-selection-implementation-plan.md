# 额外信息表单行布局与模板选择实施计划

> **供 Codex 执行：** 按测试先行方式逐项完成本计划。

**目标：** 让模板字段及参数的删除操作保持同行、信息字段逐行展示，并让新建信息的默认模板始终可切换。

**架构：** 仅调整 `App` 内两个表单的局部布局与新增信息的模板选择渲染条件。默认模板解析和保存后的会话状态保持不变；通过 `newExtraInfoTemplateID` 保持下拉框值，切换时复用现有 `selectExtraInfoTemplate` 重新生成草稿。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library。

---

### 任务 1：为字段行布局和可切换默认模板编写失败测试

**文件：**
- 修改：`frontend/src/App.test.tsx`

**步骤 1：编写字段与参数删除按钮同行测试**

```tsx
const field = screen.getByTestId('extra-info-template-fixed-field-0')
expect(getComputedStyle(field).gridTemplateColumns).toBe('minmax(0, 1fr) auto')
const parameter = screen.getByTestId('extra-info-template-parameter-0')
expect(getComputedStyle(parameter).gridTemplateColumns).toBe('minmax(0, 1fr) auto')
```

**步骤 2：编写信息字段单列测试**

```tsx
const fields = screen.getByTestId('extra-info-draft-fields')
expect(getComputedStyle(fields).gridTemplateColumns).toBe('1fr')
```

**步骤 3：编写默认模板可切换测试**

使用多个模板成功保存一条信息后再次新增，断言“选择模板”下拉框仍存在且选中上次模板；选择另一模板后断言表单字段更新。

**步骤 4：运行测试确认失败**

运行：`cd frontend && npm test -- --run src/App.test.tsx -t '删除固定字段|固定字段逐行|默认模板仍可切换'`

预期：失败，现有删除按钮独占一行、字段容器为多列，默认草稿隐藏选择框。

### 任务 2：实现模板字段同行和信息字段单列布局

**文件：**
- 修改：`frontend/src/App.tsx:968-970`
- 修改：`frontend/src/App.tsx:1021-1030`

**步骤 1：将信息固定字段容器改为单列**

```tsx
<Box data-testid="extra-info-draft-fields" sx={{display: 'grid', gridTemplateColumns: '1fr', gap: 1.5, minWidth: 0}}>
```

**步骤 2：将模板固定字段和参数条目改为两列**

```tsx
<Box sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', alignItems: 'start', gap: 1.25, ...}}>
  <Box sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', ...}}>{/* 输入字段 */}</Box>
  <IconButton aria-label={`删除固定字段 ${index + 1}`} />
</Box>
```

保留删除禁用条件和输入字段的现有行为，参数的必填控制仍留在左侧内容列中。

### 任务 3：让新增信息的默认模板始终可切换

**文件：**
- 修改：`frontend/src/App.tsx:965-967`
- 修改：`frontend/src/App.test.tsx`

**步骤 1：调整模板选择渲染条件**

当草稿尚未保存（`!extraInfoDraft?.id`）时展示 `TextField select`；有默认草稿时使用 `newExtraInfoTemplateID` 显示默认值。编辑已有信息时继续仅显示分类文本。

**步骤 2：运行针对性测试确认通过**

运行：`cd frontend && npm test -- --run src/App.test.tsx -t '删除固定字段|固定字段逐行|默认模板仍可切换'`

预期：三项测试通过。

**步骤 3：提交实现**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "fix: 优化额外信息表单布局和模板选择"
```

### 任务 4：执行回归和构建验证

**文件：**
- 无源码修改。

**步骤 1：运行前端全量测试**

运行：`cd frontend && npm test -- --run`

**步骤 2：运行 Go 测试**

运行：`go test ./... -count=1`

**步骤 3：构建 Linux 应用**

运行：`./scripts/build-linux.sh`

预期：生成 `build/bin/taskai`。

**步骤 4：检查工作区**

运行：`git diff --check && git status --short`

预期：无格式问题与未提交的生成文件差异。
