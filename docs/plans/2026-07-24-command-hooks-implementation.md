# 前后置脚本实施计划

> **供 Codex 使用：** 必须按 `superpowers:executing-plans` 逐项实施并在每个行为变更前先运行失败测试。

**目标：** 为自定义任务菜单命令提供前后置脚本、标准输入 JSON 上下文、设置 Tab 界面和使用帮助。

**架构：** 设置模型使用 `beforeScript` 和 `afterScript` 作为自定义菜单项的可选字段持久化，每项包含 `script` 和 `arguments`。应用层按保存的菜单项 ID 编排前置脚本、后台或 PTY 主命令以及后置脚本，并以任务结束标记抑制已结束任务的后置工作。React 仅请求执行已保存菜单项，设置与菜单项编辑分别使用 Tab 保留草稿。

**技术栈：** Go、Wails v2、React、TypeScript、Material UI、Vitest、Go 标准库 `os/exec` 与 `encoding/json`。

---

### 任务 1：扩展设置模型

**文件：**
- 修改：`internal/settings/settings_test.go`
- 修改：`internal/settings/settings.go`
- 修改：`frontend/src/types.ts`
- 测试：`go test ./internal/settings`

**步骤：**
1. 先在设置测试中加入 `beforeScript`、`afterScript`、空参数行和系统项脚本丢弃的断言。
2. 运行 `go test ./internal/settings`，确认测试因字段和规范化逻辑尚未存在而失败。
3. 添加 `TaskScript`，将其加入自定义 `TaskMenuItem`，并使校验只保留非空脚本及逐项非空参数。
4. 更新前端类型与菜单项克隆函数，运行 `go test ./internal/settings` 确认通过。

### 任务 2：编排主命令与脚本

**文件：**
- 修改：`app_test.go`
- 修改：`app.go`
- 修改：`internal/application/contracts.go`
- 按需要修改：`internal/terminal/manager.go`、`internal/terminal/*_test.go`
- 测试：`go test . ./internal/terminal`

**步骤：**
1. 为前置失败中止、脚本 JSON 标准输入、参数边界、后台退出、命令终端退出、后置失败通知和结束任务跳过后置写出独立失败测试。
2. 运行对应 Go 测试，确认每项因缺少统一菜单项执行入口或退出编排而失败。
3. 新增仅接受任务 ID、菜单项 ID 与终端尺寸的应用层执行入口；从保存设置读取自定义菜单项，拒绝系统项和无效 ID。
4. 以配置 Shell 和任务工作目录同步运行前置脚本；JSON 仅写入脚本的标准输入，脚本及参数按数组传入。
5. 后台命令由工作协程等待退出，命令终端按终端 ID 在 `exited` 事件中触发后置脚本；后置错误发出应用事件。
6. 在任务结束前标记任务，结束失败时恢复标记；后置启动前检查标记和执行中状态。
7. 运行 `go test . ./internal/terminal`，确认所有新旧测试通过。

### 任务 3：接入 Wails 与前端执行入口

**文件：**
- 修改：`frontend/src/api.ts`
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/App.test.tsx`
- 修改：`frontend/src/components/TaskTree.test.tsx`

**步骤：**
1. 先为统一执行 API、终端结果选择和后置错误事件添加前端失败测试。
2. 将菜单命令回调改为提交菜单项 ID，显示终端时使用统一接口返回的终端信息选中该终端。
3. 新增 Wails API 与事件订阅类型，保留现有终端事件处理。
4. 运行针对性 Vitest 测试，确认通过。

### 任务 4：实现设置 Tab 与脚本帮助

**文件：**
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`
- 按需要修改：`frontend/src/App.css`

**步骤：**
1. 先写设置主 Tab 草稿保留、菜单编辑 Tab、脚本字段保存和“？”帮助内容的失败测试。
2. 将设置内容分为“工作区与外观”“终端 Shell”“任务操作”三个 Tab，取消时仍丢弃完整草稿。
3. 将自定义菜单项编辑分为“基本配置”“前后置脚本”，分别配置可选前后置脚本和逐行参数。
4. 用带可访问名称的图标按钮和浮层展示 JSON 示例、字段定义与无占位符规则。
5. 运行 `npm test -- --run src/App.test.tsx src/components/TaskTree.test.tsx`，确认通过。

### 任务 5：文档、生成绑定与验证

**文件：**
- 修改：`README.md`
- 生成：`frontend/wailsjs/go/main/App.js`
- 生成：`frontend/wailsjs/go/main/App.d.ts`
- 生成：`frontend/wailsjs/go/models.ts`

**步骤：**
1. 用中文更新 README，说明前后置脚本配置、JSON 字段、逐行参数、错误时机和结束任务跳过规则。
2. 按 Wails 项目工作流重新生成 Go 绑定，并确认新菜单执行入口和错误事件类型可被前端调用。
3. 依次运行 `go test -race ./...`、`npm test`、`npm run build` 和 `openspec validate add-command-hooks --strict`。
4. 审查变更范围与 OpenSpec 勾选状态，确认 19 项任务均完成。
