# task-templates Specification

## Purpose
TBD - created by archiving change add-task-templates. Update Purpose after archive.
## Requirements
### Requirement: 在设置中管理并选择任务模板

系统 SHALL 在设置中维护任务模板集合，并保存可为空的当前使用模板标识。每个任务模板 MUST 保存唯一标识、非空且唯一的名称，以及一个或多个有序字段定义。系统 MUST 支持新增、编辑和删除模板，并允许用户在已有模板中选择当前使用模板。删除当前使用模板时，系统 MUST 清空当前模板选择，且不得删除任何任务已保存的模板字段值。普通设置保存 MUST 将任务模板集合与当前模板标识视为同一个可选快照：请求未提供模板集合时，系统 MUST 保留当前已持久化的模板集合和当前模板标识；请求明确提供模板集合时，系统 MUST 保存并校验该集合及其当前模板标识。明确提供空集合和空当前模板标识 MUST 继续表示删除全部模板，而不得被保留逻辑覆盖。

#### Scenario: 创建并选择任务模板
- **WHEN** 用户在设置中新建名称为“发布任务”的模板，添加字段后将其设为当前使用模板并保存
- **THEN** 系统保存该模板和当前模板标识，后续新建及编辑任务使用该模板

#### Scenario: 删除当前任务模板
- **WHEN** 用户删除当前使用的任务模板
- **THEN** 系统清空当前模板选择，任务表单不再显示模板字段，且已有任务的历史模板字段值保持不变

#### Scenario: 未选择任务模板
- **WHEN** 设置未选择当前使用模板
- **THEN** 系统允许创建和编辑任务，且不要求或展示任何模板字段

#### Scenario: 保存其他设置时未提交模板快照
- **WHEN** 已保存模板和当前模板选择的客户端提交只修改 Shell、工作区、面板宽度或当前任务状态的设置请求，且请求未提供模板集合
- **THEN** 系统保存请求中的普通设置字段，保留已保存的模板定义和当前模板选择；重新加载并打开设置时仍显示这些模板

#### Scenario: 明确清空模板集合
- **WHEN** 用户在设置中删除最后一个模板并提交空模板集合和空当前模板标识
- **THEN** 系统保存空模板集合和空当前模板选择，后续任务表单不显示模板字段

### Requirement: 校验模板字段定义与默认值

系统 MUST 要求每个模板字段保存非空键、非空显示名称、`string` 或 `bool` 输入类型、类型匹配的默认值、必填标记和环境变量注入标记。字段键 MUST 以 ASCII 字母开头，且只包含 ASCII 字母、数字和下划线；同一模板中的键在大小写不敏感比较下 MUST 唯一。将键转为大写并添加 `TASKAI_` 前缀后，系统 MUST 拒绝与 `TASKAI_TASK_ID`、`TASKAI_TERMINAL_ID`、`TASKAI_STATUS_API`、`TASKAI_EXEC_COMMAND` 或 `TASKAI_EXEC_ARGUMENTS` 冲突的键。系统 MUST 拒绝修改已经被任一任务保存的字段键的输入类型。

#### Scenario: 保存包含字符串和布尔默认值的字段
- **WHEN** 用户新增键为 `environment`、默认值为字符串 `production` 的字段，以及键为 `deploy`、默认值为布尔值 `false` 的字段
- **THEN** 系统保存两个字段及其原生类型默认值

#### Scenario: 拒绝无效或冲突的字段键
- **WHEN** 用户保存键为 `deploy-env`、与已有字段仅大小写不同的键，或键为 `task_id` 且要求环境变量注入的字段
- **THEN** 系统拒绝保存并说明字段键无效、重复或与内置环境变量冲突

#### Scenario: 拒绝改变已使用字段的输入类型
- **WHEN** 至少一个任务已保存键为 `deploy` 的值，用户将该字段从 `bool` 修改为 `string`
- **THEN** 系统拒绝保存模板，并保留原字段定义和任务值

### Requirement: 任务表单按当前模板编辑模板字段

系统 SHALL 在新建和编辑任务对话框中，将当前使用模板的字段按定义顺序渲染在任务描述之后、任务颜色之前。字符串字段 MUST 使用文本输入，布尔字段 MUST 使用复选框。新建任务 MUST 使用字段默认值；编辑任务中不存在当前字段键时 MUST 显示默认值，已保存的值（包括空字符串）MUST 优先于默认值。保存任务时，系统 MUST 校验必填字符串的去除首尾空白后的值非空，并校验必填布尔字段为 `true`。

#### Scenario: 新建任务应用模板默认值
- **WHEN** 当前模板定义 `environment` 的默认值为 `production`、`deploy` 的默认值为 `false`
- **THEN** 新建任务对话框在任务描述和颜色之间分别显示这两个默认值，保存后任务记录字符串和布尔值

