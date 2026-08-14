## Why

TaskAI 已能通过 GitHub Release 分发三端安装包，但应用无法发现新版本，用户也不能在任务工作台内完成下载与安装引导。现在需要把既有发布产物接入运行时更新流程，并在不破坏任务和终端退出语义的前提下提供可校验、可恢复的更新体验。

## What Changes

- 三端发布构建向运行中的应用注入统一语义版本，并让带预发布后缀的 tag 在 GitHub 上标记为 Prerelease。
- GitHub Release 增加 `taskai-update.json`，声明目标版本、Release 页面和各平台安装包的名称、大小与 SHA-256。
- 新增应用更新服务：启动后检查一次，此后每小时检查；包含正式版和预发布版本，忽略 Draft 和无匹配平台资产的版本。
- 在“任务工作台”标题右侧增加紧凑更新入口，展示 `new`、下载中、下载失败和已下载状态。
- 用户点击后将安装包下载到独立更新缓存，以临时文件写入并校验大小与 SHA-256；重启后重新校验已经下载的安装包。
- 下载失败时提供重试和手动打开对应 GitHub Release 页面；成功后提示立即安装或稍后安装。
- 安装时按平台启动 NSIS、DMG 或 DEB；存在执行中任务时复用现有退出确认，启动安装程序成功后才关闭终端并退出 TaskAI。
- 增加仅用于 `wails dev` 集成测试的更新源替换能力，生产构建继续固定访问官方仓库。

## Capabilities

### New Capabilities

- `application-auto-update`：定义新版本发现、发布清单、更新状态界面、安装包下载校验、缓存恢复、手动下载以及安装退出行为。

### Modified Capabilities

- `platform-release-packaging`：三端发布构建除安装包元数据外，还必须向运行中的应用注入与发布 tag 一致的版本，供更新比较使用。

## Impact

- 新增独立 Go 更新服务及平台安装程序启动实现，并接入 Wails 启动、关闭和事件发布流程。
- 增加 Wails 导出方法、生成绑定、前端 API 类型与任务工作台头部状态组件。
- 修改 `.github/workflows/build-release.yml` 和三端构建脚本，统一注入应用版本并生成 Release 更新清单。
- 在应用数据目录新增独立更新缓存，不改变任务和设置 JSON 结构。
- 依赖当前 `add-build-release-workflow` 变更提供的三端 Release 产物；实施时需保留并扩展该工作流，不重复创建发布链路。
- 不新增代码签名、公证或自动替换可执行文件能力。
