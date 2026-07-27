# 终端标题自适应布局实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 用红绿状态点替代终端状态文字，并让任务树与右侧栏的单行真实标题随容器宽度自适应裁剪。

**架构：** 在前端复用一个轻量的终端状态点组件或展示函数，避免任务树与右侧栏各自维护颜色和可访问文本。两个标题区域都采用 flex 剩余空间与 `minWidth: 0`，在不设固定宽度的前提下用 `textOverflow: clip` 裁剪溢出文本。

**技术栈：** React、TypeScript、Material UI、Vitest、Testing Library。

---

### 任务 1：编写终端状态点与标题布局的失败测试

**文件：**
- 修改：`frontend/src/components/TaskTree.test.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤 1：添加任务树状态点断言**

为活跃终端新增断言：存在可访问名称为“终端状态：运行中”的状态点，且页面不包含状态文字“运行中”。为已退出终端新增对应“终端状态：已退出”的断言。

**步骤 2：添加标题布局断言**

为任务树终端标题增加测试标识，断言其样式为单行、隐藏溢出、`textOverflow: clip`，并且标题容器可伸缩。为 `TerminalView` 的测试替身添加相同布局断言所需的属性，或将该视图单独可测地渲染。

**步骤 3：运行失败测试**

运行：`npm test -- --run src/components/TaskTree.test.tsx src/App.test.tsx`

预期：失败，原因是当前界面仍渲染状态文字 `Chip`，标题仍使用 `noWrap` 的省略号行为。

### 任务 2：实现共享的终端状态点

**文件：**
- 新建：`frontend/src/components/TerminalStatusDot.tsx`
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/components/TerminalView.tsx`

**步骤 1：实现最小状态点组件**

组件接收 `TerminalRecord['state']`，以主题 `success.main` 渲染活跃状态，以 `error.main` 渲染已退出状态；使用 `role="status"` 和 `aria-label` 暴露“终端状态：运行中”或“终端状态：已退出”。圆点为不可收缩的小型 `Box`。

**步骤 2：替换两处状态文字**

删除 `terminalStatusLabel` 和 `Chip` 的引用；在任务树和右侧终端栏中插入 `TerminalStatusDot`。保留终端图标与关闭按钮行为。

**步骤 3：运行状态点测试**

运行：`npm test -- --run src/components/TaskTree.test.tsx src/App.test.tsx`

预期：状态点断言通过，既有任务树交互测试保持通过。

### 任务 3：实现单行自适应标题

**文件：**
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/components/TerminalView.tsx`
- 修改：`frontend/src/components/TaskTree.test.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤 1：调整任务树标题容器**

给 `ListItemText` 的根节点设置 `flex: 1` 与 `minWidth: 0`；标题文本设置 `whiteSpace: 'nowrap'`、`overflow: 'hidden'`、`textOverflow: 'clip'`，不使用 `noWrap`。为其设置稳定的测试标识。

**步骤 2：调整右侧标题容器**

用可伸缩 `Box` 包裹 `Typography`，设置 `flex: 1`、`minWidth: 0`；文本应用与任务树相同的单行裁剪样式。移除原先的 `noWrap`，避免 MUI 写入省略号。

**步骤 3：运行组件测试**

运行：`npm test -- --run src/components/TaskTree.test.tsx src/App.test.tsx`

预期：任务树、右侧真实标题与长标题布局断言全部通过。

### 任务 4：完整验证

**文件：**
- 修改：无

**步骤 1：运行前端测试与构建**

运行：

```bash
cd frontend
npm test -- --run
npm run build
```

预期：全部测试通过，TypeScript 编译通过。

**步骤 2：使用项目脚本构建**

运行：`./scripts/build-linux.sh amd64`

预期：生成 `build/bin/taskai`。

**步骤 3：检查改动**

运行：`git diff --check`

预期：无空白错误。

**提交说明：** 当前工作区包含用户尚未提交的改动；依照约束，本计划不执行 `git commit`。
