# 终端滚动状态判定实施计划

> **For Codex:** 必须按 `openspec/changes/fix-terminal-scrollback-status/tasks.md` 逐项实施并更新完成状态。

**目标：** 无论用户是否向上查看终端历史，活动终端页发生的实际输出变化都能准确更新任务实时状态。

**方案：** `TerminalSessionRegistry` 在 xterm 解析输出后，从活动缓冲区的 `baseY` 读取底部终端页快照。快照不再包含用户的 `viewportY`，滚动事件也不再重置状态比较基线；尺寸适配和外观更新仍保留基线重置。先使用 Vitest 回归用例证明问题，再做最小实现。

**技术：** React、TypeScript、Vitest、`@xterm/xterm`、Wails、Go。

---

## 1. 建立回归用例

**文件：**
- 修改：`frontend/src/terminal-session.test.ts`

1. 扩展 xterm 测试桩的活动缓冲区，使 `baseY` 和 `viewportY` 可以独立设置。
2. 新增测试：用户滚动到历史位置后，底部活动页中的单元格变化仍会调用活动回调。
3. 新增测试：仅滚动或仅变更历史区域、活动页未变时，不调用活动回调。
4. 运行 `cd frontend && npm test -- terminal-session.test.ts`，确认新用例在旧实现上失败。

## 2. 最小实现

**文件：**
- 修改：`frontend/src/terminal-session.ts`
- 测试：`frontend/src/terminal-session.test.ts`

1. 让终端画面快照从 `buffer.baseY` 起读取 `rows` 行。
2. 从快照和相等性比较中移除 `viewportY`。
3. 移除只用于滚动重建快照基线的 xterm `onScroll` 监听及其释放。
4. 保留 `onWriteParsed`、单元格签名、尺寸适配和外观更新时的现有基线处理。
5. 重跑受影响测试，确认通过。

## 3. 完整验证

1. 运行 `cd frontend && npm test -- --run && npm run build`。
2. 在前端产物生成后运行 `go test -race ./...`。
3. 运行 `./scripts/build-linux.sh`。
4. 使用 `wails dev` 创建可滚动输出的终端，向上滚动到历史内容后继续产生输出，确认任务状态仍切换为工作中；随后启动编译产物复验。
