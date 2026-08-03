# manifest-file-generation Specification

## Purpose
定义固定清单文件生命周期命令的配置范围、参数校验、YAML 内容生成、分支回填与安全写入语义，确保不同任务和仓库配置下的清单生成可复现且不越过任务工作目录边界。

## Requirements

### Requirement: 提供固定的清单文件生命周期命令

系统 MUST 提供名为“生成清单文件”的固定生命周期命令。该命令 MUST 仅可被加入 `beforeStart`、`postStart` 和 `updateTask` 命令链，且不得被用户修改或删除。系统 MUST 在生命周期编排界面展示该命令及其中文参数说明。

#### Scenario: 在允许钩子中配置命令
- **WHEN** 用户为 `postStart` 命令链选择“生成清单文件”
- **THEN** 系统允许保存该命令链并展示 `dir`、`name` 两个可选参数的说明

#### Scenario: 在不允许钩子中配置命令
- **WHEN** 用户尝试将“生成清单文件”加入 `beforeEnd` 命令链
- **THEN** 系统拒绝该配置，且不会保存无效命令链

### Requirement: 校验并解析清单输出参数

清单文件命令 MUST 只接受链级参数 `dir=<相对目录>` 和 `name=<文件名>`，每个参数最多一次，且允许任意顺序。未提供 `dir` 时系统 MUST 使用任务工作目录本身；未提供 `name` 时系统 MUST 使用 `manifest.yaml`。`dir` MUST 是不离开任务工作目录的相对路径；`name` MUST 是单个非空文件名，不得为绝对路径、`.`、`..` 或包含目录分隔符。系统 MUST 拒绝重复、未知、空白或不安全的参数。

#### Scenario: 使用默认输出位置和文件名
- **WHEN** 用户将未附加参数的“生成清单文件”加入命令链
- **THEN** 系统在任务工作目录写入名为 `manifest.yaml` 的清单文件

#### Scenario: 使用自定义目录和文件名
- **WHEN** 用户使用参数 `name=iteration.yaml` 和 `dir=configs/task` 配置该命令
- **THEN** 系统在 `<任务工作目录>/configs/task/iteration.yaml` 写入清单文件

#### Scenario: 拒绝不安全或重复参数
- **WHEN** 用户配置 `dir=../outside`、`name=dir/manifest.yaml`、重复 `name` 参数或任意未知参数
- **THEN** 系统拒绝保存命令链并说明参数无效

### Requirement: 从任务和 Git 信息生成 YAML 清单

系统 MUST 以 YAML 映射生成清单，其根节点 MUST 包含 `iteration`、`desc` 和 `repos`。`iteration` MUST 等于任务标题，`desc` MUST 等于任务描述。`repos` MUST 按任务已选择的内置 Git 信息保存顺序生成；每个仓库条目 MUST 包含 `name`、`url`、`branch`，并分别取 Git 信息的项目名称、仓库地址和解析后的分支。系统 MUST 通过 YAML 序列化正确表示特殊字符、空字符串和多行文本。

#### Scenario: 导出多个 Git 仓库
- **WHEN** 任务标题为“Android 2.45”、描述为“发布准备”，并选择了包含项目名称、仓库地址和分支的两个 Git 信息
- **THEN** 清单的 `iteration` 为“Android 2.45”、`desc` 为“发布准备”，且 `repos` 依选择顺序包含两个带有对应 `name`、`url`、`branch` 的条目

#### Scenario: 导出不含 Git 信息的任务
- **WHEN** 任务未选择任何内置 Git 信息
- **THEN** 系统仍生成清单，并将 `repos` 写为 YAML 空序列

### Requirement: 回填 Git 仓库分支

系统 MUST 优先使用命令链当前执行态 Git 信息中非空的 `branch` 值。当其为空白时，系统 MUST 将该仓库的 `branch` 写为空字符串，而不得省略该键。生成清单文件 MUST 不直接读取当前任务模板字段；需要按模板字段补全分支时，命令链 MUST 先执行“更新默认分支”命令。

#### Scenario: 使用执行态默认分支补全空 Git 分支
- **WHEN** 命令链先通过“更新默认分支”将模板字段值 `android2.45-0727` 填入一个空 Git 信息分支，随后生成清单
- **THEN** 生成清单中该仓库的 `branch` 为 `android2.45-0727`

#### Scenario: 保留 Git 信息的显式分支
- **WHEN** 命令链的默认分支为 `android2.45-0727`，且一个已选 Git 信息的 `branch` 为 `dev-cj-1.2`
- **THEN** 生成清单中该仓库的 `branch` 为 `dev-cj-1.2`

#### Scenario: 未更新默认分支时输出空分支
- **WHEN** Git 信息的 `branch` 为空，任务模板包含非空 `branch`，但命令链未先执行“更新默认分支”
- **THEN** 生成清单中该仓库包含值为空字符串的 `branch` 键

### Requirement: 安全地创建并更新清单文件

系统 MUST 要求任务工作目录存在且不是符号链接。系统 MUST 在该目录内安全创建缺失的合法 `dir` 子目录，并拒绝经过符号链接、离开任务工作目录或指向目录、符号链接及其他非普通文件的目标。目标为普通文件时，系统 MUST 以原子替换方式覆盖其内容，且写入失败时不得留下部分清单内容。

#### Scenario: 原子覆盖已有清单
- **WHEN** 目标路径已有普通 `manifest.yaml` 文件，且任务更新触发清单生成
- **THEN** 系统以最新任务数据完整替换该文件内容

#### Scenario: 拒绝符号链接目标
- **WHEN** 合法输出路径中的目录或目标文件是符号链接
- **THEN** 系统使当前生命周期命令失败，且不会在任务工作目录之外写入文件

### Requirement: 保持生命周期失败与命令链数据传递语义

清单文件命令 MUST 不改变命令链传递给相邻自定义命令的字节流。该命令在 `beforeStart` 中因目录或写入问题失败时，系统 MUST 阻止任务进入执行中状态；该命令在 `postStart` 或 `updateTask` 中失败时，系统 MUST 保留已提交的状态或任务编辑结果，并记录可重试的生命周期失败。

#### Scenario: 启动前未创建任务工作目录
- **WHEN** `beforeStart` 链未先创建任务工作目录而执行清单文件命令
- **THEN** 系统使启动失败并提示任务工作目录不可用

#### Scenario: 编辑后生成失败
- **WHEN** 执行中的任务在 `updateTask` 清单生成时发生写入错误
- **THEN** 系统保留用户刚保存的任务编辑内容，并显示该生命周期命令失败及重试入口
