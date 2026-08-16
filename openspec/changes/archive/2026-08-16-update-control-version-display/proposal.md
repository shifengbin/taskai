## Why

顶栏更新按钮的位置在无更新（idle）时完全空白，用户没有任何途径在界面中看到当前运行的版本号。排查问题、确认升级是否生效时，只能依赖外部手段（文件属性、日志）获知版本。

## What Changes

- 无更新（idle）时，在顶栏更新按钮的现有位置显示当前版本号（如 `v1.2.3`、开发模式下的 `v0.0.0-dev`），纯展示、不可交互。
- 有更新（available / downloading / downloaded / download_failed）时，维持现有更新按钮的显示与全部交互行为不变。
- 后端 `updater.State` 增加 `CurrentVersion` 字段，使前端通过现有更新状态通道获取当前版本；更新服务为空（平台不支持自动更新）时同样携带该字段。
- 版本显示随更新状态切换：idle 显示版本号，非 idle 显示更新按钮，两者互斥占用同一位置。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `application-auto-update`: 新增"无更新时在更新按钮位置显示当前版本号"的需求；现有更新按钮、状态推送与安装流程的需求不变。

## Impact

- Go：`internal/updater/service.go`（`State` 结构体与 `State()` 方法携带 `CurrentVersion`）、`app.go`（`GetUpdateState` 在更新服务为空时填充 `CurrentVersion`）。
- 前端：`frontend/src/types.ts`（`UpdateState` 增加可选字段）、`frontend/src/components/UpdateControl.tsx`（idle 分支渲染版本文本）。
- 绑定：`updater.State` 为 Wails 绑定序列化类型，需执行 `wails generate module` 并检查 `frontend/wailsjs/` 生成差异。
- 测试：`internal/updater` 与 `app_updater_test.go` 的状态断言、`UpdateControl.test.tsx` 增加 idle 显示版本的用例。
- 规格文档：`openspec/specs/application-auto-update/spec.md` 同步更新。
