## ADDED Requirements

### Requirement: macOS 构建支持可选 DMG 包产出

系统 MUST 保留现有 macOS 构建脚本的默认应用包构建行为。该脚本 MUST 接受 `--dmg` 选项；指定该选项时，脚本 MUST 在成功生成 `build/bin/taskai.app` 后额外生成 `build/bin/TaskAI-<版本>-<架构>.dmg`。该脚本 MUST 支持 `amd64`、`arm64` 和 `universal` 构建架构。

#### Scenario: 保持普通 macOS 应用包构建

- **WHEN** 开发者未传入 `--dmg` 运行 macOS 构建脚本
- **THEN** 脚本仅按既有规则生成 `build/bin/taskai.app`，且不执行 DMG 打包步骤

#### Scenario: 为通用架构生成 DMG

- **WHEN** 开发者以 `universal --dmg` 运行 macOS 构建脚本并且 Wails 构建成功
- **THEN** 脚本生成 `TaskAI-<版本>-universal.dmg`，同时保留 `build/bin/taskai.app`

### Requirement: DMG 包具备拖拽安装布局与有效版本

DMG 模式 MUST 使用系统 `hdiutil` 产出只读 UDZO 镜像，且 MUST 在镜像内放置 TaskAI 应用包与一个指向 `/Applications` 的软链接。系统 MUST 使用命令行版本号、`TASKAI_VERSION` 或合法的自动开发版本，并在版本号不符合版本格式时失败而不产生 DMG。

#### Scenario: 生成带显式版本的 DMG

- **WHEN** 开发者在 DMG 模式中提供合法的显式版本号
- **THEN** DMG 的文件名包含该版本号，且镜像内包含 TaskAI 应用包与指向 `/Applications` 的软链接

#### Scenario: 拒绝非法版本号

- **WHEN** 开发者在 DMG 模式中提供不符合版本格式的版本号
- **THEN** 脚本以非零状态退出并说明版本号无效，且不产生 DMG
