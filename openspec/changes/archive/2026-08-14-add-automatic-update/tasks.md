## 1. Worktree 与基线

- [x] 1.1 确认仓库根目录的 `.worktrees` 已被 Git 忽略，从当前工作区分支创建 `add-automatic-update` 功能分支及 `.worktrees/add-automatic-update` worktree，后续功能代码只在该 worktree 修改
- [x] 1.2 在功能 worktree 记录 `go test -race ./...`、`cd frontend && npm test`、`cd frontend && npm run build` 以及现有发布脚本测试的基线结果，基线失败时先区分既有问题与本变更问题
- [x] 1.3 核对 `add-build-release-workflow` 的最终三端产物名、版本参数和 Release 创建方式；若其尚未合入当前基线，先把所需发布链路变更纳入功能分支且不重复创建工作流

## 2. 发布版本与更新清单

- [x] 2.1 先增加失败测试，覆盖 tag 与开发构建的统一版本注入、RC/Beta Release 的 Prerelease 标记、三端清单字段及缺失或非法产物时失败
- [x] 2.2 在 `main` 包定义可由 linker flag 注入且具有明确开发默认值的 `appVersion`，让 Linux、Windows、macOS 构建脚本向可执行程序注入同一规范化版本，并让 Windows 构建临时设置 `info.productVersion` 后恢复原始 `wails.json`
- [x] 2.3 实现确定性的 `taskai-update.json` 生成工具或脚本，校验语义版本与三端资产存在性，并写入 schemaVersion、version、tag、releaseUrl、平台键、文件名、字节大小和 SHA-256
- [x] 2.4 扩展 `.github/workflows/build-release.yml`，在三端产物汇总后生成并上传更新清单，并根据 tag 的预发布后缀设置 GitHub Release 的 Prerelease 元数据
- [x] 2.5 运行发布脚本与工作流静态测试，使用伪造的 DEB、NSIS、DMG 断言清单大小和 SHA-256 与文件一致，同时确认普通三端构建及原有安装包命名没有回归

## 3. 更新发现与调度

- [x] 3.1 先为 `internal/updater` 增加失败测试，覆盖正式版、RC、Beta 的语义版本顺序，以及 Draft、非法版本、旧版本、无当前平台资产和清单元数据不一致的过滤
- [x] 3.2 实现 Release/Manifest/Asset/State 类型、当前平台键映射和分页读取全部结果的官方 GitHub Release 客户端，只允许同一官方 Release 中与清单匹配的资产成为下载候选
- [x] 3.3 先增加失败测试，覆盖启动检查、每小时调度、检查超时或失败后的下周期重试、并发检查串行化以及停止服务后清理计时器与请求
- [x] 3.4 实现更新服务的启动、停止与检查状态机：启动后异步检查一次，此后每小时检查，自动检查错误保持 `idle` 或当前可用状态且不进入 `download_failed`

## 4. 下载、校验与缓存恢复

- [x] 4.1 先增加失败测试，覆盖 HTTP 状态错误、响应中断、大小不符、SHA-256 不符、重复下载、`.part` 清理和成功后的原子提交
- [x] 4.2 实现单下载写入器、流式 SHA-256 和大小校验，仅在全部校验成功后将 `.part` 原子重命名并发布 `downloaded` 状态
- [x] 4.3 先增加失败测试，覆盖离线重启后有效缓存恢复、损坏缓存删除、`.part`、当前或旧版本、其他平台缓存清理，以及每次恢复都重新计算摘要
- [x] 4.4 实现应用数据目录 `updates/<version>/` 缓存与元数据管理，保持任务和设置 JSON 不变，并仅保留最高目标版本的有效安装包

## 5. 平台安装与应用层接入

- [x] 5.1 先增加平台启动器测试，断言 Windows 直接启动 NSIS 且使用无控制台配置、macOS 使用参数化 `open`、Linux 使用参数化 `xdg-open`，路径均不经过 Shell 拼接
- [x] 5.2 实现三端安装包启动器和默认浏览器 Release 页面打开器；启动失败必须返回错误且不得关闭终端或应用
- [x] 5.3 先增加 `App` 层失败测试，覆盖状态查询、事件发布、开始下载、手动下载、启动安装，以及安装程序成功前不调用 `PrepareQuit` 的边界
- [x] 5.4 将更新服务接入 `App` 初始化、Wails `startup`/`shutdown` 和 `updater:state-changed` 事件，导出查询状态、开始下载、启动已下载更新和打开目标 Release 页面的窄接口
- [x] 5.5 增加 `updater_integration` build tag 的更新源与安装启动器替换；生产实现固定官方仓库并通过测试证明即使设置 `TASKAI_UPDATE_TEST_URL` 也不会改变生产更新源
- [x] 5.6 运行 `wails generate module` 重新生成 Wails JavaScript/TypeScript 绑定，并检查生成 API 与 Go 导出方法、更新状态类型保持一致

