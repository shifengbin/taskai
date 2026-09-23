# 任务模板目录空值持久化修复实施记录

## 目标

修复"未选择目录的任务被保存后，应用下次启动无法加载数据文件"的阻断级缺陷。线上表现为启动报错 `normalize task template fields: 任务模板字段 "dir" 的值必须是字符串、布尔值或目录数组`，任务数据全部不可用。

## 根因

任务模板目录字段（`directories`）以 `[]string` 保存，保存链路为：

前端提交 → `MergeTaskTemplateFields` → `validateTaskTemplateDirectorySelections` 重建目录值为非 nil 空切片 → `repository.Save` → `normalizeDataForSave` → `NormalizeTaskTemplateValues`。

`NormalizeTaskTemplateValues` 的 `case []string` 分支用 `append([]string(nil), value...)` 复制切片。Go 对空切片执行 append 返回 nil 切片，因此 `Save` 阶段的二次归一化把内存中的非 nil 空数组变成 nil，`json.Marshal` 随后把它写成 `null`。加载时 JSON `null` 解出 nil interface（无动态类型），类型分支全部不匹配而落入兜底分支报错，`Load` 中断。

触发条件是"保存时该字段已经是 Go `[]string` 且为空"，所以 `MergeTaskTemplateFields` 之后的重建逻辑掩盖了缺陷，只有 `Save` 内部的二次归一化会命中。

## 实现

### 根因修复

- `internal/task/model.go` 新增 `cloneDirectories`，以 `append([]string{}, value...)` 作为目录切片复制的唯一入口，保证结果非 nil。
- `NormalizeTaskTemplateValues` 与 `normalizeTaskTemplateFieldValue` 两处复制改用该函数。

### 历史数据自愈

- `internal/storage/repository.go` 新增 `dropNullTaskTemplateFields`，在任务模板字段归一化前移除值为 `null` 的字段，并计入 `changed`。
- `null` 语义为"未设置"，移除后由 `ResolveTaskTemplateFields` 按绑定模板补齐（目录字段补空数组），并沿既有 `Load` → `Save` 路径自愈写回。
- 选择在存储层清理而非让 `NormalizeTaskTemplateValues` 容忍 `nil`，避免掩盖前端后续可能出现的真实类型错误。

## 自动化验证

- `go test ./internal/task/ ./internal/storage/`：受影响包全部通过。
- `go test -race ./...`：全部通过；`go vet ./internal/...` 无输出。
- 前端 `npm test`：19 个测试文件、396 项测试全部通过；`npm run build` 通过。
- 新增回归测试 3 项，修复前分别以"落盘为 `null`"和线上一致的报错文案失败：
  - `TestTaskTemplateValuesKeepEmptyDirectoryArrayAfterJSONRoundTrip`
  - `TestRepositoryPersistsEmptyDirectorySelectionsAsEmptyArray`
  - `TestRepositoryRepairsNullTaskTemplateFieldValues`

## 真实数据验证

使用线上数据文件副本（41 条任务，其中 1 条含 `dir: null`）运行完整加载流程：全部任务正常加载，`null` 被清除，问题任务模板字段恢复为 `{"branch":"web-1.1"}`，文件自愈写回且不含 `null`。

构建 `build/bin/taskai`（版本号 `v0.0.10-local`）并启动后，线上数据文件在启动时完成同样的自愈，应用正常打开。

## 说明

- 本次修复不修改前端、Wails 绑定、HTTP 接口输出与模板定义格式。
- 主规格 `openspec/specs/task-templates/spec.md` 中原有"以空数组初始化目录字段"的要求未变，本次是让实现符合既有规格，并补充目录字段持久化往返不变量与历史 `null` 值清理要求。
