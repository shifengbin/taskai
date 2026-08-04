## 为什么

Wails 会复用已存在的 Windows `.ico` 文件，导致更新 `build/appicon.png` 后生成的可执行文件和 NSIS 安装程序可能仍显示旧图标。Linux 当前仅生成裸二进制，缺少可由 Debian 系发行版安装并在桌面应用列表中发现的发行包。当前 `build/appicon.png` 是没有 alpha 通道的 RGB 图像，DEB 直接复制它后会在圆角外侧显示不透明背景。

## 变更内容

- 将 `build/appicon.png` 确立为 Windows 构建使用的唯一图标来源，并在每次 Windows 构建前强制重新生成 `build/windows/icon.ico`，使 EXE 与 NSIS 安装程序使用当前图标。
- 扩展 Linux 构建脚本，保持原有二进制构建行为，并可选择额外生成指定架构的 `.deb` 包。
- 在 DEB 包中安装应用二进制、命令行启动入口、桌面启动器和最新应用图标，使安装后可从桌面环境启动 TaskAI。
- 将当前应用图标重新导出为保留圆角外侧透明像素的 RGBA PNG，避免 DEB 安装后的桌面图标显示为方形背景。
- 为 DEB 打包校验必要工具、规范化架构名和 Debian 版本号，并由构建环境解析运行时库依赖。
- 为任务右键操作菜单中的“搁置任务”和“取消搁置”提供与现有菜单风格一致的语义图标。
- 更新项目发行构建文档，说明 Windows 图标同步行为、Linux DEB 构建命令、产物位置及新增前置依赖。

## 能力

### 新增能力

- `platform-release-packaging`：保证 Windows 发布产物使用当前、保留透明圆角的应用图标，并支持生成可安装的 Linux Debian 发行包。
- `task-menu-action-icons`：为搁置与取消搁置任务操作提供明确且成对的菜单图标。

### 修改能力

无。

## 影响

- 受影响文件：`build/appicon.png`、`scripts/build-windows.ps1`、`scripts/build-linux.sh`、`README.md`、`frontend/src/components/TaskTree.tsx` 及其测试，以及新增的 Linux DEB 打包资源或模板。
- Windows 构建会重新写入受版本控制的 `build/windows/icon.ico`，其内容始终由 `build/appicon.png` 派生。
- Linux DEB 打包新增对 `dpkg-deb` 和 `dpkg-shlibdeps` 的依赖；普通 Linux 二进制构建不受影响。
- 不涉及应用运行时 API、任务数据或前端交互的破坏性变更；搁置操作仅增加视觉标识，不改变既有切换语义。
