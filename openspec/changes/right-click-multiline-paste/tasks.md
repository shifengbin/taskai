## 1. 开发准备

- [x] 1.1 在任务目录的 `.worktrees/` 创建本变更的 Git worktree 与开发分支，并确认与当前项目分支同步。

## 2. 模拟粘贴会话操作

- [x] 2.1 先在 `frontend/src/terminal-session.test.ts` 为通用模拟粘贴补充回归测试：完整多行文本仅调用一次 xterm `paste()`，目标会话缺失或关闭时返回失败且不调用后端写入。
- [x] 2.2 将仅用于快捷输入的会话模拟粘贴操作提炼为可供快捷输入和右键共用的操作，保留 `writeInput()` 的直接键盘输入语义。
- [x] 2.3 运行 `cd frontend && npm test -- --run src/terminal-session.test.ts`。

## 3. 右键粘贴路由

- [x] 3.1 先在 `frontend/src/components/TerminalView.test.tsx` 更新右键粘贴回归测试：多行剪贴板内容原样调用 xterm `paste()`，不调用直接 PTY 写入。
- [x] 3.2 修改默认终端的右键处理器，读取到非空剪贴板内容后调用通用模拟粘贴操作；保持禁用 TaskAI 鼠标剪贴板的终端不拦截事件、不读取剪贴板。
- [x] 3.3 补充目标终端在异步读取剪贴板期间关闭时不写入任何终端的测试，并运行 `cd frontend && npm test -- --run src/components/TerminalView.test.tsx`。

## 4. 验证与交付

- [x] 4.1 运行受影响前端测试、完整前端测试与构建：`cd frontend && npm test -- --run src/terminal-session.test.ts src/components/TerminalView.test.tsx`、`npm test`、`npm run build`。
- [x] 4.2 运行 `openspec validate right-click-multiline-paste --strict` 和 `git diff --check`。
- [x] 4.3 使用项目 Linux 构建脚本编译可执行程序并打开，等待用户确认右键多行粘贴行为。
- [ ] 4.4 经用户确认后，将开发分支合并回当前项目分支，重新编译验证，归档 OpenSpec 与同步中文实施文档，最后提交 Git 变更。
