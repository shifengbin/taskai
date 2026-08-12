# 右侧终端会话详情行内展示实施计划

> **For Codex:** 实施时必须使用 `openspec-apply-change` 和 `superpowers:test-driven-development`，按 `openspec/changes/show-terminal-original-title-inline/tasks.md` 逐项完成并更新状态。

**目标：** 将右侧终端标题栏中别名终端的名称改为 `别名(实际标题:启动命令)`，移除该位置的悬浮提示，同时保持任务树提示和别名编辑行为不变。

**架构：** 扩展共享 `TerminalName` 组件，使调用位置可以选择现有 Tooltip 方式或行内会话详情方式。默认仍为 Tooltip，任务树无需改变；`TerminalView` 显式启用行内方式，因此两处继续共享详情派生和编辑状态，又不会互相改变展示行为。

**技术栈：** React 18、TypeScript、Vitest、Testing Library、Radix Tooltip、Wails。

---

## 任务一：以测试固定共享组件的两种展示方式

**文件：**

- 修改：`frontend/src/components/TerminalName.test.tsx`
- 参考：`frontend/src/components/TerminalName.tsx`

### 步骤 1：编写行内会话详情方式的失败测试

给具有标题 `npm run dev`、启动命令 `zsh`、别名 `前端调试` 的终端传入新的行内展示属性，断言：

```tsx
expect(screen.getByText('前端调试(npm run dev:zsh)')).toBeInTheDocument()
fireEvent.pointerMove(screen.getByText('前端调试(npm run dev:zsh)'))
expect(screen.queryByTestId('terminal-alias-details')).not.toBeInTheDocument()
expect(screen.queryByText(/原标题/)).not.toBeInTheDocument()
```

再覆盖三条边界：启动命令缺失时显示 `未提供启动命令`；无别名时仍只显示 `npm run dev`；双击组合标题后，输入框值只包含 `前端调试`，不包含括号内会话详情。

### 步骤 2：运行测试并确认失败

```sh
cd frontend
npm test -- TerminalName.test.tsx
```

预期：测试因 `TerminalName` 尚不支持行内展示属性或尚未输出组合标题而失败；原有 Tooltip 测试继续通过。

### 步骤 3：不修改既有任务树提示测试

保留现有默认 Tooltip 内容、右侧定位和 `pointer-events-none` 断言，确保新属性没有改变默认行为。

## 任务二：实现共享组件的行内会话详情方式

**文件：**

- 修改：`frontend/src/components/TerminalName.tsx`
- 测试：`frontend/src/components/TerminalName.test.tsx`

### 步骤 1：增加明确的详情展示属性

在 `TerminalNameProps` 中增加可选属性，例如：

```ts
detailsDisplay?: 'tooltip' | 'inline-session-details'
```

默认值使用 `tooltip`，使任务树和其他未传值的调用保持现状。

### 步骤 2：派生行内展示文本

保留 `hasAlias` 判断和当前编辑分支。非编辑状态下：

```ts
const displayName = terminalDisplayName(terminal)
const details = terminalAliasDetails(terminal)
const renderedName = detailsDisplay === 'inline-session-details' && hasAlias
  ? `${displayName}(${details.actualName}:${details.command})`
  : displayName
```

需要从 `types.ts` 引入现有 `terminalActualName`，不得修改 `terminal.alias`、`terminal.title` 或 `terminal.command`。

### 步骤 3：在行内方式下跳过 Tooltip

名称元素继续包含整段 `renderedName` 并沿用双击处理。仅当 `detailsDisplay === 'tooltip'` 且存在别名时包装现有 Tooltip；行内方式直接返回名称元素。

### 步骤 4：运行共享组件测试

```sh
cd frontend
npm test -- TerminalName.test.tsx
```

预期：新增行内展示、编辑边界与原有 Tooltip 测试全部通过。

## 任务三：接入右侧终端标题栏

**文件：**

- 修改：`frontend/src/components/TerminalView.tsx`
- 修改：`frontend/src/components/TerminalView.test.tsx`
- 回归：`frontend/src/components/TaskTree.test.tsx`

### 步骤 1：先修改右侧组件测试

把“右侧标题栏继续使用默认的终端提示定位”改为“右侧标题栏直接显示会话详情且没有提示”。对别名为 `前端调试`、实际标题为 `npm run dev`、启动命令为 `zsh` 的终端断言：

```tsx
expect(screen.getByTestId('terminal-view-title')).toHaveTextContent('前端调试(npm run dev:zsh)')
fireEvent.pointerMove(screen.getByTestId('terminal-view-title'))
expect(screen.queryByTestId('terminal-alias-details')).not.toBeInTheDocument()
expect(screen.getByTestId('terminal-view-title')).not.toHaveTextContent('原标题')
```

继续保留标题容器可收缩、名称单行裁剪和双击保存别名的既有测试。

### 步骤 2：确认接入前测试失败

```sh
cd frontend
npm test -- TerminalView.test.tsx
```

预期：右侧标题仍只显示别名并创建 Tooltip，因此新断言失败。

### 步骤 3：在右侧标题启用行内方式

只在 `TerminalView` 的 `TerminalName` 调用中传入行内会话详情属性。不更改标题栏的 40px 网格行、`minWidth: 0` 容器、名称裁剪类或快捷输入操作区。

### 步骤 4：运行三个相关组件测试

```sh
cd frontend
npm test -- TerminalName.test.tsx TerminalView.test.tsx TaskTree.test.tsx
```

预期：全部通过；任务树仍显示位于右侧的两行提示，右侧标题不再出现提示。

## 任务四：完整验证与桌面集成测试

**文件：**

- 更新任务状态：`openspec/changes/show-terminal-original-title-inline/tasks.md`
- 必要时同步：`docs/plans/2026-08-12-terminal-original-title-inline-implementation-plan.md`

### 步骤 1：运行完整前端验证

```sh
cd frontend
npm test
npm run build
```

预期：测试全部通过，Vite 生产构建成功。

### 步骤 2：严格校验 OpenSpec

```sh
openspec validate show-terminal-original-title-inline --strict
```

预期：变更校验通过且没有警告或错误。

### 步骤 3：使用 Wails 开发模式集成测试

从项目根目录运行：

```sh
wails dev
```

保持进程运行，从输出取得调试地址后通过浏览器验证：

1. 为活动终端设置别名，右侧立即显示 `别名(实际标题:启动命令)`，且没有固定的 `原标题` 文字。
2. 鼠标悬浮右侧组合标题，不出现 Tooltip。
3. 双击组合标题，只编辑别名；Enter、失焦、Escape 与清空行为保持不变。
4. 悬浮任务树别名，仍在条目右侧显示实际标题与启动命令两行提示。
5. 在亮色、暗色和较窄窗口下，标题保持 40px 高且按现有规则单行裁剪。

验证完成后关闭 `wails dev`。

### 步骤 4：编译并打开可执行程序

使用 `scripts` 目录中的项目编译脚本生成程序，不设置禁用终端颜色的环境变量。打开编译产物并复验上述关键场景，保持程序运行，等待用户确认。

### 步骤 5：按项目流程完成合并与归档

用户确认后，将 worktree 分支合并到当前工作区对应分支，重新编译验证；随后同步文档、归档 OpenSpec 变更、提交全部变更并移除已合并 worktree。
