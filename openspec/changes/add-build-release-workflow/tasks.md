## 1. 扩展 macOS 构建脚本产出 DMG

- [x] 1.1 为 `scripts/build-macos.sh` 增加 `--dmg` 选项与 `--version`（及 `TASKAI_VERSION` 环境变量）入参，参照 `build-linux.sh` 校验版本号格式；无版本时回落到 `0.0.0+git.<sha>`。
- [x] 1.2 `wails build` 成功后，用系统 `hdiutil` 将 `build/bin/taskai.app` 打成带 `/Applications` 软链接的只读 UDZO dmg，命名为 `TaskAI-<版本>-<架构>.dmg`，输出到 `build/bin`。
- [x] 1.3 为 DMG 打包逻辑补充脚本测试（参照 `build-linux.test.sh`），校验产物存在、dmg 可挂载、且包含 `.app` 与 `/Applications` 链接。

## 2. 新增 GitHub Actions 工作流

- [x] 2.1 新增 `.github/workflows/build-release.yml`，触发条件为 `push.tags: ['v*']` 与 `workflow_dispatch`。
- [x] 2.2 定义三 job 矩阵（`ubuntu-latest` / `windows-latest` / `macos-latest`），各自 `setup-go` 1.23、`npm ci`、`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`。
- [x] 2.3 Linux job 安装 `libgtk-3-dev`、`libwebkit2gtk-4.1-dev`、`dpkg-dev` 等系统依赖，再调用 `scripts/build-linux.sh amd64 --deb --version <版本>`。
- [x] 2.4 为 `scripts/build-windows.ps1` 增加可选 `-Version` 参数（写入 `wails.json` 的 `version`），并在 Windows job 中以 `-NSIS -Version <版本>` 调用。
- [x] 2.5 macOS job 调用扩展后的 `scripts/build-macos.sh --dmg --version <版本>`。
- [x] 2.6 每个 job 用 `actions/upload-artifact` 上传对应产物（deb / exe / dmg）。
- [x] 2.7 仅 tag 触发时增加发布 job：`needs` 三构建 job，下载三端 artifact 并挂载到 GitHub Release（`gh release create` 或 `softprops/action-gh-release`）。

## 3. 验证

- [ ] 3.1 在 macOS 本地运行扩展后的 `build-macos.sh`，确认产出 `.app` 与 `.dmg`，且 dmg 可挂载、含 `/Applications` 链接。
- [ ] 3.2 以 `workflow_dispatch` 手动触发一次管线，确认三 job 均成功且 artifact 齐全。
- [ ] 3.3 打一个测试用 `v*` tag，确认 Release 自动挂载 deb / exe / dmg 三端产物。
