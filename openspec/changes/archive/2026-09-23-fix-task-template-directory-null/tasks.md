## 1. 根因定位与基线

- [x] 1.1 复现用户报错，确认数据文件中存在 `templateFields.dir = null` 的任务，并确认错误来自加载归一化
- [x] 1.2 逐层验证保存链路：`MergeTaskTemplateFields` 输出非 nil 空数组，`Save` 内部的二次归一化把空 `[]string` 变成 nil 切片，序列化为 `null`
- [x] 1.3 以最小程序确认 `append([]string(nil), 空切片...)` 返回 nil 切片且序列化为 `null`，加载时成为 nil interface 落入类型校验兜底分支

## 2. 回归测试

- [x] 2.1 增加单元测试：空目录数组经"合并 → JSON 往返 → 归一化"后仍为空数组且加载不报错
- [x] 2.2 增加存储层写入侧测试：保存未选择目录的目录字段后，数据文件中该字段为空数组而非 `null`，且可重新加载
- [x] 2.3 增加存储层加载侧测试：含 `null` 模板字段值的既有数据文件可加载，`null` 值被移除且其他字段值保持不变
- [x] 2.4 确认三个测试在修复前分别失败（报错文案与线上一致），修复后全部通过

## 3. 实现

- [x] 3.1 在 `internal/task/model.go` 新增 `cloneDirectories`，保证目录切片复制结果非 nil
- [x] 3.2 在 `NormalizeTaskTemplateValues` 与 `normalizeTaskTemplateFieldValue` 中改用 `cloneDirectories`
- [x] 3.3 在 `internal/storage/repository.go` 新增 `dropNullTaskTemplateFields`，并在任务归一化前清理 `null` 值、计入 `changed`

## 4. 验证

- [x] 4.1 运行 `go test ./internal/task/ ./internal/storage/`，受影响包测试通过
- [x] 4.2 运行 `go test -race ./...`，全部通过；`go vet ./internal/...` 无输出
- [x] 4.3 运行前端 `npm test`（19 个文件、396 项）与 `npm run build` 通过，确认跨层无回归
- [x] 4.4 使用真实数据文件副本运行完整加载流程：41 条任务正常加载，`null` 被清除并自愈写回
- [x] 4.5 使用 `scripts/build-linux.sh amd64` 构建新版本并启动，确认线上数据文件在启动后不再包含 `null` 模板字段值

## 5. 文档与归档

- [x] 5.1 将最终设计与验证结果记录到中文 `docs/plans/` 实施记录
- [x] 5.2 同步 `openspec/specs/task-templates/spec.md`，并运行 `openspec validate` 通过
- [x] 5.3 提交全部 Git 变更，确认提交只包含本变更相关文件
