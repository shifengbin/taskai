## MODIFIED Requirements

### Requirement: 搁置任务操作显示收纳和恢复图标

系统 MUST 在执行中任务的右键及任务操作菜单中，为搁置切换系统项显示与状态对应的成对图标：未搁置任务显示 MUI `ArchiveOutlined` 图标，已搁置任务显示 MUI `UnarchiveOutlined` 图标。图标 MUST 与同一菜单中现有固定操作的尺寸和图标加文本布局一致，且 MUST 与当前状态对应的已配置菜单名称一同完整显示。

#### Scenario: 显示未搁置任务的收纳操作
- **WHEN** 用户打开未搁置执行中任务的右键菜单或任务操作菜单
- **THEN** 菜单显示 `ArchiveOutlined` 图标和已配置的未搁置名称

#### Scenario: 显示已搁置任务的恢复操作
- **WHEN** 用户打开已搁置执行中任务的右键菜单或任务操作菜单
- **THEN** 菜单显示 `UnarchiveOutlined` 图标和已配置的已搁置名称
