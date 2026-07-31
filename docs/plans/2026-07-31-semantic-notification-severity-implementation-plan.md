# 语义化提示颜色实施计划

> **执行说明：** 使用 `superpowers:executing-plans` 按步骤实施并逐项验证。

**目标：** 让复制成功提示使用绿色，且所有错误与校验失败提示继续使用红色。

**实现方式：** 将 `App` 的消息状态改为包含文本和严重程度的通知对象；保留 Snackbar 的现有行为，统一由错误辅助函数创建 `error` 通知，复制成功创建 `success` 通知。

**技术栈：** React、TypeScript、Material UI、Vitest。

---

### 任务 1：添加失败的通知语义测试

**文件：**

- 修改：`frontend/src/App.test.tsx`

1. 为复制成功用例断言 Snackbar Alert 的 `severity` 为 `success`。
2. 为复制失败用例断言 Snackbar Alert 的 `severity` 为 `error`。
3. 运行 `npm test -- App.test.tsx -t '任务详情展示额外信息和系统变量，并复制当前命令链输入|复制当前命令链输入失败时不写入剪贴板'`，确认成功通知断言在实现前失败。

### 任务 2：实现语义化通知状态

**文件：**

- 修改：`frontend/src/App.tsx`

1. 定义通知对象，包含消息文本和 `success`/`error` 严重程度。
2. 将现有错误、校验和事件消息改为 `error` 通知。
3. 将复制成功消息改为 `success` 通知，并将 Snackbar Alert 绑定到通知严重程度。
4. 重新运行任务 1 的定向测试，确认通过。

### 任务 3：完整验证

**文件：**

- 修改：`frontend/src/App.test.tsx`
- 修改：`frontend/src/App.tsx`

1. 运行 `npm test`。
2. 运行 `go test ./...`。
3. 运行 `scripts/build-linux.sh`。

本计划不会自动创建 Git 提交。