## 6. 任务工作台更新交互

- [x] 6.1 先增加前端失败测试，覆盖初始状态查询、事件订阅和卸载清理，以及无更新、`new`、下载中、下载失败、已下载五种展示状态
- [x] 6.2 在前端 API 和类型层接入生成的更新方法与 `updater:state-changed` 事件，通过查询、立即订阅、再次查询和事件序号握手，确保启动检查早于订阅或发生在查询窗口内时都能恢复最新状态
- [x] 6.3 在“任务工作台”标题右侧实现固定尺寸的紧凑更新入口，提供目标版本 tooltip，下载中禁用重复点击且不以百分比改变头部宽度
- [x] 6.4 先增加前端失败交互测试，覆盖下载失败后的重新下载、手动下载和取消，下载成功立即确认、选择稍后后再次打开，以及打开 Release 页面失败的普通错误提示
- [x] 6.5 实现下载失败与安装确认对话框，使手动下载只打开本次目标版本 Release 页面，并在稍后安装或取消后保留已下载缓存状态
- [x] 6.6 先增加前端失败测试，覆盖有无执行中任务的安装路径、取消后保持运行，以及安装启动失败时不调用 `prepareQuit`/`quit`
- [x] 6.7 复用现有“仍有执行中的任务”确认语义并加入“关闭终端并安装”动作；确认后严格按启动安装程序、`PrepareQuit`、Wails `Quit` 的顺序执行

## 7. 自动化与集成测试

- [x] 7.1 将当前工作区对应分支的最新提交合并到功能 worktree 分支，解决冲突后重新运行更新服务、App、前端和发布脚本的相关测试
- [x] 7.2 实现本地更新测试服务器：提供高于测试当前版本的 Release、匹配清单和安装包，第一次安装包请求失败、第二次返回大小与 SHA-256 正确的内容，并记录请求次数供断言
- [x] 7.3 启动本地测试服务器，并使用 `TASKAI_UPDATE_TEST_URL=http://127.0.0.1:<端口> wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 持续运行应用；不得设置 `NO_COLOR` 等禁用终端颜色的变量，也不得让进程自动退出
- [x] 7.4 从 `wails dev` 输出取得调试地址，使用 `mcp chrome-devtools` 验证启动出现含 `v0.0.0-rc6` 提示的 `new`，点击后进入下载中，首次失败显示下载失败及“重新下载”“手动下载”，重试成功后出现安装确认
- [x] 7.5 在同一浏览器测试中选择稍后并确认入口变为已下载，刷新页面确认状态保持，再次点击确认安装对话框可重开；同时检查控制台无新增业务错误、头部无重叠或布局跳动
- [x] 7.6 完成浏览器验证后关闭 `wails dev` 与本地测试服务器，确认没有遗留调试或测试服务进程
- [x] 7.7 执行完整验证：`go test -race ./...`、`cd frontend && npm test`、`cd frontend && npm run build`，并运行 Linux、Windows、macOS 发布脚本测试和更新清单测试

## 8. 可执行程序与人工确认

- [x] 8.1 使用仓库当前平台的正式构建方式编译可执行程序，确认未启用 `updater_integration`、版本注入有效且生产更新源固定为官方仓库
- [x] 8.2 打开已编译的 TaskAI 且保持运行，检查任务工作台、现有任务与终端基础功能正常，然后等待用户确认，不在确认前合并功能分支

## 9. 合并、复验与归档

- [x] 9.1 用户确认后，将功能 worktree 分支合并到当前工作区项目对应分支；发生冲突时保留双方有效变更并完成冲突测试
- [x] 9.2 在合并后的当前工作区重新执行 `go test -race ./...`、前端测试与构建、发布脚本测试，并重新编译当前平台可执行程序验证合并结果
- [x] 9.3 同步 README、设计记录和必要的发布说明，核对 OpenSpec 需求与任务均已满足后归档 `add-automatic-update` 变更
- [x] 9.4 提交合并、实现、测试与文档变更，确认工作区干净且提交包含归档结果
- [x] 9.5 移除已经合并的 `.worktrees/add-automatic-update` worktree，并清理对应已合并功能分支
