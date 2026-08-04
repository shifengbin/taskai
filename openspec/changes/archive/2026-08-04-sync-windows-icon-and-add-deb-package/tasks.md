## 1. Windows 图标构建

- [x] 1.1 修改 `scripts/build-windows.ps1`，在调用 Wails 前删除已有的 `build/windows/icon.ico`，并保持架构、`-NSIS` 与构建失败处理的现有语义。
- [x] 1.2 在 Windows 主机上分别运行普通和 `-NSIS` 构建，确认生成的 EXE 与安装程序均使用由当前 `build/appicon.png` 重建的 ICO。

## 2. Linux Debian 打包

- [x] 2.1 扩展 `scripts/build-linux.sh` 的参数解析，保留现有架构位置参数，并增加 `--deb`、`--version`、`TASKAI_DEB_VERSION` 和 Debian 版本号校验。
- [x] 2.2 仅在 `--deb` 模式校验 `dpkg-deb`、`dpkg-shlibdeps`，规范化 Debian 架构名，并在 Wails 二进制构建成功后创建可清理的 DEB 暂存目录。
- [x] 2.3 生成 DEB 控制信息和运行时依赖，将二进制、`taskai` 启动入口、桌面启动器及当前应用图标安装到约定路径，并输出 `build/bin/taskai_<版本>_<架构>.deb`。

## 3. 搁置任务菜单图标

- [x] 3.1 在 `frontend/src/components/TaskTree.tsx` 中为“搁置任务”使用 `ArchiveOutlined`，为“取消搁置”使用 `UnarchiveOutlined`，并复用既有菜单图标与文本布局。
- [x] 3.2 更新任务树前端测试，覆盖正常与已搁置执行中任务在右键和任务操作菜单中的图标、文本与既有切换回调。

## 4. 文档与验证

- [x] 4.1 更新 `README.md`，说明 Windows 图标自动刷新、Linux `--deb` 与版本参数、产物位置、DEB 工具依赖和安装检查命令。
- [x] 4.2 运行现有 Go 与前端验证、普通 Linux 构建及带显式版本的 `--deb` 构建；用 `dpkg-deb --info` 和 `dpkg-deb --contents` 检查 DEB 元数据和安装布局。
- [x] 4.3 运行 `openspec validate sync-windows-icon-and-add-deb-package --strict --no-interactive` 与 `git diff --check`，修复本变更引入的文档或脚本格式问题。

## 5. 发布图标透明圆角

- [x] 5.1 从现有 `frontend/src/assets/task-ai-mark.svg` 重新导出 `build/appicon.png`，保留现有图案与圆角比例，并使圆角外侧为透明 alpha 像素。
- [x] 5.2 扩展 `scripts/build-linux.test.sh`，验证源 PNG 及解包后 DEB 图标均含 alpha 通道，且左上角透明。
- [x] 5.3 运行带显式版本的真实 `--deb` 构建，检查包内图标的透明通道、角点像素和安装布局；更新 README 或 OpenSpec 验证记录。
