## MODIFIED Requirements

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
