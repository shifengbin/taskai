# 终端选区自动复制实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 终端选中非空文本时自动复制到系统剪贴板。

**架构：** `TerminalView` 在创建 xterm 后订阅选区变化事件，通过 `getSelection()` 读取文本，并调用 Wails Runtime 的 `ClipboardSetText`。空选区和写入失败均不影响终端操作。

**技术栈：** React、TypeScript、xterm、Wails Runtime、Vitest、Testing Library。

---

### 任务 1：终端选区复制

**文件：**

- 修改：`frontend/src/components/TerminalView.tsx`
- 测试：`frontend/src/components/TerminalView.test.tsx`
- 修改：`README.md`

**步骤 1：编写失败测试**

扩展 xterm 模拟对象，暴露 `getSelection`、`onSelectionChange` 和可触发的选区回调；模拟 `ClipboardSetText`。新增测试：非空选区变化会写入相同文本，空选区不写入。

**步骤 2：确认测试失败**

运行：

```bash
cd frontend && npm test -- --run src/components/TerminalView.test.tsx
```

预期：测试因尚未注册选区监听或未调用剪贴板 API 而失败。

**步骤 3：实现最小功能**

从 `../wailsjs/runtime/runtime` 引入 `ClipboardSetText`。xterm 创建完成后订阅 `onSelectionChange`，当 `getSelection()` 返回非空内容时调用该 API 并忽略拒绝结果；在 effect 清理函数中释放监听器。

**步骤 4：确认测试通过**

再次运行步骤 2 的命令，预期全部通过。

**步骤 5：更新说明**

在 README 的终端功能描述中说明：终端选中的非空文本会自动复制到系统剪贴板。

**步骤 6：完整验证**

运行：

```bash
go test -race ./...
cd frontend && npm test && npm run build
./scripts/build-linux.sh amd64
git diff --check
```

预期：全部命令成功。根据工作区约束，不在本任务中创建 Git 提交。
