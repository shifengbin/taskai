# 终端右键粘贴实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 在终端内容区域右键时，将系统剪贴板的非空文本直接输入当前终端。

**架构：** `TerminalView` 的终端容器处理 `contextmenu` 事件，阻止默认菜单后调用 Wails Runtime 的 `ClipboardGetText`。成功读取的非空文本复用现有 `onWrite` 回调；空值和异常无副作用。

**技术栈：** React、TypeScript、Wails Runtime、Vitest、Testing Library。

---

### 任务 1：终端容器右键粘贴

**文件：**

- 修改：`frontend/src/components/TerminalView.tsx`
- 测试：`frontend/src/components/TerminalView.test.tsx`
- 修改：`README.md`

**步骤 1：编写失败测试**

模拟 `ClipboardGetText`，渲染终端内容容器并触发 `contextmenu`。断言事件默认行为被阻止，非空剪贴板内容传给 `onWrite`；空内容不传入。

**步骤 2：确认测试失败**

运行：

```bash
cd frontend && npm test -- --run src/components/TerminalView.test.tsx
```

预期：测试因右键容器尚未读取剪贴板或调用 `onWrite` 而失败。

**步骤 3：实现最小功能**

从 Wails Runtime 导入 `ClipboardGetText`。为终端内容容器添加 `onContextMenu`：阻止默认行为，异步读取剪贴板，将非空内容传给 `onWrite`，忽略读取失败。

**步骤 4：确认测试通过**

再次运行步骤 2 的命令，预期全部通过。

**步骤 5：更新说明**

在 README 的终端功能描述中补充“右键可直接粘贴系统剪贴板内容”。

**步骤 6：完整验证**

运行：

```bash
go test -race ./...
cd frontend && npm test && npm run build
./scripts/build-linux.sh amd64
git diff --check
```

预期：全部命令成功。根据工作区约束，不在本任务中创建 Git 提交。
