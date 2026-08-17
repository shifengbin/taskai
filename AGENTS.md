# TaskAI 协作指南

## 仓库结构

- `main.go` 启动 Wails；`app.go` 负责导出的 Wails 绑定与应用编排。
- `internal/lifecycle/` 管理任务状态变更与生命周期命令链；`internal/storage/` 持久化任务和设置；`internal/settings/` 定义并校验配置。
- `internal/terminal/` 管理 PTY 会话与平台进程集成；`internal/workspace/` 管理任务工作目录路径与文件系统操作。
- `frontend/src/` 包含 React 18、Vite 和 MUI 用户界面；生成的 Wails 前端绑定位于 `frontend/wailsjs/`。
- `openspec/specs/` 存放可执行的行为规格；`docs/plans/` 存放已确认的设计和实施记录。
- `scripts/` 存放编译脚本

## 文档规范

项目中的文档需要使用中文,避免行业黑话(门禁,抓手等等),专有名词可以不使用中文

## 开发与验证

先运行最小相关测试，例如受影响的 Go 包或前端测试文件。跨层变更或交付较大改动前，运行完整验证：

```sh
go test -race ./...
cd frontend && npm test && npm run build
```

修改导出的 Wails 应用方法或绑定的 Go 类型后，重新生成绑定并检查生成的前端接口：

```sh
wails generate module
git diff -- frontend/wailsjs/
```

开发完成后需要进行编译,编译可以使用scripts文件夹下的编译脚本

## 变更边界

- 在所属层内完成变更，并保留 API 边界现有的校验和错误处理。
- 内置工作目录和 Git 命令必须保留既有的路径边界、所有权令牌与文件系统身份校验，以及工作目录边界。保留结束任务后的受控清理行为，例如删除任务工作目录；不得绕过检查或扩大其操作目标。
- 工作区根目录和所有权元数据目录仅按目录形状与软件访问能力校验（普通目录、非符号链接），不校验目录所有者、ACL 或权限位。这是基于单人使用个人电脑的威胁模型决策：删除安全由随机令牌、目录自身标记和文件系统身份比对独立支撑。
- 以持久化的任务数据为唯一事实来源；不得把临时运行时状态改作任务的持久化状态。

## 生命周期命令规则

- 保留五个钩子及其语义：`beforeStart`、`postStart`、`beforeEnd`、`postEnd` 和 `updateTask`。
- 保留执行顺序和提交时机：`beforeStart` 在开始状态提交前运行，之后运行 `postStart`；`beforeEnd` 在完成状态提交前运行，之后运行 `postEnd`。仅当执行中任务的编辑成功保存后，才运行 `updateTask`。
- 不得无意改变失败、状态提交或重试行为。`beforeStart` 或 `beforeEnd` 失败会阻止对应状态变更；`postStart`、`postEnd` 和 `updateTask` 在持久化后运行，失败时必须保留已提交的执行中、已完成或已更新状态。重试仅能针对执行记录中的钩子和任务状态进行。
- 将持久化的 `Task.LifecycleExecution` 记录与任务状态、生命周期命令链配置，以及仅在内存中的工作器和命令请求区分开来；该记录包含运行 ID、版本、进度、失败和重试信息。

## 跨平台终端规则

- 将 Windows 和 Unix 的 PTY/进程实现保留在各自的平台文件与构建约束中。
- 修改共享的终端或进程契约时，检查并测试相应的 Windows 与 Unix 实现。不得把特定操作系统的路径、Shell、信号或进程假设引入共享代码。

## OpenSpec 流程

- 用户可见行为发生变更时，更新 `openspec/specs/` 下对应的规格；代码与规格不得冲突。
- 实施重要功能或行为变更前及实施过程中，遵循既有的 OpenSpec 提案、设计和任务流程。
- 流程要求时，保持 `docs/plans/` 中已确认的实施记录与最终变更一致。
