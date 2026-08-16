## ADDED Requirements

### Requirement: 三端发布构建在原生运行器上并行执行

工作流 MUST 在推送 `v*` tag 或 `workflow_dispatch` 触发时，于 `ubuntu`、`windows`、`macos` 三个原生运行器上各运行一个构建任务，分别产出 Linux、Windows、macOS 发布产物。工作流 MUST NOT 在单一运行器上交叉编译其他平台的发布产物。

#### Scenario: 推送 tag 触发三端并行构建

- **WHEN** 推送形如 `v1.2.3` 的 tag
- **THEN** 工作流在三个原生运行器上并行运行 Linux、Windows、macOS 构建任务

#### Scenario: 手动触发三端构建

- **WHEN** 通过 `workflow_dispatch` 手动触发工作流
- **THEN** 工作流同样在三个原生运行器上并行运行三端构建任务

### Requirement: 版本号从 git tag 派生并统一传入三端

工作流 MUST 从触发 tag 剥掉前导 `v` 得到版本号，并传入三端构建脚本。无 tag 触发时，工作流 MUST 回落到 `0.0.0+git.<短 sha>` 形式的开发版本号。

#### Scenario: 从 tag 派生版本号

- **WHEN** 工作流由 `v1.2.3` tag 触发
- **THEN** 三端构建脚本接收到的版本号为 `1.2.3`

#### Scenario: 无 tag 时回落到开发版本号

- **WHEN** 工作流由 `workflow_dispatch` 手动触发且无关联 tag
- **THEN** 三端构建脚本接收到形如 `0.0.0+git.<短 sha>` 的版本号

### Requirement: 每次构建上传平台产物

每个构建任务 MUST 在成功后用 `upload-artifact` 上传本平台产物（Linux 的 `.deb`、Windows 的 NSIS 安装程序、macOS 的 `.dmg`），且产物命名 MUST 包含当前版本号。

#### Scenario: 三端分别上传产物

- **WHEN** 任一平台构建任务成功
- **THEN** 该任务上传对应的 `.deb`、`.exe` 或 `.dmg` 产物

### Requirement: 仅 tag 触发时发布 GitHub Release

工作流 MUST 仅在 `push.tags v*` 触发时，增加一个依赖三构建任务的发布任务，下载三端产物并挂载到同一 GitHub Release。`workflow_dispatch` 手动触发 MUST NOT 创建或更新 GitHub Release。

#### Scenario: tag 触发时挂载三端产物到 Release

- **WHEN** 工作流由 `v*` tag 触发且三构建任务成功
- **THEN** 发布任务下载三端产物并挂载到该 tag 对应的 GitHub Release

#### Scenario: 手动触发不创建 Release

- **WHEN** 工作流由 `workflow_dispatch` 触发
- **THEN** 工作流只上传 artifact，不创建 GitHub Release

### Requirement: 本期发布产物保持未签名

工作流 MUST NOT 对发布产物进行 Windows 代码签名或 macOS 公证。tag 发布的 Release 说明 MUST 给出 macOS 与 Windows 首次运行的放行步骤。

#### Scenario: 检查产物签名状态

- **WHEN** 开发者检查任一发布产物
- **THEN** 产物不携带有效代码签名
