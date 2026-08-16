# 更新入口位置版本号显示实施记录

## 目标

无更新时在“任务工作台”标题右侧的更新入口位置显示当前应用版本号（如 `v0.0.0-rc5`），有更新时保持原有 `new` 更新入口不变——同一位置二选一互斥显示，版本号为纯展示文本、不可交互。

## 实现

版本来源选择编译注入的 `main.appVersion`（`-ldflags "-X main.appVersion=..."`，开发构建默认 `v0.0.0-dev`），不新增 Wails 绑定：

- `internal/updater/service.go`：`State` 结构体新增 `CurrentVersion string`（`json:"currentVersion,omitempty"`）。`State()` getter 与 `publishState()` 两个出口均以服务持有的规范化 `currentVersion` 填充——事件是前端主要更新通道，两个出口都必须带上当前版本。
- `app.go`：`GetUpdateState` 的 `updaterService == nil` 分支（无更新服务的平台/配置）返回 `State{Status: StatusIdle, CurrentVersion: appVersion}`。
- `frontend/src/types.ts`：`UpdateState` 增加可选字段 `currentVersion?: string`（可选以兼容旧绑定优雅降级）。
- `frontend/src/components/UpdateControl.tsx`：`state.status === 'idle'` 且 `currentVersion` 非空时渲染静态 `<span data-testid="app-version">`（小号、弱化颜色 `text-snap-ink/60`、自然宽度）；非 idle 分支的按钮渲染完全不变；兜底 `idleState` 常量不加版本字段。
- `wails generate module` 再生成 `frontend/wailsjs/go/models.ts`，仅新增 `currentVersion` 可选字段。

## 验证

- 单测：`internal/updater`（`TestServiceStateCarriesCurrentVersion`：构造注入 `1.0.0` 规范化为 `v1.0.0`，发布 available 状态同时携带目标 `v1.1.0` 与当前 `v1.0.0`）、根包（`TestGetUpdateStateWithoutUpdaterCarriesCurrentVersion`）、前端 `UpdateControl.test.tsx` 16/16（idle 渲染静态版本、缺失时不渲染、available 时按钮替换版本）。
- 集成测试（`wails dev -tags updater_integration` + 浏览器，隔离 devserver 端口避免并行会话冲突）：
  - idle 显示注入版本 `v0.0.0-rc5`，为 SPAN 纯文本而非按钮；
  - 重启注入 `v0.0.0-rc4` 后显示随之变为 `v0.0.0-rc4`（证明版本来自编译注入而非硬编码）；
  - 接 `cmd/update-test-server` 后检查发现 `v0.0.0-rc6`：出现 `new` 更新入口且版本文本消失（互斥成立），绑定状态同时含 `version: v0.0.0-rc6` 与 `currentVersion: v0.0.0-rc5`；
  - 点击/悬浮版本号不触发对话框、导航或状态变化（不可交互）。
- 正式构建：`scripts/build-windows.ps1 -Version v0.0.0-rc99` 产物含注入版本、无 `TASKAI_UPDATE_TEST_URL` 标记（未启用 updater_integration）；启动后用户确认顶栏显示 `v0.0.0-rc99`。

## 已知非回归问题

全量测试在本机存在预先失败（已在基线 8902234 原样复现，与本变更无关）：根包生命周期链用例在并行负载下偶发等待超时、`internal/lifecycle`/`internal/appdata` 符号链接用例因 Windows 未开启开发者模式权限失败、`internal/settings` 路径分隔符断言、`internal/terminal` cmd 转义断言（该修复已由并行变更 fix-windows-cmd-dropped-path-escaping 处理）；前端 `App.test.tsx` 两个「填写 Git 仓库地址」用例同样在基线复现。race 全量无 DATA RACE。

设计决策与规格 delta 详见归档变更 `openspec/changes/archive/2026-08-16-update-control-version-display/`。
