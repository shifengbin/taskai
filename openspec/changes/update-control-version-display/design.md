## Context

顶栏布局为 `[logo] 任务工作台 [UpdateControl] [弹性空隙] [右侧图标]`，`UpdateControl`（`frontend/src/components/UpdateControl.tsx`）仅在 `state.status !== 'idle'` 时渲染更新按钮，idle 时该位置空白。

当前版本的事实来源是 Go 侧 `version.go` 的 `var appVersion = "v0.0.0-dev"`，编译时由构建脚本通过 `-X main.appVersion=$app_version` 注入，仅被 `updater.NewService(Options{CurrentVersion: ...})` 消费。前端没有获取当前版本的通道；`UpdateState.version` 语义是"目标版本"（要更新到的版本），下载失败等文案依赖该语义，不能挪用。

更新状态通过 `App.GetUpdateState()` 绑定与 `update:state` 事件（`onUpdateState`）推送到前端，`UpdateControl` 已在监听。`updater.State` 是运行时状态（不持久化），更新缓存元数据（`cacheMetadata`）是独立结构，互不影响。

`app.go` 的 `GetUpdateState` 在 `updaterService == nil` 时返回 `State{Status: StatusIdle}`（平台不支持自动更新、或集成模式下未设置测试源时会出现该分支）。

## Goals / Non-Goals

**Goals:**

- idle 时在更新按钮现有位置显示当前版本号（`v` 前缀格式，开发模式为 `v0.0.0-dev`）。
- 复用现有更新状态通道传递当前版本，不新增 Wails 绑定方法。
- 版本显示为纯展示文本，不引入任何交互。
- 非 idle 状态行为与现状完全一致。

**Non-Goals:**

- 版本号点击交互（打开 release 页等）——idle 时 `releaseUrl` 未知，做成可点击需硬编码仓库地址，暂不引入。
- 设置页或"关于"面板中的版本信息。
- 更新按钮存在时并排显示当前版本（探索时已否决的共存方案）。
- 改变更新检查节奏、状态机或安装流程。

## Decisions

### 1. 版本通过 `updater.State.CurrentVersion` 传递，而非新增绑定

**选择**：`updater.State` 增加字段 `CurrentVersion string \`json:"currentVersion,omitempty"\``，`service.State()` 与 `app.go` 的 nil 分支负责填充。

**理由**：`UpdateControl` 已监听该状态，零额外请求；`UpdateClient` 接口不变，`UpdateControl.test.tsx` 的现有 mock 不需要加方法；`updater_integration` 模式与生产模式走同一条通道。

**备选**：新增 `App.GetAppVersion()` 绑定——职责更单一，但需要新增绑定方法、`UpdateClient` 接口方法、全部 mock 更新与 `wails generate module`，改动面更大而用户可见效果相同。前端构建期注入（vite define）会造成 Go 与前端两份版本事实来源，发版时可能不同步，直接排除。

**语义代价的接受理由**：`State` 从"更新流程状态"扩展为"更新流程状态 + 当前版本"，但当前版本本就是更新判断的基准值（`semver.Compare(candidate.Version, service.currentVersion)`），放在同一结构中并不冲突。

### 2. 字段填充位置

- `updater.Service` 已持有规范化后的 `currentVersion`（`canonicalVersion` 保证 `v` 前缀），`State()` 返回时填充。
- `app.go` `GetUpdateState` 的 `updaterService == nil` 分支显式填充 `CurrentVersion: appVersion`（此处直接用包内变量，已是 `v0.0.0-dev` 格式，无需再规范化）。
- 集成/生产 `newApplicationUpdater` 构造路径无需改动，版本由 `Options.CurrentVersion` 进入服务。

### 3. 前端渲染与降级

- `frontend/src/types.ts` 的 `UpdateState` 增加可选字段 `currentVersion?: string`。
- `UpdateControl` 在 `state.status === 'idle'` 且 `currentVersion` 非空时渲染一段次级样式的静态文本（`data-testid="app-version"`，样式基调为小号、弱化颜色，与"任务工作台"标题形成层级），替代现有"什么都不渲染"。
- `state.status !== 'idle'` 时现有按钮渲染逻辑原样保留，版本文本不渲染。
- 兜底常量 `idleState = {status: 'idle'}`（初始请求失败时使用）不含版本，此时优雅降级为不显示，与现状一致，不报错。
- 无 Tooltip 包裹：文本本身即完整信息。

### 4. 布局处理

版本文本为自然宽度，不套用更新按钮的 `w-[86px]` 固定宽度。顶栏中间是弹性空隙（`flex-1`），右侧图标位置不受该文本宽度变化影响；idle 与非 idle 切换时"任务工作台"右侧内容整体变化，无对齐抖动问题。

## Risks / Trade-offs

- [风险] `updater.State` 语义膨胀，未来字段继续堆积 → 缓解：本次仅加一个与更新判断直接相关的字段；若后续出现更多"应用元信息"需求，再评估拆分独立绑定。
- [风险] `wails generate module` 生成的 TS 模型与手写 `types.ts` 不同步 → 缓解：任务中显式包含"重新生成绑定并检查 `git diff -- frontend/wailsjs/`"，以 Go 结构为唯一事实来源。
- [风险] 前端在旧状态事件（无 `currentVersion` 字段）下渲染空白 → 缓解：字段可选 + 空值降级不渲染，行为与现状一致。
- [权衡] idle 显示版本、非 idle 只显示按钮，升级过程中当前版本不可见 → 已在探索阶段与用户确认接受（升级完成后版本号更新即为反馈）。

## Migration Plan

纯增量字段与纯增量渲染分支，无数据迁移。回滚只需还原前端渲染分支；后端字段保留无害（`omitempty`，旧前端忽略未知字段）。

## Open Questions

（无——显示形态（纯展示）、来源（State 字段）、互斥关系（方案 A）均已在探索阶段确认。）
