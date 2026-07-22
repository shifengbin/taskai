# 任务操作菜单配置实施计划

> **供 Codex 使用：** 按 `superpowers:executing-plans` 技能逐项执行此计划。

**目标：** 允许用户在设置中编排任务右键与下拉菜单，并运行显示或隐藏的自定义命令。

**架构：** 设置持久化一组有序菜单项。系统项由后端规范化，名称、行为和必填性不可变，仅顺序可变；自定义项保存名称、可执行命令、逐行参数和终端显示选项。前端两个菜单入口读取同一配置；显示终端的命令使用任务 PTY，隐藏终端的命令在任务工作目录后台启动。

**技术栈：** Go、Wails、React、Material UI、xterm、Vitest、Go `os/exec`。

---

### 任务 1：定义并持久化菜单配置

**文件：**
- 修改：`internal/settings/settings.go`
- 修改：`internal/settings/settings_test.go`
- 修改：`frontend/src/types.ts`

**步骤：**
1. 编写失败测试，覆盖默认系统项、系统项规范化、排序保留、自定义项校验和旧设置回退默认值。
2. 扩展设置模型，定义有序菜单项、系统项 ID、类型、名称、命令、参数与显示终端字段。
3. 在设置校验中固定系统项内容、拒绝重复 ID 和无效自定义命令，并补齐被删除的系统项。
4. 运行 `go test ./internal/settings`，确认通过。

### 任务 2：执行自定义任务命令

**文件：**
- 修改：`internal/terminal/types.go`
- 修改：`internal/terminal/backend_unix.go`
- 修改：`internal/terminal/backend_windows.go`
- 修改：`internal/terminal/*_test.go`
- 修改：`app.go`
- 修改：`app_test.go`

**步骤：**
1. 编写失败测试，验证自定义可见命令以指定参数在任务工作目录启动 PTY，隐藏命令在同一目录后台启动。
2. 为终端启动请求加入命令和参数，保留 Shell 终端的既有默认行为。
3. 在应用层增加执行可见命令终端和后台命令的绑定；两者均仅允许执行中任务使用。
4. 运行 `go test ./internal/terminal .`，确认通过。

### 任务 3：渲染可编排任务菜单

**文件：**
- 修改：`frontend/src/components/TaskTree.tsx`
- 修改：`frontend/src/components/TaskTree.test.tsx`
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/api.ts`

**步骤：**
1. 编写失败测试，验证右键和“任务操作”显示相同顺序；执行系统项和自定义项时调用正确回调。
2. 将菜单从固定 JSX 改为设置驱动；非执行中任务仅保留编辑系统项。
3. 自定义可见命令创建并选中终端，自定义隐藏命令仅调用后台执行接口。
4. 运行 `npm test -- --run src/components/TaskTree.test.tsx src/App.test.tsx`，确认通过。

### 任务 4：增加设置编辑器

**文件：**
- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**
1. 编写失败测试，验证新增自定义项、上移下移、系统项禁用编辑、自定义项可编辑及保存的设置载荷。
2. 在设置对话框增加“任务操作菜单”列表、排序按钮和新增按钮。
3. 为自定义项提供名称、启动命令、逐行参数、显示终端和删除控件；系统项只显示排序控件。
4. 运行 `npm test -- --run src/App.test.tsx`，确认通过。

### 任务 5：规格、绑定与交付验证

**文件：**
- 修改：`openspec/changes/add-task-terminal-workspace/specs/task-tree-interface/spec.md`
- 修改：`README.md`
- 生成：`frontend/wailsjs/go/main/App.d.ts`
- 生成：`frontend/wailsjs/go/main/App.js`

**步骤：**
1. 用中文补充菜单配置、系统项限制、自定义终端与后台命令场景。
2. 更新 README 的功能说明和参数输入规则。
3. 运行 `go test -race ./...`、`cd frontend && npm test && npm run build`、`openspec validate add-task-terminal-workspace --strict`。
4. 运行 `./scripts/build-linux.sh`，确认 `build/bin/taskai` 生成。