#### Scenario: 编辑旧任务补齐新增字段
- **WHEN** 当前模板新增默认值为 `staging` 的 `environment` 字段，用户编辑一个尚未保存该键的旧任务
- **THEN** 表单显示 `staging`，且用户保存后任务记录该值

#### Scenario: 必填布尔字段必须勾选
- **WHEN** 当前模板的 `deploy` 布尔字段标记为必填，用户未勾选该字段而尝试保存任务
- **THEN** 系统拒绝保存并提示该字段必须为 `true`

### Requirement: 保留历史模板字段值并仅使用当前可见字段

系统 MUST 在任务中以键值对象保存模板字段值，且值仅能是字符串或布尔值。当前模板被切换、删除字段或修改字段键时，系统 MUST 保留不再属于当前模板的历史键和值。历史字段 MUST 不在任务表单中显示，且 MUST 不出现在任务 HTTP 响应、生命周期命令链标准输入或环境变量中；字段以后以相同键重新加入当前模板时，系统 MUST 再次使用保留值。

#### Scenario: 删除字段后保留历史值
- **WHEN** 任务保存 `environment=production`，随后当前模板删除 `environment` 字段
- **THEN** 任务继续保存该历史值，但编辑表单、HTTP 响应和命令链输入均不包含 `environment`

#### Scenario: 恢复同键字段
- **WHEN** 当前模板重新添加键为 `environment` 的字符串字段
- **THEN** 编辑包含历史 `environment=production` 的任务时，系统显示 `production` 而非新字段默认值

### Requirement: 通过 HTTP 和命令链输出当前模板字段

系统 SHALL 在 `GET /api/v1/tasks` 与 `GET /api/v1/tasks/{taskId}` 的每个任务对象中返回 `templateFields` 对象。系统 MUST 仅输出当前模板定义的字段，并以字符串或 JSON 布尔值表示字段值；无当前模板或没有可见字段值时 MUST 返回空对象。系统 MUST 使用相同语义在生命周期命令链的标准输入 JSON 中输出 `templateFields`。

#### Scenario: HTTP 列表和详情返回模板字段
- **WHEN** 当前模板包含 `environment=production` 和 `deploy=true`，外部扩展请求任务列表和该任务详情
- **THEN** 两个响应中的任务对象均包含 `"templateFields":{"environment":"production","deploy":true}`

#### Scenario: 命令链标准输入返回模板字段
- **WHEN** 生命周期命令链开始执行一个当前模板字段为 `environment=production` 的任务
- **THEN** 每个自定义命令收到的初始标准输入 JSON 包含 `"templateFields":{"environment":"production"}`

#### Scenario: 无模板时返回空对象
- **WHEN** 设置没有当前使用模板
- **THEN** 任务 HTTP 响应和生命周期命令链标准输入均包含 `"templateFields":{}`

### Requirement: 向自定义生命周期命令注入选定字段环境变量

系统 MUST 只向自定义生命周期 Shell 命令注入当前模板中已标记环境变量注入的字段。环境变量名 MUST 为 `TASKAI_` 加字段键的大写形式；字符串字段使用原字符串值，布尔字段使用 `true` 或 `false`。可选字符串值为空时，系统 MUST 仍注入值为空的变量。系统 MUST 不向内置创建工作目录、删除工作目录或 Git 克隆命令注入这些变量。

#### Scenario: 注入字符串和布尔环境变量
- **WHEN** `environment` 与 `deploy` 均标记环境变量注入，任务值分别为 `production` 与 `true`
- **THEN** 自定义生命周期命令收到 `TASKAI_ENVIRONMENT=production` 和 `TASKAI_DEPLOY=true`

#### Scenario: 注入空字符串值
- **WHEN** 可选的 `release_note` 字符串字段标记环境变量注入且任务值为空字符串
- **THEN** 自定义生命周期命令收到值为空的 `TASKAI_RELEASE_NOTE` 环境变量

#### Scenario: 内置命令保持既有执行方式
- **WHEN** 命令链包含创建工作目录或 Git 克隆等内置命令
- **THEN** 系统继续按既有 Go 或 Git 执行路径运行，且不要求这些命令接收模板环境变量

### Requirement: 兼容没有任务模板的既有数据

系统 MUST 能够加载未包含任务模板、当前模板标识或任务 `templateFields` 的既有数据文件，并将缺失值归一化为无模板、无当前模板选择和空任务字段对象。保存后，系统 MUST 保留既有任务的标题、描述、颜色、附加信息和生命周期命令链。

#### Scenario: 加载旧数据文件
- **WHEN** 应用加载不含任务模板相关字段的既有数据文件
- **THEN** 应用正常启动，现有任务保持可用，且 HTTP 响应中的每个任务提供空 `templateFields` 对象
