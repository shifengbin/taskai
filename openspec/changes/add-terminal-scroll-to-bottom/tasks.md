## 1. 建立隔离开发环境与测试基线

- [x] 1.1 在任务目录的 `.worktrees` 下为 `add-terminal-scroll-to-bottom` 创建独立 Git worktree 和功能分支，并确认原工作区与 worktree 的分支、状态和路径正确。
- [x] 1.2 在 worktree 中运行 `frontend/src/terminal-session.test.ts` 与 `frontend/src/components/TerminalView.test.tsx` 的现有测试，记录无关失败并确认功能修改前的基线。

## 2. 终端会话滚动能力

- [x] 2.1 扩展 `frontend/src/terminal-session.test.ts` 的 xterm 测试桩，补充 `baseY`、`viewportY`、`onScroll`、`scrollToBottom()` 和监听释放能力，并先增加滚动状态初始同步、滚动变化、目标会话隔离及释放行为的失败测试。
- [x] 2.2 在 `TerminalSessionRegistry` 中实现按会话订阅底部状态和回到底部的公共能力，在附着与尺寸适配后同步状态，并在 detach、切换和 dispose 路径释放视图滚动监听。
- [x] 2.3 保留并补强终端活动状态回归测试，验证仅滚动历史不触发画面活动、查看历史时底部页面变化仍触发活动，且滚动监听不修改画面比较基线。
- [x] 2.4 运行 `cd frontend && npm test -- --run src/terminal-session.test.ts`，确认会话层新增与既有测试全部通过。

## 3. 终端视图回到底部交互

- [x] 3.1 扩展 `frontend/src/components/TerminalView.test.tsx` 的 xterm 测试桩，并先增加位于底部时隐藏、向上滚动后显示、手动回到底部后隐藏、点击仅滚动当前会话、切换会话同步状态及异常快照可用的失败测试。
- [x] 3.2 在 `TerminalView` 中接入会话滚动状态，使用现有 `IconButton` 和主题令牌在终端内容区右下角渲染带“回到底部”可访问名称与悬浮说明的紧凑浮动按钮。
- [x] 3.3 实现按钮点击处理：将当前会话滚动到最新输出并隐藏按钮，活动终端恢复 xterm 焦点，异常退出快照保持只读语义。
- [x] 3.4 运行 `cd frontend && npm test -- --run src/components/TerminalView.test.tsx src/terminal-session.test.ts`，确认组件与会话协作测试全部通过。

## 4. 审查与集成验证

- [x] 4.1 将当前工作区对应分支的最新变更合并到 worktree 功能分支，解决冲突并重新运行最小相关测试。
- [x] 4.2 对滚动订阅生命周期、会话隔离、异常快照只读行为、实时状态基线和按钮可访问性进行代码审查，根据审查结果修正并复测。
- [x] 4.3 运行 `cd frontend && npm test -- --run && npm run build`，随后在项目根目录运行 `go test -race ./...`，确认完整自动化验证通过。
- [x] 4.4 使用 `wails dev` 启动应用，从输出取得调试地址并使用 chrome-devtools 测试长终端滚屏、持续输出、手动回底部、按钮点击、终端切换、窗口调整和异常退出快照；测试完成后关闭调试进程。
- [x] 4.5 使用 `scripts` 下的 Linux 构建脚本编译可执行程序并打开应用，不禁用终端颜色且不让程序自动退出，完成关键场景复验后等待用户确认。

## 5. 合并、文档与归档

- [ ] 5.1 用户确认后将 worktree 功能分支合并到当前工作区项目对应分支，解决可能的冲突并再次编译项目验证合并结果。
- [ ] 5.2 将已确认行为同步到 `openspec/specs/terminal-scroll-to-bottom/spec.md` 和中文实施记录，校验 OpenSpec 变更完整性后归档该变更。
- [ ] 5.3 提交全部 Git 变更，确认提交内容只包含本功能及其文档，然后移除已合并的 worktree。
