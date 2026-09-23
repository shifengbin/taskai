## Context

任务模板的 `directories` 字段以 `[]string` 保存。写入路径为：前端提交 → `MergeTaskTemplateFields` → `validateTaskTemplateDirectorySelections` 把目录值重建为 `make([]string, 0, len(...))`（非 nil）→ `repository.Save` → `normalizeDataForSave` → `NormalizeTaskTemplateValues`。

`NormalizeTaskTemplateValues` 的 `case []string` 分支使用 `append([]string(nil), value...)` 复制切片。Go 对空切片执行 append 会返回 nil 切片，于是 `Save` 阶段把内存中本为非 nil 的空数组变成 nil，`json.Marshal` 将其序列化为 `null`。加载时 JSON `null` 解出的是 nil interface（无动态类型），`NormalizeTaskTemplateValues` 的类型分支全部不匹配，落到兜底分支返回错误，`Load` 中断，应用启动失败。

该缺陷只在"保存时该字段已是 Go `[]string` 且为空"时触发，因此 `MergeTaskTemplateFields` 之后的校验掩盖了它，而 `Save` 内部的二次归一化恰好构成触发条件。

## Goals / Non-Goals

**Goals:**

- 未选择目录的目录字段持久化为空数组，保存与重新加载往返后语义不变。
- 已经被写入 `null` 的既有数据文件能够正常加载，并在加载过程中被修复。
- 保持任务模板字段校验的严格性：类型不匹配（数字、对象等）的提交仍然被拒绝。

**Non-Goals:**

- 不修改前端提交格式与目录选择交互。
- 不放宽 `NormalizeTaskTemplateValues` 对非 `null` 非法类型的校验。
- 不修改 HTTP 接口输出、生命周期命令链输入或模板定义格式。
- 不改变必填目录字段、单目录/多目录约束的校验行为。

## Decisions

### 决策 1：复制目录值时保证非 nil（根因修复）

在 `internal/task` 中新增 `cloneDirectories`，以 `append([]string{}, value...)` 作为唯一的目录切片复制入口，并在 `NormalizeTaskTemplateValues` 与 `normalizeTaskTemplateFieldValue` 两处使用。

选择统一辅助函数而不是就地改写 `[]string(nil)` 的原因：两处复制共享同一不变量（结果非 nil 才能 JSON 往返），把不变量写在一处并附注释，可避免后续再次引入空切片复制。

### 决策 2：历史 `null` 值在加载层清理，而不是放宽类型校验

在 `internal/storage/repository.go` 的任务归一化循环中，先调用 `dropNullTaskTemplateFields` 移除值为 `nil` 的字段，再执行 `NormalizeTaskTemplateValues`，并把清理结果计入 `changed` 以触发自愈写回。

`null` 在语义上是"未设置"，模板目录字段本身也没有默认值（目录字段 MUST NOT 保存默认值），移除后由 `ResolveTaskTemplateFields` 按字段类型补齐为空数组。选择在存储层清理而不是让 `NormalizeTaskTemplateValues` 容忍 `nil` 的原因：该函数同时校验前端提交的字段值，容忍 `nil` 会掩盖前端后续可能出现的真实类型错误；清理属于"修复历史写入缺陷"的数据迁移职责，与既有的 `migrateTaskDirectoryLinkSelections` 归为同类。

### 决策 3：自愈写回而不是只读兼容

清理命中时把 `changed` 置为真，沿用既有的 `Load` → `Save` 写回路径，使数据文件在首次以修复版本启动后即恢复干净状态，不需要用户手工编辑文件，也不需要在后续每次加载重复清理。

## Risks / Trade-offs

- [清理 `null` 会丢失"显式清空"语义] → 前端清空目录字段使用空数组，清空字符串字段使用空串，清空布尔字段使用 `false`，不提交 `null`；`null` 只由缺陷版本产生。
- [历史 `null` 对应的模板已被删除，缺失值不会被补齐] → 与既有"兼容没有任务模板的既有数据"行为一致，任务仍可正常加载，只是该字段不再存在。
- [非目录字段的 `null` 也被一并清理] → 清理以"值为 `null`"为唯一条件，符合"未设置"语义；这些字段同样由绑定模板默认值补齐。
- [仅在加载层修复会让其他调用方继续写出 `null`] → 根因已由决策 1 消除，加载层清理只用于处理历史数据。

## Migration Plan

不需要额外的数据迁移操作。升级到修复版本后首次启动时，`Load` 自动清理 `null` 模板字段值并写回数据文件；若升级后出现异常，可用升级前的数据文件备份回滚，修复版本不会改写除 `null` 清理外的任何字段。
