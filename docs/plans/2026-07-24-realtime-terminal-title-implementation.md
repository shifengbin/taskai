# 实时终端标题实施计划

> **供 Codex 使用：** 必须使用 `superpowers:executing-plans` 逐项实施；每项行为变更遵循 `superpowers:test-driven-development`，先运行失败测试。按当前工作区约束不创建提交。

**目标：** 实时识别终端输出中的 OSC 0/2 标题序列，并在任务树及右侧终端栏显示程序设置的真实终端标题。

**架构：** 后端维持现有终端输出事件，不解析或修改 ANSI 数据。前端为每个终端维护独立的、跨事件片段的 OSC 解析状态；从输出中识别完整标题后更新 `TerminalRecord.title`，原始输出仍照常交给 xterm。终端退出时清理相应解析状态。

**技术栈：** React、TypeScript、xterm.js、Vitest、Material UI、Bash 构建脚本。

---

### 任务 1：实现可测试的 OSC 标题解析器

**文件：**
- 创建：`frontend/src/terminal-title.ts`
- 创建：`frontend/src/terminal-title.test.ts`

**步骤：**
1. 在 `terminal-title.test.ts` 编写失败测试，覆盖单个输出片段中的 `ESC ] 0 ; 标题 BEL`、`ESC ] 2 ; 标题 ESC \\`、同一片段内连续标题，以及标题序列被拆分到多个输出事件时的恢复。
2. 运行 `cd frontend && npm test -- --run src/terminal-title.test.ts`，确认因解析器不存在而失败。
3. 在 `terminal-title.ts` 定义每个终端独立保存的流式状态机和解析函数。状态机逐字符跟踪 OSC 起始、命令号、标题内容、BEL/ST 终止及超长丢弃状态；函数接收上次状态与新的原始输出，返回下次状态以及最后一个完整 OSC 0/2 标题；不得改写原始输出，也不得将其他 OSC 序列误认为标题。
4. 重跑目标 Vitest 测试，确认完整、分段及多次标题更新均通过。

### 任务 2：将实时标题接入终端事件状态

**文件：**
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/state.ts`
- 修改：`frontend/src/state.test.ts`
- 修改：`frontend/src/App.tsx`

**步骤：**
1. 在状态测试中加入标题字段：输出事件只更新对应终端的标题而不影响其他终端的输出或状态；退出事件后删除对应终端的解析状态。
2. 运行 `cd frontend && npm test -- --run src/state.test.ts`，确认新增断言失败。
3. 在 `TerminalRecord` 增加可选 `title`。在 `App` 的终端事件订阅中以 `taskId` 和 `terminalId` 为键维护解析状态：输出事件调用解析器，并在得到标题时与原有输出更新一起写入终端记录；`exited` 事件清理该键的解析状态。
4. 保持 `applyTerminalEvent` 的输出追加、错误和退出行为不变，并以明确的可选标题参数或等价的最小接口更新指定终端。
5. 重跑状态与解析器测试，确认标题更新、输出路由和退出清理通过。

### 任务 3：显示真实标题并覆盖回退文案

**文件：**
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/components/TaskTree.test.tsx`
- 修改：`frontend/src/components/TerminalView.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 在任务树和应用测试中写入失败断言：带 `title` 的终端子项和右侧终端栏显示标题；标题缺失时显示“终端”，且不得生成“终端 1”一类序号名称。
2. 运行 `cd frontend && npm test -- --run src/components/TaskTree.test.tsx src/App.test.tsx`，确认新断言失败。
3. 为终端显示添加唯一的标题回退辅助逻辑，任务树和 `TerminalView` 共用相同的 `title ?? '终端'` 语义，避免各处出现不同回退名称。
4. 重跑对应 Vitest 测试，确认真实标题、回退文案和既有终端交互均通过。

### 任务 4：完整验证与构建

**文件：**
- 修改：`README.md`（仅在终端名称行为需要用户说明时）

**步骤：**
1. 审查说明文档；如需要，在 `README.md` 中以中文说明程序可通过终端标准 OSC 标题控制序列动态更新显示名称。
2. 运行 `go test -race ./...`、`cd frontend && npm test -- --run && npm run build` 与 `git diff --check`。
3. 运行 `./scripts/build-linux.sh amd64`，确认重新生成绑定并产出 Linux 二进制文件。
