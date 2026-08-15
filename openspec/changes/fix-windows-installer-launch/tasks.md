# Tasks: fix-windows-installer-launch

## 1. 失败测试先行

- [x] 1.1 在 `internal/updater` 新增 Windows 单元测试：`DefaultSystemLauncher().LaunchInstaller(<合法 .exe 路径>)` 必须通过 `ShellExecute` 以 `open` 动词启动（替换可注入的 `shellExecute` 接缝断言参数），不得走 `CreateProcess`；先运行确认失败
- [x] 1.2 保留既有测试通过：Release 页面启动仍为 rundll32 后台进程（隐藏控制台）；`InstallerInvocation` 的扩展名校验不变

## 2. 实现

- [x] 2.1 新增 `internal/updater/launcher_windows.go`（`//go:build windows`）：`startInstallerDetached` 使用 `ShellExecuteW`（`open` 动词、安装包所在目录、`SW_SHOWNORMAL`），`shellExecute` 为包级变量接缝
- [x] 2.2 新增 `internal/updater/launcher_other.go`（`//go:build !windows`）：`startInstallerDetached` 复用 `startDetached`
- [x] 2.3 `internal/updater/launcher.go`：`SystemLauncher` 增加 `startInstaller` 字段并由 `NewSystemLauncher` 注入；`LaunchInstaller` 改用它；`OpenReleasePage` 行为不变
- [x] 2.4 `go test ./internal/updater/...` 全绿（含新测试）
- [x] 2.5 扩展 `cmd/update-test-server`：清单与安装包资产按运行平台提供（原 linux-amd64 .deb 行为保持不变），使 Windows/macOS 也能走完集成发现与下载流程

## 3. 真机机制验证（不触碰真实安装器）

- [x] 3.1 编译一个 `-H=windowsgui` 的临时哑 exe（运行后写标记文件并退出），通过修复后的启动路径 `ShellExecute` 打开它，断言标记文件出现——证明动词/参数/工作目录调用序列在真实 ShellExecute 下可用且无控制台窗口
- [x] 3.2 删除所有临时探针/哑 exe，确认工作区无残留

## 4. 集成测试（wails dev + 浏览器）

- [x] 4.1 启动 `cmd/update-test-server`，记录其输出的 `TASKAI_UPDATE_TEST_URL`
- [x] 4.2 以 `TASKAI_UPDATE_TEST_URL=http://127.0.0.1:18971 wails dev -tags updater_integration -ldflags "-X main.appVersion=v0.0.0-rc5"` 启动应用（不禁用终端颜色，不让进程自动退出）；从输出获取调试地址 http://localhost:34115
- [x] 4.3 用浏览器（mcp chrome-devtools 不可用时以 puppeteer-core + Edge 等价替代）访问调试地址，依次验证：入口出现 `new` → 开始下载 → 首次 503 进入下载失败（错误文案可见）→ 重新下载成功且安装包为 windows `.exe` → `LaunchDownloadedUpdate`（集成桩）无错误 → 服务器安装包请求计数恰好 2。9 项断言 8 项通过，唯一失败为已知的 `/favicon.ico` 404（与本次改动无关）
- [x] 4.4 关闭 wails dev 与测试服务器

## 5. 构建与确认

- [x] 5.1 使用仓库正式 Windows 构建脚本编译可执行程序并启动，确认未启用 `updater_integration`
- [x] 5.2 用户已在真实应用中确认：下载 → 立即安装 → UAC 提升与 NSIS 向导正常出现（2026-08-15）

## 6. 收尾

- [ ] 6.1 合并 `fix-windows-installer-launch` 到工作区主分支（先反向合并主分支解决冲突）
- [ ] 6.2 合并后重新编译验证
- [ ] 6.3 归档 openspec 变更并同步规格
- [ ] 6.4 提交 git 变更并移除 worktree
- [ ] 6.5 发布新版本（含本修复），使 v0.0.1 用户可通过自动更新升级
