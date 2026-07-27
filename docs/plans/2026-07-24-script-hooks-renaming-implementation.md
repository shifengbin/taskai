# 前后置脚本语义重命名实施计划

> **供 Codex 使用：** 必须使用 `superpowers:executing-plans` 逐项实施；每项行为变更遵循 `superpowers:test-driven-development`，先运行失败测试。按当前工作区约束不创建提交。

**目标：** 将新增的命令钩子功能改为前置、后置脚本配置，并使用无迁移的脚本 JSON 字段。

**架构：** Go 设置模型由 `CommandHook`/`beforeHook`/`afterHook` 改为 `TaskScript`/`beforeScript`/`afterScript`，应用层使用同名脚本执行编排。React、Wails 绑定、应用事件及中文文档使用一致术语；主菜单命令仍保留 `command` 字段。

**技术栈：** Go、Wails v2、React、TypeScript、Material UI、Vitest、Bash 构建脚本。

---

### 任务 1：更新 OpenSpec 与设置模型

**文件：**
- 修改：`openspec/changes/add-command-hooks/{proposal.md,design.md,tasks.md}`
- 修改：`openspec/changes/add-command-hooks/specs/**/*.md`
- 修改：`internal/settings/settings_test.go`
- 修改：`internal/settings/settings.go`
- 修改：`frontend/src/types.ts`

**步骤：**
1. 在设置测试中将期望字段改为 `BeforeScript`、`AfterScript` 与 `TaskScript{Script, Arguments}`。
2. 运行 `go test ./internal/settings`，确认因旧字段不再匹配而失败。
3. 重命名设置模型、JSON 字段与规范化函数；系统菜单项继续丢弃脚本配置。
4. 更新 OpenSpec 的需求、设计和新增任务，明确该新功能不提供旧钩子字段迁移。
5. 重新运行 `go test ./internal/settings`。

### 任务 2：重命名应用层脚本执行编排

**文件：**
- 修改：`app_test.go`
- 修改：`app.go`

**步骤：**
1. 把执行测试中的设置字段、错误文本和测试名称改为前置/后置脚本语义。
2. 运行 `go test .`，确认旧的钩子符号导致编译或断言失败。
3. 将应用层 runner、starter、payload、克隆和错误事件改为 script 命名；保持主命令 JSON 上下文字段不变。
4. 运行 `go test .`，确认前置阻断、后置退出、任务结束跳过与参数 JSON 测试通过。

### 任务 3：更新前端脚本配置与帮助

**文件：**
- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/api.ts`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 将前端测试预期改为“前后置脚本”Tab、“前置脚本”和“后置脚本”可访问标签及脚本错误事件。
2. 运行 `npm test -- --run src/App.test.tsx`，确认旧文案和字段导致失败。
3. 重命名脚本草稿、保存字段、应用事件订阅与帮助文案；说明脚本可使用路径或 Shell `PATH` 中的可执行脚本。
4. 重跑目标 Vitest 测试，确认草稿保存、帮助与错误提示通过。

### 任务 4：生成绑定、更新说明并验证

**文件：**
- 修改：`README.md`
- 生成：`frontend/wailsjs/go/main/App.{js,d.ts}`
- 生成：`frontend/wailsjs/go/models.ts`

**步骤：**
1. 更新 README 的脚本配置、标准输入、错误和结束跳过说明。
2. 使用 `./scripts/build-linux.sh amd64` 重新生成 Wails 绑定与 Linux 生产产物，清理生成模型的尾随空白。
3. 运行 `go test -race ./...`、`cd frontend && npm test -- --run && npm run build`、`openspec validate add-command-hooks --strict` 与 `git diff --check`。
4. 勾选 OpenSpec 新增的重命名任务。
