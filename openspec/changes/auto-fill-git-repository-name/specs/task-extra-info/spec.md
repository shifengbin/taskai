## ADDED Requirements

### Requirement: Git 信息仓库地址自动回填项目名称
系统 MUST 在用户新增或编辑分类为 `git` 的可复用信息时，于用户修改键为 `repository` 的固定字段后，保留该仓库地址输入值，并在键为 `name` 的固定字段去除首尾空白后为空时自动回填项目名称。系统 MUST 将去除首尾空白后的仓库地址中最后一个 `/` 后、结尾 `.git` 前的非空内容作为项目名称；无法按此规则提取时，系统 MUST 保持项目名称不变。系统 MUST 不覆盖任何非空项目名称，且不得对非 `git` 分类触发该行为。

#### Scenario: 从 SSH 仓库地址回填空项目名称
- **WHEN** 用户在项目名称为空的 Git 信息中填写仓库地址 `git@gitlab.jiandan100.cn:webdev/interact-study.git`
- **THEN** 系统将仓库地址保存为输入值，并将项目名称自动填写为 `interact-study`

#### Scenario: 保留已填写的项目名称
- **WHEN** 用户在项目名称为“互动学习”的 Git 信息中填写仓库地址 `git@gitlab.jiandan100.cn:webdev/interact-study.git`
- **THEN** 系统更新仓库地址，但项目名称仍为“互动学习”

#### Scenario: 无法提取时不修改项目名称
- **WHEN** 用户在项目名称为空的 Git 信息中填写不含结尾 `.git` 或不含有效最后路径段的仓库地址
- **THEN** 系统更新仓库地址，但项目名称保持为空

#### Scenario: 非 Git 分类不自动回填
- **WHEN** 用户在非 `git` 分类的信息中修改键为 `repository` 的固定字段，且名称字段为空
- **THEN** 系统更新仓库地址，但不自动修改名称字段
