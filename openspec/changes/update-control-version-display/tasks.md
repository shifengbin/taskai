# Tasks: update-control-version-display

## 1. Worktree 准备

- [ ] 1.1 在 `taskai/` 仓库下创建 `.worktrees/` 目录，从当前工作区分支创建 worktree 分支 `update-control-version-display` 并进入开发

## 2. 失败测试先行

- [ ] 2.1 `internal/updater` 新增测试：`NewService` 以 `CurrentVersion: "v1.0.0"` 构造后，`State()` 返回的 `CurrentVersion` 为 `v1.0.0`；以无 `v` 前缀的 `1.0.0` 构造时返回规范化的 `v1.0.0`；先运行确认编译失败（字段尚不存在）
- [ ] 2.2 `app_test.go` 新增测试：`updaterService` 为 nil 时 `GetUpdateState()` 返回 `State{Status: idle, CurrentVersion: appVersion}`（即 `v0.0.0-dev`）；先运行确认失败
- [ ] 2.3 `frontend/src/components/UpdateControl.test.tsx` 新增用例（先运行确认失败）：idle 且 `currentVersion: "v1.2.3"` 时渲染 `v1.2.3` 静态文本（`data-testid="app-version"`）；idle 且无 `currentVersion` 时该位置不渲染任何内容；`available` 状态渲染更新按钮且不渲染版本文本；版本文本不是按钮、不可点击

## 3. 实现

- [ ] 3.1 `internal/updater/service.go`：`State` 结构体增加 `CurrentVersion string`（`json:"currentVersion,omitempty"`）；`State()` 返回时以服务持有的规范化 `currentVersion` 填充
- [ ] 3.2 `app.go`：`GetUpdateState` 的 `updaterService == nil` 分支返回 `State{Status: StatusIdle, CurrentVersion: appVersion}`
- [ ] 3.3 `frontend/src/types.ts`：`UpdateState` 增加可选字段 `currentVersion?: string`
- [ ] 3.4 `frontend/src/components/UpdateControl.tsx`：`state.status === 'idle'` 且 `currentVersion` 非空时渲染次级样式的静态文本（小号、弱化颜色、自然宽度、`data-testid="app-version"`），非 idle 分支保持现有按钮渲染完全不变；兜底 `idleState` 常量不加版本字段（缺失时优雅降级为不显示）
- [ ] 3.5 执行 `wails generate module` 并检查 `git diff -- frontend/wailsjs/`，确认生成模型仅新增 `currentVersion` 可选字段

## 4. 最小相关测试

- [ ] 4.1 `go test ./internal/updater/... ./...`（包根，覆盖 `app_test.go`）全绿
- [ ] 4.2 `cd frontend && npm test`（重点 `UpdateControl.test.tsx`）全绿

## 5. 集成测试（wails dev + 浏览器，先冒烟后专项）

- [ ] 5.1 冒烟：以 `wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 启动（不设置 `TASKAI_UPDATE_TEST_URL`，不禁用终端颜色，不让进程自动退出），从输出获取调试地址；用浏览器（mcp chrome-devtools）访问，确认应用正常加载、任务列表可用
- [ ] 5.2 专项—无更新显示当前版本：同一会话中断言顶栏"任务工作台"右侧出现 `v0.0.0-rc5` 静态文本（更新服务为 nil 的路径）；元素为纯文本而非按钮
- [ ] 5.3 专项—版本来自编译注入：重启为 `-ldflags "-X main.appVersion=v0.0.0-rc4"`，确认显示变为 `v0.0.0-rc4`（证明非硬编码）
- [ ] 5.4 专项—有更新时按钮替换版本：启动 `go run ./cmd/update-test-server` 记录 `TASKAI_UPDATE_TEST_URL`，以 `TASKAI_UPDATE_TEST_URL=http://127.0.0.1:<端口> wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 启动；等待检查完成后断言：出现 `new` 更新入口且版本文本消失（互斥）
- [ ] 5.5 专项—版本号不可交互：在 5.2 的显示状态下点击/悬浮版本文本，断言不触发任何对话框、导航或下载状态变化
- [ ] 5.6 关闭 wails dev 与测试服务器，清理临时进程

## 6. 完整验证

- [ ] 6.1 `go test -race ./...` 与 `go test -tags updater_integration ./...` 全绿
- [ ] 6.2 `cd frontend && npm test && npm run build` 全绿

## 7. 构建与确认

- [ ] 7.1 使用 `scripts/` 下当前平台正式构建脚本编译可执行程序并打开程序（确认未启用 `updater_integration`、版本注入有效）
- [ ] 7.2 在编译产物中确认顶栏显示的版本号与注入版本一致；等待用户确认

## 8. 合并与收尾

- [ ] 8.1 先将工作区主分支反向合并到 `update-control-version-display` 分支，出现冲突先解决
- [ ] 8.2 确认无误后将 `update-control-version-display` 合并回工作区对应分支
- [ ] 8.3 合并后重新编译项目验证合并无问题
- [ ] 8.4 归档 openspec 变更并同步 `openspec/specs/application-auto-update/spec.md`；按流程要求同步 `docs/plans/` 实施记录
- [ ] 8.5 提交 git 变更并移除已合并的 worktree
