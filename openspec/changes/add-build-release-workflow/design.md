## Context

TaskAI 是 Wails v2 桌面应用，后端 Go 1.23、前端 React + Vite。Wails 通过 CGO 绑定各平台的原生 webview C 库：Linux 依赖 `webkit2gtk`、Windows 依赖 WebView2、macOS 依赖系统 WebKit。三个编译脚本各自校验宿主操作系统（`uname -s`、`$env:OS`），且都需要平台原生的 C 工具链与系统库，因此跨平台交叉编译在本项目不可行。

现有脚本的能力不对称：`build-linux.sh` 自带完整的 DEB 打包（`dpkg-deb`、`dpkg-shlibdeps` 自动依赖、桌面入口、图标层级）；`build-windows.ps1` 通过 `-NSIS` 开关产出安装程序；`build-macos.sh` 只运行 `wails build` 停在 `.app`，没有 `.dmg`。版本号方面，只有 Linux 脚本接受 `--version` / `TASKAI_DEB_VERSION`。

Linux 构建还会对 Wails v2.12.0 应用一个文件拖放补丁（`prepare-wails-linux-file-drop-patch.sh`），该补丁只支持 Wails v2.12.0；`go.mod` 恰好锁定该版本。

## Goals / Non-Goals

**Goals:**

- 在 tag 推送或手动触发时，自动在三个原生 runner 上并行产出 `.deb`、`.exe`（NSIS）与 `.dmg`。
- 让 `build-macos.sh` 与另外两个脚本对称：自行产出 `.dmg`，并接受统一版本号。
- 版本号以 git tag 为唯一来源，三端共用。
- 保持本地与 CI 行为一致：开发者在本机跑同一脚本即可得到相同产物。

**Non-Goals:**

- 不做 Windows 代码签名与 macOS 公证；接受 SmartScreen 与 Gatekeeper 的首次运行警告。
- 不改动 Linux / Windows 脚本的既有打包行为。
- 不支持单 runner 交叉编译或容器化产出异平台包。
- 不升级 Wails 版本（保持 v2.12.0 以兼容 Linux 文件拖放补丁）。
- 不改变产物之外的应用运行时行为。

## Decisions

### 采用三原生 runner 矩阵，而非交叉编译

工作流以 `ubuntu-latest`、`windows-latest`、`macos-latest` 三个 job 并行，每个 job 只构建本平台产物。Wails 的 webview CGO 绑定要求平台原生 C 编译器与系统库，Linux 上无法为 Windows/macOS 链接 WebView2 或 macOS WebKit；用 Docker 也无法在 Linux 容器内产出原生 Windows 或 macOS 包。因此三原生 runner 是唯一可行路径。

### macOS 的 `.dmg` 由 `build-macos.sh` 用系统 `hdiutil` 产出

`wails build` 产出 `build/bin/taskai.app` 后，脚本用系统自带的 `hdiutil` 创建可读写 dmg、放入 `.app` 与一个 `/Applications` 软链接（拖拽安装的标准体验），再用 `hdiutil convert` 压成只读 UDZO dmg，命名为 `TaskAI-<版本>-<架构>.dmg`。这与 `build-linux.sh` 使用 `dpkg-deb` 的「平台原生打包工具」哲学一致，且不引入 `brew install create-dmg` 之类的额外依赖，本地与 CI 行为完全相同。

把 dmg 逻辑放在脚本里、而非只在 CI 中加一个 `create-dmg` Action，是为了让三个脚本对称：本地开发者一条命令即可复现 dmg；只在 CI 打包会造成本地与 CI 行为分叉，不采用。

### 版本号以 git tag 为唯一来源

工作流从触发 tag 剥掉 `v` 前缀得到版本号，三端共用：Linux 走既有 `--version`，macOS 走新增 `--dmg --version`，Windows 走新增的 `-Version` 参数（写入 `wails.json` 的 `version` 字段，供 NSIS 安装程序与二进制元数据使用）。版本号合法性复用 Linux 脚本既有的 `dpkg --validate-version` 校验。手动触发（无 tag）时回落到 `0.0.0+git.<sha>`，与 Linux 脚本既有的本地默认一致。

### 三端产物先 upload-artifact，仅 tag 触发时再发布 Release

每次运行（包括手动触发）都用 `upload-artifact` 保存三端产物，便于在不发版时也能取回构建结果。仅在 `push.tags v*` 触发时，增加一个依赖三构建 job 的发布 job，下载三端 artifact 并挂载到 GitHub Release。这样「构建」与「发布」解耦，手动触发不会意外产生公开 Release。

### 本期显式不签名

代码签名（Windows）与公证（macOS）需要证书与 Apple Developer 凭据，且会显著扩大配置与密钥管理范围。本期把「跑通三端自动构建与发布」作为目标，签名作为后续独立变更。文档与规格中明确记录 SmartScreen 与 Gatekeeper 的用户侧后果。

## Risks / Trade-offs

- [macOS runner 上 `hdiutil` 偶发设备忙] → 复用既有临时目录清理模式（`trap cleanup EXIT`），失败即非零退出，不产出半成品 dmg。
- [GitHub 托管镜像的 WebView2 / Xcode CLT 版本漂移] → 不锁死版本，依赖镜像自带；若上游破坏构建，记录在工作流注释中并跟进。
- [tag 不符合版本格式] → 复用 `dpkg --validate-version` 校验，非法版本使整条管线失败而非产出错误命名的产物。
- [Wails 上游升级破坏 v2.12.0 文件拖放补丁] → 本期锁定 v2.12.0；升级另开变更并同步更新补丁。
- [不签名影响用户首次运行体验] → 在 Release 说明中给出 macOS `xattr -dr com.apple.quarantine` 与 Windows「仍要运行」的放行步骤。

## Migration Plan

无需迁移配置或数据。合并并打首个 `v*` tag 后即触发管线；在此之前开发者仍可手动运行各平台脚本，行为与现状一致（macOS 脚本扩展后默认行为仍产出 `.app`，`.dmg` 仅在传入 `--dmg` 时产出）。

## Open Questions

- 手动触发（非 tag）是否也需要可选地创建 Release？当前默认只 `upload-artifact`，不发 Release。
