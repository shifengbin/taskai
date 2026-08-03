## ADDED Requirements

### Requirement: Windows 发布构建使用当前应用图标

系统 MUST 将 `build/appicon.png` 作为 Windows 应用图标的唯一来源。Windows 构建脚本 MUST 在每次调用 Wails 构建前移除已有的 `build/windows/icon.ico`，使 Wails 重新生成多分辨率 ICO 并将其编译到可执行文件。启用 NSIS 时，安装程序 MUST 使用同一重新生成的 ICO。

#### Scenario: 已有 ICO 时更新 Windows 可执行文件图标
- **WHEN** `build/windows/icon.ico` 已存在，且开发者运行 Windows 构建脚本
- **THEN** 脚本在 Wails 构建前移除该 ICO，并由当前 `build/appicon.png` 重新生成用于 EXE 的 ICO

#### Scenario: 生成 NSIS 安装程序
- **WHEN** 开发者以 `-NSIS` 运行 Windows 构建脚本
- **THEN** 生成的 NSIS 安装程序和同次构建的 EXE 使用由当前 `build/appicon.png` 派生的同一 ICO

### Requirement: 发布图标保留圆角外侧透明区域

系统 MUST 将 `build/appicon.png` 保存为带 alpha 通道的 1024px PNG。图标圆角轮廓外的像素 MUST 透明，且保留现有 TaskAI 图案、配色和圆角比例。Windows ICO 与 Linux DEB MUST 继续从该同一 PNG 资源派生或安装，不得为 Linux 维护独立图标。

#### Scenario: 检查发布图标的角点透明度
- **WHEN** 开发者检查 `build/appicon.png` 的左上角像素
- **THEN** 图像包含 alpha 通道，且该像素的 alpha 值为零

#### Scenario: 检查 DEB 中安装的图标
- **WHEN** 开发者解包使用当前 `build/appicon.png` 构建的 DEB
- **THEN** `/usr/share/icons/hicolor/512x512/apps/taskai.png` 保留 alpha 通道和透明的左上角像素

### Requirement: Linux 构建支持可选 Debian 包产出

系统 MUST 保留现有 Linux 构建脚本的默认二进制构建行为。该脚本 MUST 接受 `--deb` 选项；指定该选项时，脚本 MUST 在成功生成 `build/bin/taskai` 后额外生成 `build/bin/taskai_<版本>_<架构>.deb`。脚本 MUST 支持 `amd64`、`arm64` 和 `arm` 构建架构，并分别在 DEB 控制信息中声明 `amd64`、`arm64` 和 `armhf`。

#### Scenario: 保持普通 Linux 二进制构建
- **WHEN** 开发者未传入 `--deb` 运行 Linux 构建脚本
- **THEN** 脚本仅按既有规则生成 `build/bin/taskai`，且不要求 Debian 打包工具存在

#### Scenario: 为 AMD64 生成 DEB
- **WHEN** 开发者以 `amd64 --deb` 运行 Linux 构建脚本并且 Wails 构建成功
- **THEN** 脚本生成架构字段为 `amd64` 的 `taskai_<版本>_amd64.deb`，同时保留 `build/bin/taskai`

#### Scenario: 映射 32 位 ARM Debian 架构
- **WHEN** 开发者以 `arm --deb` 运行 Linux 构建脚本并且 Wails 构建成功
- **THEN** 脚本在 DEB 控制信息和产物文件名中使用 `armhf` 架构

### Requirement: Debian 包提供可发现的应用安装布局

系统生成的 DEB MUST 安装 TaskAI 可执行文件、`taskai` 命令行启动入口、桌面启动器和当前 `build/appicon.png`。桌面启动器 MUST 通过 `Exec=taskai` 启动应用，并通过 `Icon=taskai` 关联已安装的应用图标。

#### Scenario: 检查 DEB 内容
- **WHEN** 开发者检查已生成 DEB 的文件列表
- **THEN** 包含应用二进制、`/usr/bin/taskai`、`/usr/share/applications/taskai.desktop` 和 `/usr/share/icons/hicolor/512x512/apps/taskai.png`

#### Scenario: 安装后从桌面环境启动
- **WHEN** 用户安装生成的 DEB 并在支持 freedesktop 桌面文件的环境中选择 TaskAI
- **THEN** 桌面环境使用包内的图标显示 TaskAI，并通过 `taskai` 命令启动应用

### Requirement: Debian 包具备有效版本和运行时依赖

DEB 模式 MUST 校验 `dpkg-deb` 与 `dpkg-shlibdeps` 可用。系统 MUST 使用 `dpkg-shlibdeps` 从同次构建的应用二进制解析运行时共享库依赖，并写入 DEB 控制信息。系统 MUST 使用命令行版本号、`TASKAI_DEB_VERSION` 或合法的自动开发版本，并在版本号不符合 Debian 格式时失败而不产生 DEB。

#### Scenario: 生成带显式版本的包
- **WHEN** 开发者在 DEB 模式中提供合法的显式版本号
- **THEN** DEB 的文件名和控制信息均使用该版本号，且控制信息包含由应用二进制解析出的运行时依赖

#### Scenario: 缺少 Debian 打包工具
- **WHEN** 开发者请求 `--deb`，但 `dpkg-deb` 或 `dpkg-shlibdeps` 不可用
- **THEN** 脚本以非零状态退出并说明缺少的工具，且不产生不完整的 DEB

#### Scenario: 拒绝非法 Debian 版本
- **WHEN** 开发者在 DEB 模式中提供不符合 Debian 版本格式的版本号
- **THEN** 脚本以非零状态退出并说明版本号无效，且不产生 DEB
