# macos-workspace-build-compatibility Specification

## Purpose
定义 macOS 工作区包不依赖扩展 ACL cgo 的构建边界、Unix 私有目录保留的所有者与普通权限校验，以及平台构建变更的分层集成测试要求。

## Requirements

### Requirement: Darwin 工作区包不依赖扩展 ACL cgo
系统 MUST 在 Darwin 上构建 `internal/workspace` 时不选择任何 cgo 源文件，也不得引用 macOS 扩展 ACL C API。启用或禁用 cgo 时，工作区包都 MUST 保留完整的 Go 实现并能够参与应用构建。

#### Scenario: 检查 Darwin 包文件选择
- **WHEN** 测试以 `GOOS=darwin` 和 `CGO_ENABLED=1` 查询 `internal/workspace` 的 Go 包信息
- **THEN** 包的 `CgoFiles` 为空，且不会选择引用 `acl_delete_file_np` 的源文件

#### Scenario: 构建 macOS universal 应用
- **WHEN** macOS GitHub Actions 使用项目脚本执行 Wails `darwin/universal` 构建
- **THEN** Wails 成功生成绑定并完成 amd64 与 arm64 应用构建，不出现 ACL C API 名称解析错误

### Requirement: Unix 私有目录继续校验所有者和普通权限
系统 SHALL 在 Darwin 和 Linux 上继续把新建私有元数据目录设置为 `0700`，并 MUST 拒绝使用不属于当前用户或普通 Unix 权限允许组或其他用户访问的私有目录。系统 SHALL NOT 读取、清理或根据 macOS 扩展 ACL 拒绝目录。

#### Scenario: 收紧当前用户目录权限
- **WHEN** 当前用户拥有的私有元数据目录具有比 `0700` 更宽的普通 Unix 权限
- **THEN** 系统把权限收紧为 `0700`，验证所有者后继续工作区操作

#### Scenario: 拒绝其他用户拥有的目录
- **WHEN** 私有元数据目录不属于当前有效用户
- **THEN** 系统拒绝使用该目录，且不创建或删除任务工作目录

#### Scenario: 忽略 macOS 扩展 ACL
- **WHEN** Darwin 上的私有元数据目录存在扩展 ACL
- **THEN** 系统不读取或修改该 ACL，仅依据目录所有者和普通 Unix 权限继续校验

### Requirement: 平台构建变更经过分层集成测试
系统 MUST 在交付前验证 Darwin amd64 与 arm64 的工作区包能够编译，并 MUST 运行 Go 完整测试、前端测试与构建、Linux 可执行程序编译以及 Wails 开发模式启动检查。浏览器测试 MUST 使用 `wails dev` 输出的调试地址确认应用加载和控制台状态，完成后 MUST 关闭调试进程。

#### Scenario: 编译 Darwin 双架构工作区包
- **WHEN** 开发者分别以 `GOARCH=amd64` 和 `GOARCH=arm64`、`GOOS=darwin` 编译 `internal/workspace` 测试程序
- **THEN** 两次编译均成功，且不要求 ACL cgo 源文件

#### Scenario: 启动开发应用进行浏览器测试
- **WHEN** 完整自动化测试通过后以 `wails dev` 启动应用
- **THEN** 测试从彩色终端输出取得调试地址，通过浏览器确认应用加载、任务列表显示且控制台没有启动错误，然后关闭调试进程
