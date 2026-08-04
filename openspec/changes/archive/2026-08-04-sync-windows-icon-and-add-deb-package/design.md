## 背景

项目使用 Wails v2.10.2 构建桌面应用。该版本的 Windows 打包逻辑仅在 `build/windows/icon.ico` 缺失时，才会从 `build/appicon.png` 生成多分辨率 ICO；`-clean` 只清理 `build/bin`，不会刷新已有 ICO。NSIS 安装程序同样直接引用该 ICO。Linux 打包逻辑为空，现有 `scripts/build-linux.sh` 仅产出 `build/bin/taskai`。

项目已经约定在目标 Linux 主机上构建 Wails 应用。当前构建环境提供 `dpkg-deb`、`dpkg-shlibdeps` 和 `dpkg-deb --root-owner-group`，可在不引入新的语言级依赖的前提下创建 Debian 包并从实际二进制解析运行时依赖。

## 目标 / 非目标

**目标：**

- 使每次 Windows 构建的 EXE 和可选 NSIS 安装程序均使用当前 `build/appicon.png` 派生的多分辨率 ICO。
- 保留现有 Linux 二进制构建命令和产物位置，并通过显式选项生成架构正确、可安装且可从桌面启动的 DEB 包。
- 让 DEB 的动态库依赖由实际构建产物解析，避免写死 WebKitGTK 的发行版相关包名。
- 让任务右键操作菜单的搁置与取消搁置动作具有准确、成对且与既有菜单一致的图标。
- 在 README 中提供可复制的构建、检查和安装说明。

**非目标：**

- 不添加 RPM、AppImage、Flatpak、Snap、自动签名或上传发布平台的流程。
- 不升级 Wails、Go、GTK 或 WebKitGTK，不改变应用运行时功能。
- 不承诺一个 DEB 包跨所有 Debian/Ubuntu 发行版可用；包仍面向构建主机兼容的 ABI 与软件源。
- 不在本变更中改变 Windows 资源的版本号、应用名称或 NSIS 安装行为。
- 不改变搁置任务的持久化、排序、生命周期锁定、终端运行或菜单文本，不为自定义任务菜单项新增图标配置。

## 决策

### 1. 以 `build/appicon.png` 为唯一的应用图标来源

Windows PowerShell 脚本会在调用 `wails build` 前删除已存在的 `build/windows/icon.ico`。Wails 随后使用其内置生成器从 `appicon.png` 创建包含 `16` 到 `256` 像素图像的 ICO，并将该文件编译进 EXE；带 `-NSIS` 的构建会让安装器 UI 复用同一文件。

不使用独立图像转换工具或在脚本中维护第二个图标来源。前者增加 ImageMagick 等额外依赖，后者会再次造成两个资源不同步。受版本控制的 ICO 仍作为 Wails 和 NSIS 的工作输入，但其内容始终可由 PNG 重建。

### 2. 在现有 Linux 脚本中增加可选 DEB 模式

`scripts/build-linux.sh` 保持现有架构位置参数，并接受可选 `--deb` 和 `--version <Debian 版本号>`：

```bash
./scripts/build-linux.sh [amd64|arm64|arm] [--deb] [--version 版本号]
```

不传 `--deb` 时，脚本行为及 `build/bin/taskai` 二进制产物完全不变。传入 `--deb` 后，脚本先完成同一次 Wails 构建，再将该二进制组装为 DEB。版本号优先级为命令行 `--version`、环境变量 `TASKAI_DEB_VERSION`、基于当前 Git 短提交号的合法开发版本；脚本必须拒绝不符合 Debian 版本格式的输入。

复用现有脚本避免产生两套不一致的 GTK/WebKitGTK 检查与 Wails 构建参数。单独的 `build-deb.sh` 虽可隔离逻辑，但会让二进制和 DEB 构建路径逐渐分叉。

### 3. 用 Debian 原生工具从暂存目录构建包

DEB 模式额外验证 `dpkg-deb` 与 `dpkg-shlibdeps` 可用，并使用 `mktemp -d` 创建暂存根目录，退出时通过 `trap` 清理。脚本将在暂存目录中创建：

```text
DEBIAN/control
usr/lib/taskai/taskai
usr/bin/taskai -> ../lib/taskai/taskai
usr/share/applications/taskai.desktop
usr/share/icons/hicolor/512x512/apps/taskai.png
```

`taskai.desktop` 使用 `Exec=taskai`、`Icon=taskai`，从而安装后同时支持命令行和应用列表启动。图标直接复制当前 `build/appicon.png`，保证 Linux 包与 Windows 产物使用同一设计源。

