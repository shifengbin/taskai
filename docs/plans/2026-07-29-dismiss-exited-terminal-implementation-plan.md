# 异常退出终端可关闭实现计划

> **执行要求：** 必须使用 `superpowers:executing-plans`，按以下任务逐项实施。

**目标：** 让用户可以关闭任务树中异常退出的终端，并立即移除该终端条目。

**架构：** 前端 `TaskTree` 继续使用现有 `onCloseTerminal` 回调，不再以终端运行状态决定关闭按钮是否渲染。应用层已经在关闭成功后从本地终端列表删除该条目；终端管理器也能幂等处理已退出的会话，因此无需修改后端接口或状态模型。

**技术栈：** React、TypeScript、Material UI、Vitest、Testing Library、Wails。

---

### 任务 1：为异常退出终端补充任务树关闭入口

**文件：**

- 修改：`frontend/src/components/TaskTree.test.tsx:301-325`
- 修改：`frontend/src/components/TaskTree.tsx:403-416`

**第 1 步：编写失败测试**

在既有的“异常退出终端以状态点表示异常状态”测试中传入 `onCloseTerminal` spy。断言异常终端条目存在名称为“关闭终端”的按钮；点击该按钮后断言 spy 收到该异常终端。

```tsx
const onCloseTerminal = vi.fn()

render(<TaskTree terminals={[{...terminal, state: 'exited'}]} onCloseTerminal={onCloseTerminal} {...props}/>)

await user.click(within(terminalItem).getByRole('button', {name: '关闭终端'}))
expect(onCloseTerminal).toHaveBeenCalledWith(expect.objectContaining({id: terminal.id, state: 'exited'}))
```

**第 2 步：验证测试失败**

执行：

```bash
cd frontend && npm test -- --run src/components/TaskTree.test.tsx
```

预期：新增断言因找不到“关闭终端”按钮而失败。

**第 3 步：实现最小修改**

将 `TaskTree` 中关闭按钮的渲染条件从“终端状态为 `active` 且存在回调”改为“存在回调”。保留图标、无障碍标签、提示文本和 `stopPropagation` 行为。

```tsx
{onCloseTerminal && (
  <Tooltip title="关闭终端">
    {/* 保持原有 IconButton 与点击回调 */}
  </Tooltip>
)}
```

**第 4 步：验证测试通过**

再次执行：

```bash
cd frontend && npm test -- --run src/components/TaskTree.test.tsx
```

预期：该测试文件全部通过，且异常状态点断言保持通过。

**第 5 步：回归验证与提交**

执行：

```bash
cd frontend && npm test && npm run build
cd .. && go test -race ./...
./scripts/build-linux.sh
```

预期：前端测试、前端构建、Go 测试和 Linux 构建均成功。仅暂存本次修改的组件、测试和计划文档后提交。
