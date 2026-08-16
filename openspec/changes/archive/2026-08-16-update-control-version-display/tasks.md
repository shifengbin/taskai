# Tasks: update-control-version-display

## 1. Worktree 准备

- [x] 1.1 在 `taskai/` 仓库下创建 `.worktrees/` 目录，从当前工作区分支创建 worktree 分支 `update-control-version-display` 并进入开发

## 2. 失败测试先行

- [x] 2.1 `internal/updater` 新增测试：`NewService` 以 `CurrentVersion: "v1.0.0"` 构造后，`State()` 返回的 `CurrentVersion` 为 `v1.0.0`；以无 `v` 前缀的 `1.0.0` 构造时返回规范化的 `v1.0.0`；先运行确认编译失败（字段尚不存在）
- [x] 2.2 `app_test.go` 新增测试：`updaterService` 为 nil 时 `GetUpdateState()` 返回 `State{Status: idle, CurrentVersion: appVersion}`（即 `v0.0.0-dev`）；先运行确认失败
- [x] 2.3 `frontend/src/components/UpdateControl.test.tsx` 新增用例（先运行确认失败）：idle 且 `currentVersion: "v1.2.3"` 时渲染 `v1.2.3` 静态文本（`data-testid="app-version"`）；idle 且无 `currentVersion` 时该位置不渲染任何内容；`available` 状态渲染更新按钮且不渲染版本文本；版本文本不是按钮、不可点击

## 3. 实现

- [x] 3.1 `internal/updater/service.go`：`State` 结构体增加 `CurrentVersion string`（`json:"currentVersion,omitempty"`）；`State()` 返回时以服务持有的规范化 `currentVersion` 填充
- [x] 3.2 `app.go`：`GetUpdateState` 的 `updaterService == nil` 分支返回 `State{Status: StatusIdle, CurrentVersion: appVersion}`
- [x] 3.3 `frontend/src/types.ts`：`UpdateState` 增加可选字段 `currentVersion?: string`
- [x] 3.4 `frontend/src/components/UpdateControl.tsx`：`state.status === 'idle'` 且 `currentVersion` 非空时渲染次级样式的静态文本（小号、弱化颜色、自然宽度、`data-testid="app-version"`），非 idle 分支保持现有按钮渲染完全不变；兜底 `idleState` 常量不加版本字段（缺失时优雅降级为不显示）
- [x] 3.5 执行 `wails generate module` 并检查 `git diff -- frontend/wailsjs/`，确认生成模型仅新增 `currentVersion` 可选字段

## 4. 最小相关测试

- [x] 4.1 `go test ./internal/updater/... ./...`（包根，覆盖 `app_test.go`）：`internal/updater` 全绿；根包 updater 相关全部通过（含新增 `TestGetUpdateStateWithoutUpdaterCarriesCurrentVersion`）。全量 `./...` 中根包生命周期超时、`internal/settings` 路径断言、`internal/lifecycle` 符号链接权限、`internal/terminal` cmd 转义等失败已在基线提交 8902234（无本次改动）上原样复现，属本机/并行负载环境预先存在问题，与本变更无关
- [x] 4.2 `cd frontend && npm test`（重点 `UpdateControl.test.tsx`）：`UpdateControl.test.tsx` 16/16 全过；`App.test.tsx` 2 个「填写 Git 仓库地址」用例在基线前端代码上同样失败（已在 8902234 复现），属预先存在的环境问题，与本变更无关

## 5. 集成测试（wails dev + 浏览器，先冒烟后专项）

- [x] 5.1 冒烟：以 `wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 启动（不设置 `TASKAI_UPDATE_TEST_URL`，不禁用终端颜色，不让进程自动退出），从输出获取调试地址；用浏览器（mcp chrome-devtools）访问，确认应用正常加载、任务列表可用
- [x] 5.2 专项—无更新显示当前版本：同一会话中断言顶栏"任务工作台"右侧出现 `v0.0.0-rc5` 静态文本（更新服务为 nil 的路径）；元素为纯文本而非按钮
- [x] 5.3 专项—版本来自编译注入：重启为 `-ldflags "-X main.appVersion=v0.0.0-rc4"`，确认显示变为 `v0.0.0-rc4`（证明非硬编码）
- [x] 5.4 专项—有更新时按钮替换版本：启动 `go run ./cmd/update-test-server` 记录 `TASKAI_UPDATE_TEST_URL`，以 `TASKAI_UPDATE_TEST_URL=http://127.0.0.1:<端口> wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 启动；等待检查完成后断言：出现 `new` 更新入口且版本文本消失（互斥）
- [x] 5.5 专项—版本号不可交互：在 5.2 的显示状态下点击/悬浮版本文本，断言不触发任何对话框、导航或下载状态变化
- [x] 5.6 关闭 wails dev 与测试服务器，清理临时进程

## 6. 完整验证

- [x] 6.1 `go test -race ./...` 与 `go test -tags updater_integration ./...`：两次全量均无 DATA RACE，`internal/updater` 在 race 下全绿（4.9s），updater_integration 标签下集成链路无新增失败。失败集合与 4.1 记录相同（根包生命周期时序抖动、`internal/lifecycle`/`internal/appdata` 符号链接权限「A required privilege is not held by the client」、`internal/settings` 路径断言、`internal/terminal` cmd 转义），均已在基线 8902234 复现或属同类环境问题，与本变更无关
- [x] 6.2 `cd frontend && npm test && npm run build`：`npm run build` 全绿（tsc + vite，exit 0，dist 产物正常）；`npm test` 361/362，唯一失败为 `App.test.tsx`「填写 Git 仓库地址不会覆盖已有项目名称」——已在基线 8902234 复现的预先存在问题（本轮仅 1 个失败、上轮 2 个，随负载波动），`UpdateControl.test.tsx` 全过

## 7. 构建与确认

- [x] 7.1 使用 `scripts/` 下当前平台正式构建脚本编译可执行程序并打开程序（确认未启用 `updater_integration`、版本注入有效）：已用 `scripts/build-windows.ps1 -Version v0.0.0-rc99` 构建出 `build\bin\taskai.exe`（2m0s，14.1MB）；二进制含 `v0.0.0-rc99`、不含 `TASKAI_UPDATE_TEST_URL`（未启用 updater_integration）；用户关闭阻塞锁的并行实例后已成功启动（PID 32608，与 `E:\taskai` 实例共存）
- [x] 7.2 在编译产物中确认顶栏显示的版本号与注入版本一致：用户确认顶栏「任务工作台」右侧显示 `v0.0.0-rc99` 静态文本，无误

## 8. 合并与收尾

- [x] 8.1 先将工作区主分支反向合并到 `update-control-version-display` 分支，出现冲突先解决：合并 8aad076（含 fix-windows-cmd-dropped-path-escaping 归档），ort 策略无冲突（5393aae）
- [x] 8.2 确认无误后将 `update-control-version-display` 合并回工作区对应分支：`--no-ff` 合并进 main（306b82d），未触碰其他会话未提交文件
- [x] 8.3 合并后重新编译项目验证合并无问题：主工作区 `scripts/build-windows.ps1` 构建成功（exit 0）
- [x] 8.4 归档 openspec 变更并同步 `openspec/specs/application-auto-update/spec.md`；按流程要求同步 `docs/plans/` 实施记录：变更归档至 `openspec/changes/archive/2026-08-16-update-control-version-display/`，新增实施记录 `docs/plans/2026-08-16-update-control-version-display-implementation-plan.md`
- [x] 8.5 提交 git 变更并移除已合并的 worktree