脚本以 `dpkg-shlibdeps -O` 解析已生成二进制的 `shlibs:Depends` 字段并写入 `control`，再以 `dpkg-deb --root-owner-group --build` 输出 `build/bin/taskai_<版本>_<Debian 架构>.deb`。架构映射为 `amd64 -> amd64`、`arm64 -> arm64`、`arm -> armhf`。使用原生工具而不是手写 `Depends`，可随所链接的 GTK/WebKitGTK ABI 变化保持正确。

### 4. 将 DEB 验证作为发行构建的独立检查

实现验证将保留现有 Go、前端及 Linux 二进制构建检查，并增加一次 `--deb` 构建。验证使用 `dpkg-deb --info` 确认包名、版本、架构和依赖，使用 `dpkg-deb --contents` 确认二进制、启动入口、桌面文件和图标。Windows 图标刷新在 Windows 主机上通过普通与 `-NSIS` 构建进行确认。

### 5. 以收纳和恢复图标表示搁置状态切换

`TaskTree` 已为固定任务菜单项使用 MUI 的小号描边图标和文本标签。搁置菜单项将遵循同一布局：未搁置的执行中任务显示 `ArchiveOutlined` 与“搁置任务”，已搁置的执行中任务显示 `UnarchiveOutlined` 与“取消搁置”。文本标签保留，图标不单独承担可访问名称。

不使用 `PauseCircleOutline` / `PlayCircleOutline`，因为搁置不会暂停任务或终端；也不使用 `VisibilityOffOutlined`，因为搁置任务仍可在展开区域内显示和操作。`ArchiveOutlined` / `UnarchiveOutlined` 更准确表达任务在正常列表与搁置区之间的收纳和恢复，且无需引入新的图标库或图片资源。

### 6. 从现有 SVG 重新导出带 alpha 的发布图标

`frontend/src/assets/task-ai-mark.svg` 已以圆角矩形定义图标轮廓，轮廓外侧未绘制区域天然透明。当前 `build/appicon.png` 在导出时被扁平化为 RGB，导致其圆角外侧变为不透明白色。将同一 SVG 按 1024px 画布重新导出为 RGBA PNG，可保留既有配色、图案和圆角比例，同时让左上角等轮廓外像素的 alpha 为零。

继续使用 `build/appicon.png` 作为唯一发布资源：Linux DEB 直接复制该 PNG，Windows Wails 由该 PNG 生成 ICO。这样不改变现有构建接口，也不会产生仅 Linux 使用的第二套图标。验证会检查源 PNG 和 DEB 解包出的图标均含 alpha 通道，且左上角像素透明。

## 风险 / 权衡

- [DEB 依赖随构建发行版不同而不同] → 使用 `dpkg-shlibdeps` 从实际二进制推导依赖，并在文档中明确应在目标发行版或兼容环境中构建。
- [旧的 `icon.ico` 被手动修改] → 构建前必然重建 ICO；需要保留的设计修改必须先反映到 `appicon.png`。
- [DEB 模式缺少 Debian 工具] → 仅在请求 `--deb` 时失败并提示缺失工具；普通 Wails 二进制构建不增加这些前置条件。
- [开发版本重复或不适合正式升级] → 正式发布使用显式 `--version` 或 `TASKAI_DEB_VERSION`；自动版本仅用于开发构建。
- [Windows 资源缓存仍显示旧图标] → 脚本保证发布文件内资源是最新的；Windows Explorer 缓存不属于构建脚本可控制的范围，验证应以新生成的 EXE 或安装器文件为准。
- [收纳图标被理解为删除或完成] → 保留“搁置任务 / 取消搁置”完整文本，并使用成对的恢复图标降低歧义；不改变任务状态颜色或生命周期。
- [图标导出时丢失透明通道] → 以 SVG 的未绘制区域为透明基准导出 RGBA PNG，并在 DEB 解包检查中验证角点 alpha，避免仅凭视觉回归判断。

## 迁移计划

1. 更新两个构建脚本及 README，不迁移应用数据或配置。
2. 更新任务树菜单，在两种搁置状态下渲染收纳或恢复图标，并由前端测试覆盖。
3. 在 Linux 构建主机执行普通构建和 `--deb` 构建，并检查 DEB 元数据与文件列表。
4. 在 Windows 主机执行普通构建和 `-NSIS` 构建，确认 EXE 与安装程序使用最新图标。
5. 如需回滚，移除 Linux 脚本的可选 DEB 分支、Windows 图标刷新和菜单图标；不会影响已生成的二进制、DEB 或用户数据。

## 开放问题

无。
