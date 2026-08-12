## Why

目前三端编译脚本（`scripts/build-linux.sh`、`scripts/build-windows.ps1`、`scripts/build-macos.sh`）只能由开发者在本地、分别在对应平台主机上手动执行，缺少一次性产出三端发布产物的自动化方式。其中 macOS 脚本只产出 `.app`，缺少 macOS 标准的 `.dmg` 交付物；Linux 与 Windows 脚本虽已自带打包（`.deb` 与 NSIS 安装程序），但没有任何 CI 把它们串起来。

需要一条 GitHub Actions 管线，在打 tag 或手动触发时，于三个原生 runner 上并行编译，自动产出 Linux `.deb`、Windows `.exe`（NSIS 安装程序）与 macOS `.dmg`，并挂载到同一个 GitHub Release。

## What Changes

- 新增 GitHub Actions 工作流 `build-release.yml`：在推送 `v*` tag 或手动触发（`workflow_dispatch`）时运行。
- 工作流以三 job 矩阵在 `ubuntu-latest`、`windows-latest`、`macos-latest` 三个原生 runner 上并行编译。
- Linux job 安装 `libwebkit2gtk-4.1-dev`、`libgtk-3-dev`、`dpkg-dev` 等系统依赖后，调用 `build-linux.sh amd64 --deb --version <版本>`。
- 为 `scripts/build-windows.ps1` 增加可选 `-Version` 参数（写入 `wails.json`）；Windows job 以 `build-windows.ps1 -NSIS -Version <版本>` 产出安装程序。
- macOS job 调用扩展后的 `build-macos.sh --dmg --version <版本>` 产出 `.dmg`。
- 三端产物先用 `upload-artifact` 上传；仅 tag 触发时再下载三端产物并挂载到同一 GitHub Release。
- 版本号统一从 git tag 派生（剥掉 `v` 前缀）；无 tag 时回落到 `0.0.0+git.<sha>`。
- 扩展 `scripts/build-macos.sh`：新增 `--dmg` 选项与 `--version`（及 `TASKAI_VERSION` 环境变量）入参，产出 `.dmg`，使用系统自带的 `hdiutil`，不引入额外打包依赖。
- 本期不做代码签名与公证：接受 Windows SmartScreen 与 macOS Gatekeeper 的首次运行警告。

## Capabilities

### New Capabilities

- `automated-release-pipeline`：定义在 tag 推送或手动触发时，于三个原生 runner 上并行构建三端发布产物，并按 tag 派生统一版本号、上传产物、在 tag 触发时发布 GitHub Release 的行为。

### Modified Capabilities

- `platform-release-packaging`：在既有 Linux DEB 与 Windows 图标规则之外，新增 macOS 构建脚本的 `.dmg` 产出规则与 `--version` 入参，使三端脚本在「编译 + 用平台原生工具打包 + 接收版本号」三件事上保持对称。

## Impact

- 新增 `.github/workflows/build-release.yml`。
- 修改 `scripts/build-macos.sh`（增加 `--dmg`、`--version` / `TASKAI_VERSION` 与 `hdiutil` 打包步骤，并将 `-Platform` 校正为 `-platform`）；新增 `scripts/build-macos.test.sh` 集成测试。
- `scripts/build-windows.ps1` 增加可选 `-Version` 参数（写入 `wails.json` 的 `version` 字段），属向后兼容的增量；`scripts/build-linux.sh` 既有行为不变，仅由 CI 调用。
- 不改动应用代码、Wails 绑定、生命周期命令或任何运行时行为。
- 不引入代码签名证书或 Apple 公证配置；签名留作后续独立变更。
