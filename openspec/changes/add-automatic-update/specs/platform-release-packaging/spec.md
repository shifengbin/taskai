## ADDED Requirements

### Requirement: 发布构建向应用注入统一版本

系统 MUST 让 Linux、Windows 和 macOS 发布构建将同一发布版本注入 TaskAI 可执行程序。由 tag 触发的构建所嵌入版本 MUST 与该 tag 表示的语义版本等价，并可由运行中的应用读取用于更新比较；非 tag 开发构建 MUST 使用明确的开发版本，不能从安装包文件名反推运行版本。

#### Scenario: 三端产物嵌入 tag 版本

- **WHEN** 发布工作流由 `v1.2.3-rc.1` tag 触发并完成三端构建
- **THEN** Linux、Windows 和 macOS 应用均报告与 `v1.2.3-rc.1` 等价的当前版本

#### Scenario: 开发构建使用明确版本

- **WHEN** 开发者在没有发布 tag 的情况下构建应用
- **THEN** 应用报告构建流程提供的明确开发版本，且不读取 DEB、NSIS 或 DMG 文件名推断版本
