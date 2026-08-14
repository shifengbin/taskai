# Windows Release NSIS 构建修复设计

## 背景

`v0.0.0-rc6` 发布工作流中，Linux 与 macOS 构建成功，Windows 构建在上传安装包时失败。Wails v2.12.0 在找不到 `makensis` 时只输出警告并结束安装包生成，不会让 `wails build -nsis` 返回失败，因此工作流继续执行，但 `build/bin/*-installer.exe` 不存在。

`v0.0.0-rc7` 首次修复已通过 Chocolatey 将 NSIS 安装到 `C:\Program Files (x86)\NSIS`，但同一 PowerShell 步骤的 PATH 不会自动刷新，直接执行 `Get-Command makensis` 仍然失败。

## 方案

Windows 构建任务在运行项目测试与构建之前，通过 Chocolatey 安装 NSIS。任务使用标准安装目录中的绝对路径验证 `makensis.exe`，随后将该目录写入 GitHub Actions 的 `GITHUB_PATH`，使后续 Wails 构建步骤能够发现命令。验证失败时任务直接失败，不进入会产生误导性成功状态的 Wails 构建步骤。

发布工作流测试增加静态回归断言，要求 Windows 条件步骤同时包含 NSIS 安装和 `makensis` 可用性验证。项目不放宽安装包资产匹配规则，也不退回上传裸 `taskai.exe`。

## 发布策略

保留已经推送但发布失败的 `v0.0.0-rc6` 与 `v0.0.0-rc7` 标签，避免重写公共 Git 引用。PATH 修复提交合并并推送到 `main` 后创建 `v0.0.0-rc8`，由标签触发三端构建、更新清单生成和 GitHub Prerelease 创建。

## 验证

先让新增回归断言在未修改工作流时失败，再加入最小工作流修复并确认测试通过。随后运行 Go race 测试、前端测试与构建、发布脚本测试和 OpenSpec 严格校验。推送 `v0.0.0-rc8` 后持续检查 GitHub Actions，最终核对 Release 为 Prerelease，并确认包含 DEB、NSIS、DMG 与 `taskai-update.json` 四类资产。
