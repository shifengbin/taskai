## Why

目录字段（`directories`）未选择目录时，前端提交空数组。保存链路中复制目录切片使用 `append([]string(nil), value...)`，对空切片 append 返回 nil 切片；`Save` 内部的二次归一化再次归一化这个 nil 后，`json.Marshal` 把它写成 `null` 落盘。下次启动 `Load` 读到 `null` 得到 nil interface，任务模板字段类型校验落到兜底分支直接报错，整个数据文件无法加载。

实际影响：用户在 v0.0.9 中新建任务且不选择目录，数据文件即被写入 `null`，应用下次启动报 `normalize task template fields: 任务模板字段 "dir" 的值必须是字符串、布尔值或目录数组`，任务数据全部不可用，属于阻断级缺陷。已确认线上用户数据文件中存在该形态的 `null`（41 条任务中 1 条）。

## What Changes

- 复制目录值时保证结果非 nil：未选择目录的字段持久化为空数组 `[]`，不再写出 `null`。
- 加载既有数据文件时清理历史版本写入的 `null` 任务模板字段值：`null` 视为未设置，移除后由绑定模板的默认值补齐，并触发一次自愈写回。
- 增加写入侧与加载侧回归测试，覆盖"空目录数组落盘为空数组"和"含 `null` 的历史文件可加载并修复"。

## Capabilities

### New Capabilities

### Modified Capabilities

- `task-templates`：增加目录字段空值持久化与"保存 → 重新加载"往返不变量；扩展既有数据兼容要求，覆盖 `templateFields` 中的 `null` 值清理。

## Impact

- `internal/task/model.go`：新增 `cloneDirectories`，两处目录切片复制改为保证非 nil。
- `internal/storage/repository.go`：新增 `dropNullTaskTemplateFields`，在任务模板字段归一化前清理 `null` 值并标记数据已变更。
- `internal/task/task_template_test.go`、`internal/storage/repository_test.go`：补充回归测试。
- 不修改前端、Wails 绑定、HTTP 接口输出格式、模板定义格式或生命周期行为。
