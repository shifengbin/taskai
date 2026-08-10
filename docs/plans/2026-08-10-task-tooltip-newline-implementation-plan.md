# 任务悬浮提示换行实施计划

> **For Codex:** 使用测试驱动方式逐项执行本计划。

**目标：** 任务项悬浮提示按原始任务描述显示换行和连续空白，同时保留长行自动折行。

**架构：** 只在 `TaskTree` 已有 portal 提示容器上调整 Tailwind 空白处理类；数据读取、提示状态、定位和空描述回退保持不变。

**技术栈：** React 18、TypeScript、Tailwind CSS、Vitest、Testing Library。

---

### 任务 1：多行悬浮提示回归测试

**文件：**

- 修改：`frontend/src/components/TaskTree.test.tsx:985`

**步骤 1：编写失败测试**

在已有“悬浮任务条目时显示完整描述”测试之后，添加一个描述为 `第一行\n  缩进的第二行` 的任务。悬浮任务项后，断言 `role="tooltip"` 包含完整字符串并带有 `whitespace-pre-wrap`。

**步骤 2：确认测试失败**

运行：`cd frontend && npm test -- --run src/components/TaskTree.test.tsx`

预期：新增断言失败，因为提示容器尚未使用 `whitespace-pre-wrap`。

### 任务 2：保留提示换行

**文件：**

- 修改：`frontend/src/components/TaskTree.tsx:658-666`

**步骤 1：最小实现**

在提示容器的 `className` 中加入 `whitespace-pre-wrap`：

```tsx
className="... max-w-[480px] break-words whitespace-pre-wrap ..."
```

**步骤 2：确认测试通过**

运行：`cd frontend && npm test -- --run src/components/TaskTree.test.tsx`

预期：`TaskTree` 的所有测试通过。

### 任务 3：完整验证与验收构建

**文件：**

- 修改：`openspec/changes/preserve-task-tooltip-newlines/tasks.md`

**步骤 1：运行验证**

```sh
cd frontend && npm test && npm run build
cd .. && go test ./...
openspec validate preserve-task-tooltip-newlines --strict
scripts/build-linux.sh
```

**步骤 2：启动验收程序**

运行 Linux 构建产物，悬浮一个包含换行的任务项，确认提示按多行显示。待用户确认后再合并分支。
