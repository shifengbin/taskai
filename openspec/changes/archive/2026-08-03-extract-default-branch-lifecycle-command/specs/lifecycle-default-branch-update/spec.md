## ADDED Requirements

### Requirement: 提供固定的默认分支更新命令
系统 MUST 提供名为“更新默认分支”的固定生命周期命令。该命令 MUST 仅可加入 `beforeStart`、`postStart` 和 `updateTask` 命令链，且不得被用户修改或删除。系统 MUST 在生命周期编排界面显示中文说明及可配置的链级参数。

#### Scenario: 在允许钩子中配置默认分支更新
- **WHEN** 用户将“更新默认分支”加入仅适用于 `updateTask` 的命令链
- **THEN** 系统接受该命令链并显示其模板字段参数说明

#### Scenario: 拒绝不允许的钩子
- **WHEN** 用户尝试将“更新默认分支”加入 `beforeEnd` 命令链
- **THEN** 系统拒绝保存该命令链

### Requirement: 解析默认分支模板字段参数
“更新默认分支”命令 MUST 只接受至多一个链级参数 `templateField=<模板字段键>`。未提供参数时系统 MUST 使用 `branch` 作为模板字段键。系统 MUST 拒绝空值、重复键、未知键或不含等号的参数。命令运行时 MUST 使用调度时冻结的活动任务模板可见字段；指定字段不存在或字符串值为空白时 MUST 解析为空默认分支，字段存在但不是字符串时 MUST 使当前命令链失败。

#### Scenario: 使用默认模板字段
- **WHEN** 命令未配置追加参数，冻结的模板字段 `branch` 值为 `release/2.0`
- **THEN** 系统将 `release/2.0` 作为本次执行的默认分支

#### Scenario: 使用指定模板字段
- **WHEN** 命令配置 `templateField=releaseBranch`，冻结的模板字段 `releaseBranch` 值为 `release/2.1`
- **THEN** 系统将 `release/2.1` 作为本次执行的默认分支

#### Scenario: 拒绝无效参数
- **WHEN** 用户配置 `templateField=a` 和 `templateField=b`、`templateField= ` 或 `field=branch`
- **THEN** 系统拒绝保存该命令链并说明默认分支更新参数无效

### Requirement: 仅在命令链执行态补全默认分支
“更新默认分支”命令 MUST 仅在当前命令链的内存任务副本中处理内置 Git 信息。解析出的默认分支非空时，系统 MUST 仅为 `branch` 为空白的内置 Git 信息填写该值，MUST 保留显式 Git 分支，且 MUST 不处理其他额外信息。该命令 MUST 将解析结果作为当前执行的单仓库克隆默认分支；命令链完成、失败或重试后，系统 MUST 不将该结果写回任务、额外信息或模板持久化数据。

#### Scenario: 补全空 Git 分支但保留显式值
- **WHEN** 默认分支为 `release/2.0`，一个内置 Git 信息的 `branch` 为空，另一个为 `hotfix/2.0.1`
- **THEN** 同一执行链后续命令看到的两个分支分别为 `release/2.0` 和 `hotfix/2.0.1`

#### Scenario: 执行后不改写任务快照
- **WHEN** 任务保存的内置 Git 信息 `branch` 为空，命令链通过“更新默认分支”补全并执行成功
- **THEN** 重新读取任务时该 Git 信息的保存分支仍为空

#### Scenario: 空模板字段不补全
- **WHEN** 指定模板字段不存在或其字符串值为空白，且内置 Git 信息 `branch` 为空
- **THEN** 后续命令看到的该 Git 信息分支仍为空

### Requirement: 默认分支消费者只读取执行态数据
“克隆指定 Git 仓库” MUST 使用当前命令链执行态默认分支选择 Git 分支，不得直接解析任务模板字段；“生成清单文件”和“Git 仓库克隆” MUST 使用当前执行态内置 Git 信息的 `branch`，不得自行回退任务模板字段。未先执行“更新默认分支”或解析结果为空时，两个 Git 克隆命令 MUST 让远程仓库决定默认分支，清单 MUST 写入空字符串分支。任务创建和编辑 MUST 保留用户提交的 Git 信息分支，不得根据模板字段回填空分支。

#### Scenario: 链内默认分支驱动单仓库与清单
- **WHEN** 链先执行“更新默认分支”，其默认分支为 `release/2.0`，随后执行“克隆指定 Git 仓库”和“生成清单文件”，且 Git 信息分支为空
- **THEN** 指定仓库以 `release/2.0` 检出，清单中对应 Git 信息的分支为 `release/2.0`

#### Scenario: 未加入更新命令时不隐式读取模板
- **WHEN** 任务模板 `branch` 为 `release/2.0`，但链未包含“更新默认分支”，且 Git 信息分支为空
- **THEN** 指定仓库与批量 Git 仓库克隆均使用远程默认分支，清单写入空字符串分支

#### Scenario: 保存任务不回填模板分支
- **WHEN** 用户保存模板字段 `branch=release/2.0` 且所选内置 Git 信息 `branch` 为空的任务
- **THEN** 保存后的任务 Git 信息分支仍为空，直到包含“更新默认分支”的命令链运行时才可能在执行态补全
