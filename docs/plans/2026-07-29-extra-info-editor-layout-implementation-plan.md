# 额外信息编辑表单布局实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 让模板、信息和任务的额外信息编辑表单使用紧凑且可自适应换行的布局，消除输入框过大、控件贴近和操作项拥挤的问题。

**架构：** 仅修改 `App.tsx` 中三个已有编辑区域的 MUI 布局属性和字段尺寸，不改变模板、信息、任务的状态模型、校验规则或后端请求。为关键布局容器添加稳定的测试标识，并用最小列宽的 CSS Grid 在实际弹窗宽度不足时自动换行；参数行则用分隔线和垂直留白区分，避免嵌套卡片。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library、Go、Wails。

---

### 任务 1：为三个编辑场景补充布局回归测试

**文件：**
- 修改：`frontend/src/App.test.tsx`
- 参考：`frontend/src/App.tsx` 中额外信息模板、信息编辑弹窗和任务动态参数编辑区

**步骤 1：编写失败的测试**

在现有额外信息测试附近增加覆盖三个场景的测试：

```tsx
expect(screen.getByTestId('extra-info-template-basic-fields')).toHaveStyle({ display: 'grid' })
expect(screen.getByTestId('extra-info-template-fixed-field-0')).toHaveStyle({ borderTopStyle: 'solid' })
expect(screen.getByTestId('extra-info-draft-fields')).toHaveStyle({ display: 'grid' })
expect(screen.getByTestId('task-extra-info-parameter-0')).toHaveStyle({ borderTopStyle: 'solid' })
expect(screen.getByLabelText('分类')).toHaveClass('MuiInputBase-sizeSmall')
```

测试通过既有 UI 流程分别打开模板编辑、信息编辑和任务动态参数编辑，验证紧凑网格、分隔行和小尺寸输入框的结构存在；继续使用已存在的模拟服务，避免改变业务测试环境。

**步骤 2：运行测试，确认失败**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：失败，提示上述测试标识不存在，或分类输入框尚未使用 `small` 尺寸。

**步骤 3：提交测试（可选的独立检查点）**

若测试改动可独立提交，执行：

```bash
git add frontend/src/App.test.tsx
git commit -m "test: 覆盖额外信息编辑表单布局"
```

### 任务 2：统一模板和信息编辑弹窗的紧凑布局

**文件：**
- 修改：`frontend/src/App.tsx` 中 `extraInfoEditorOpen` 和 `extraInfoTemplateDraft` 对应的两个 `Dialog`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：实现最小布局改动**

1. 将分类、备注、模板选择和所有固定字段 `TextField` 统一设置为 `size="small"`。
2. 将模板的分类和备注包入 `data-testid="extra-info-template-basic-fields"` 的两列自适应网格；信息的固定字段包入 `data-testid="extra-info-draft-fields"` 的自适应网格。
3. 为固定字段和动态参数的每一项增加 `py`、`borderTop`、`borderColor: 'divider'` 和稳定测试标识，例如 `extra-info-template-fixed-field-0`。删除原有参数外框的 `border`、`borderRadius`、`p`，使列表保持平面结构。
4. 参数键、显示名称、类型使用 `repeat(auto-fit, minmax(...))` 网格；复选框、必填设置和删除按钮置于可换行的独立操作行。
5. 保持既有 `onChange`、内置 Git 字段限制、默认值输入和删除逻辑不变。

关键网格形态：

```tsx
sx={{
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
  gap: 1.5,
  minWidth: 0,
}}
```

**步骤 2：运行目标测试，确认通过**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：`App.test.tsx` 全部通过；既有模板、信息创建编辑、复选框默认值和参数行为测试继续通过。

**步骤 3：提交实现**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "fix: 整理额外信息编辑表单布局"
```

### 任务 3：整理任务中的动态参数编辑区

**文件：**
- 修改：`frontend/src/App.tsx` 中 `TaskExtraInfoSnapshotFields`
- 测试：`frontend/src/App.test.tsx`

**步骤 1：实现最小布局改动**

1. 减小任务动态参数编辑区在窄窗口的左侧缩进，保留与信息复选框的层级关系。
2. 为每条可编辑参数增加 `data-testid="task-extra-info-parameter-{index}"`、细分隔线和一致的纵向间距。
3. 使用与模板一致的最小列宽网格排列键、显示名称和类型，值输入、复选框、必填和删除按钮使用可换行操作行。
4. 保持只读快照仅显示名称和值的行为，以及任务级动态参数可新增、编辑、删除的业务逻辑不变。

**步骤 2：运行目标测试，确认通过**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：任务额外信息的既有勾选、动态参数新增和编辑测试继续通过，新增布局断言通过。

**步骤 3：提交实现（若任务 2 未合并提交）**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "fix: 优化任务动态参数编辑布局"
```

### 任务 4：完整验证

**文件：**
- 验证：`frontend/src/App.test.tsx`、前端测试套件、Go 代码和 Linux 构建产物

**步骤 1：检查格式和完整前端测试**

```bash
git diff --check
cd frontend && npm test -- --run
```

预期：无空白错误，前端所有测试通过。

**步骤 2：运行后端测试和项目构建**

```bash
go test ./... -count=1
./scripts/build-linux.sh
```

预期：Go 测试通过，Linux 构建成功并生成 `build/bin/taskai`。

**步骤 3：检查工作区状态**

```bash
git status --short
git diff --check
```

预期：仅包含预期的源代码、测试和计划文档改动；不存在格式错误。
