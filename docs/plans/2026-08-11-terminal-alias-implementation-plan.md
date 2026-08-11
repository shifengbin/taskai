# 终端别名 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 允许用户为当前终端会话设置仅运行期有效的别名，并在两个终端名称入口显示两行实际信息提示。

**Architecture:** 在 `TerminalRecord` 中添加独立于 `title` 的可选 `alias`，由名称辅助函数集中派生实际名称、展示名称与提示内容。新增可复用的 `TerminalName` 组件承载双击编辑和提示；`App` 仅更新对应会话的 `alias`，因此标题解析和实时状态链不改变。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、Radix Tooltip、Tailwind CSS。

---

### Task 1: 定义别名名称模型

**Files:**
- Modify: `frontend/src/types.ts:175-188`
- Test: `frontend/src/types.test.ts`

**Step 1: Write the failing test**

新增测试，断言 `terminalActualName` 保留现有标题回退、`terminalDisplayName` 优先使用去除首尾空白后的别名、空白别名恢复实际名称，且 `terminalAliasDetails` 始终按实际名称和启动命令（含缺省文本）返回两行数据。

**Step 2: Run test to verify it fails**

Run: `npm test -- --run src/types.test.ts`

Expected: FAIL，提示名称辅助函数或 `alias` 字段尚不存在。

**Step 3: Write minimal implementation**

在 `TerminalRecord` 添加 `alias?: string`，并将现有标题回退抽取为 `terminalActualName`。`terminalDisplayName` 只在别名经 `trim()` 后非空时返回别名；新增 `terminalAliasDetails` 返回实际名称和 `command?.trim() || '未提供启动命令'`。

**Step 4: Run test to verify it passes**

Run: `npm test -- --run src/types.test.ts`

Expected: PASS。

### Task 2: 构建共享的可编辑终端名称组件

**Files:**
- Create: `frontend/src/components/TerminalName.tsx`
- Create: `frontend/src/components/TerminalName.test.tsx`

**Step 1: Write the failing test**

为组件写入测试：双击文本后输入框自动聚焦；Enter 和失焦分别保存去除首尾空白后的别名；Escape 取消草稿；有别名时打开两行提示；无别名时不渲染提示。

**Step 2: Run test to verify it fails**

Run: `npm test -- --run src/components/TerminalName.test.tsx`

Expected: FAIL，提示组件文件不存在。

**Step 3: Write minimal implementation**

组件接收 `terminal`、`onAliasChange`、`className` 与可选测试标识。使用本地编辑状态和 `Input`，将 `onDoubleClick` 替换为已聚焦的单行输入框；Enter/blur 调用 `onAliasChange(terminal, draft.trim() || undefined)`，Escape 恢复显示。仅别名非空时用 `Tooltip`、`TooltipTrigger` 和 `TooltipContent` 包裹名称，内容渲染实际名称和启动命令两个块级行。

**Step 4: Run test to verify it passes**

Run: `npm test -- --run src/components/TerminalName.test.tsx`

Expected: PASS。

### Task 3: 在任务树和终端标题栏复用名称组件

**Files:**
- Modify: `frontend/src/components/TaskTree.tsx:66-123,597-624`
- Modify: `frontend/src/components/TerminalView.tsx:1-27,171-183`
- Modify: `frontend/src/components/TaskTree.test.tsx`
- Modify: `frontend/src/components/TerminalView.test.tsx`

**Step 1: Write the failing test**

分别为任务树和标题栏增加组件测试，断言双击名称调用重命名回调，保存后重新渲染显示别名；验证两处都保留选择终端与现有标题栏操作。

**Step 2: Run test to verify it fails**

Run: `npm test -- --run src/components/TaskTree.test.tsx src/components/TerminalView.test.tsx`

Expected: FAIL，组件未提供别名回调或未渲染可编辑名称。

**Step 3: Write minimal implementation**

给 `TaskTreeProps` 和 `TerminalViewProps` 添加 `onAliasChange`，在两个现有 `terminalDisplayName` 位置改用 `TerminalName`。在任务树编辑时阻止双击向行级选择行为传播；其余单击、键盘选择、关闭与快捷输入行为不改。

**Step 4: Run test to verify it passes**

Run: `npm test -- --run src/components/TaskTree.test.tsx src/components/TerminalView.test.tsx`

Expected: PASS。

### Task 4: 在应用状态中更新会话别名并保护实时状态

**Files:**
- Modify: `frontend/src/App.tsx:280-296,846-854,1266-1340`
- Modify: `frontend/src/state.test.ts`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing test**

在状态和应用测试中断言，别名更新只替换匹配会话的 `alias`，不改写 `title`、`realtimeStatus` 或其他会话；同时实际 OSC 标题变化在存在别名时仍通过现有 `shouldReportTerminalTitleActivity` 上报。

**Step 2: Run test to verify it fails**

Run: `npm test -- --run src/state.test.ts src/App.test.tsx`

Expected: FAIL，应用没有别名更新路径或状态断言未满足。

**Step 3: Write minimal implementation**

在 `App` 添加稳定的别名更新回调，通过终端 `taskId` 与 `id` 更新单条 `TerminalRecord` 的 `alias`，并将回调传给 `TaskTree` 和 `TerminalView`。不得修改终端事件处理、标题解析状态或 `shouldReportTerminalTitleActivity` 的入参。

**Step 4: Run test to verify it passes**

Run: `npm test -- --run src/state.test.ts src/App.test.tsx`

Expected: PASS。

### Task 5: 完整验证和 OpenSpec 同步

**Files:**
- Modify: `openspec/changes/add-terminal-alias/tasks.md`

**Step 1: Run focused tests**

Run: `npm test -- --run src/types.test.ts src/components/TerminalName.test.tsx src/components/TaskTree.test.tsx src/components/TerminalView.test.tsx src/state.test.ts src/App.test.tsx`

Expected: PASS。

**Step 2: Run full validation**

Run: `npm test -- --run && npm run build && go test -race ./...`

Expected: PASS；Go 测试在前端构建生成 `frontend/dist` 后执行。

**Step 3: Update OpenSpec tasks**

将完成的 `tasks.md` 检查项更新为 `- [x]`，再运行 `openspec status --change add-terminal-alias` 确认 8/8 完成。
