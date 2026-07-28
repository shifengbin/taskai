# 额外信息分类与多键值实施计划

> **执行要求：** 按本计划逐项完成，每个行为先写失败测试，再写最小实现，并在每项后运行对应测试。

**目标：** 支持独立分类管理、每条额外信息的多固定键值和多动态参数，以及任务表单中的跨分类选择与已选信息移除。

**架构：** 在仓储顶层保存分类名称列表，模板以名称关联分类。领域模型以固定字段数组替代单个固定键值，并在规范化时兼容旧 JSON；任务快照和 HTTP 映射复用该字段数组。前端从 Wails 绑定取得分类与模板，在管理弹窗中维护分类，在任务弹窗中按当前分类筛选并集中展示跨分类已选项。

**技术栈：** Go、Wails v2、React、TypeScript、MUI、Vitest、OpenSpec。

---

### 任务 1：扩展领域模型并兼容旧单键值数据

**文件：**
- 修改：`internal/task/model.go:44-232`
- 修改：`internal/task/model_test.go:55-99`

**步骤 1：先写失败测试**

在 `model_test.go` 新增测试：模板包含两个固定字段和两个参数时可创建快照；固定字段键、参数键在同一信息内重复或冲突时返回错误；传入旧 `Key`、`KeyDisplayName`、`Value` 结构时规范化为一个固定字段。

**步骤 2：运行失败测试**

运行：`go test ./internal/task -run 'TestExtraInfo.*(Fields|Legacy)' -count=1`

预期：失败，当前模型尚未定义固定字段数组或旧字段迁移。

**步骤 3：写最小实现**

新增 `ExtraInfoField`：

```go
type ExtraInfoField struct {
    Key string `json:"key"`
    DisplayName string `json:"displayName"`
    Value string `json:"value"`
}
```

令模板和任务快照保存 `Fields []ExtraInfoField`。保留旧 JSON 字段仅用于读取，在 `NormalizeExtraInfoTemplate` 和 `NormalizeExtraInfo` 中转换为 `Fields`；校验字段键与参数键的非空、唯一和互不冲突。

**步骤 4：运行通过测试**

运行：`go test ./internal/task -count=1`

预期：通过。

### 任务 2：持久化分类并限制删除

**文件：**
- 修改：`internal/storage/repository.go:13-164`
- 修改：`internal/storage/repository_test.go:61-132`

**步骤 1：先写失败测试**

覆盖以下行为：创建和列出唯一分类；同名分类失败；分类下有模板时删除失败；删除全部模板后可删除分类；旧 JSON 未保存分类列表时，从模板分类名称得到可选分类；模板保存时分类不存在失败。

**步骤 2：运行失败测试**

运行：`go test ./internal/storage -run 'TestRepository.*ExtraInfoCatalogue' -count=1`

预期：失败，仓储尚未保存分类或提供 CRUD 方法。

**步骤 3：写最小实现**

在 `Data` 增加 `ExtraInfoCatalogues []string`。增加 `ListExtraInfoCatalogues`、`SaveExtraInfoCatalogue` 和 `DeleteExtraInfoCatalogue`；分类名称规范化、去重，并在删除前检查 `ExtraInfoTemplates`。`Load` 对缺失的分类列表从模板 `Catalogue` 推导，保证旧数据可选可保存。

**步骤 4：运行通过测试**

运行：`go test ./internal/storage -count=1`

预期：通过。

### 任务 3：暴露分类绑定并保持任务快照语义

**文件：**
- 修改：`internal/application/contracts.go`
- 修改：`internal/lifecycle/service.go`
- 修改：`internal/lifecycle/service_test.go`
- 修改：`app.go`
- 修改：`app_test.go:282-415`

**步骤 1：先写失败测试**

增加应用层测试，通过 App/Wails 方法创建、列出、删除分类，确认有模板时删除失败、清空模板后删除成功；同时确认多固定字段快照创建、更新和模板删除后历史任务不变。

**步骤 2：运行失败测试**

运行：`go test . ./internal/lifecycle -run '(TestApp.*ExtraInfoCatalogue|Test.*ExtraInfo)' -count=1`

预期：失败，应用绑定和生命周期尚未公开分类操作。

**步骤 3：写最小实现**

在仓储接口、生命周期服务和 `App` 添加 `ListExtraInfoCatalogues`、`SaveExtraInfoCatalogue`、`DeleteExtraInfoCatalogue`。模板保存经服务层检查关联分类；创建和编辑任务继续只保存完整快照。

**步骤 4：运行通过测试**

运行：`go test . ./internal/lifecycle -count=1`

预期：通过。

### 任务 4：更新任务详情 HTTP 聚合

**文件：**
- 修改：`app.go` 中任务资源映射
- 修改：`app_test.go:360-415`
- 修改：`internal/realtime/http_test.go`（如需补充 HTTP 边界断言）

**步骤 1：先写失败测试**

建立一个含两个固定字段和参数的 Git 快照，断言任务详情中的 `extraInfo.git[0]` 是包含全部键值的扁平对象，且不包含显示名称或旧单键值包装。

**步骤 2：运行失败测试**

运行：`go test . -run TestAppHTTPTaskDetailFlattensExtraInfoByCatalogue -count=1`

预期：失败，现有映射只读取单个 `Key`、`Value`。

**步骤 3：写最小实现**

遍历快照 `Fields` 写入响应对象，再遍历 `Parameters` 写入参数值；保持按 `Catalogue` 追加数组和空附加信息省略逻辑。

**步骤 4：运行通过测试**

运行：`go test . -run TestAppHTTPTaskDetailFlattensExtraInfoByCatalogue -count=1`

预期：通过。

### 任务 5：更新前端类型和 Wails API 边界

**文件：**
- 修改：`frontend/src/types.ts:19-55`
- 修改：`frontend/src/api.ts:1-55`
- 自动生成并检查：`frontend/wailsjs/go/main/App.d.ts`、`frontend/wailsjs/go/main/App.js`、`frontend/wailsjs/go/models.ts`

**步骤 1：先写失败测试**

在 `frontend/src/App.test.tsx` 的绑定模拟中增加分类操作，构造带 `fields` 的模板。添加 API/组件场景，使分类列表和多字段模板能进入表单。

**步骤 2：运行失败测试**

运行：`npm test -- --run src/App.test.tsx -t '分类'`

预期：失败，类型和 API 尚未包含分类与字段数组。

**步骤 3：写最小实现**

定义 `ExtraInfoField`、`TaskExtraInfoField` 和分类 API 包装；所有与 Wails 交互的模板/快照使用新模型构造器。运行 Wails 生成绑定，而不手改生成文件。

**步骤 4：运行通过测试**

运行：`npm test -- --run src/App.test.tsx -t '分类'`

预期：通过。

### 任务 6：实现分类管理和多固定字段编辑

**文件：**
- 修改：`frontend/src/App.tsx:60-1200`
- 修改：`frontend/src/App.test.tsx:303-400`

**步骤 1：先写失败测试**

覆盖以下流程：在额外信息管理弹窗新建分类；信息编辑时分类为下拉选择而不是自由输入；增加两行固定字段并保存；删除仍被模板使用的分类显示错误；删除该分类的最后一条信息后可删除分类。

**步骤 2：运行失败测试**

运行：`npm test -- --run src/App.test.tsx -t '额外信息管理|分类'`

预期：失败，当前表单使用自由文本分类并只允许一个固定键值。

**步骤 3：写最小实现**

在独立管理弹窗顶部放置分类 `Select`、新建和删除图标按钮；在信息表单中以固定字段行替换单一键/显示名称/值输入，并支持增加、移除和错误展示。删除分类只调用后端，错误通过现有消息栏展示。

**步骤 4：运行通过测试**

运行：`npm test -- --run src/App.test.tsx -t '额外信息管理|分类'`

预期：通过。

### 任务 7：实现按分类选择和已选信息标签

**文件：**
- 修改：`frontend/src/App.tsx:620-1050`
- 修改：`frontend/src/App.test.tsx:347-400`

**步骤 1：先写失败测试**

准备 Git 与环境两个分类的模板。断言选择 Git 信息后切换至环境分类，Git 勾选保留；顶部出现两个紧凑标签；点击 Git 标签的移除按钮会取消该信息、保留环境信息及其参数；创建任务时传入两项快照和各自参数。

**步骤 2：运行失败测试**

运行：`npm test -- --run src/App.test.tsx -t '跨分类|已选信息'`

预期：失败，当前界面同时展示所有分类且没有已选信息集中区域。

**步骤 3：写最小实现**

添加任务表单的当前分类状态，默认选第一个分类；当前分类下以复选框显示模板。使用小尺寸、低圆角 `Chip` 在上方渲染所有已选模板，`onDelete` 调用现有移除逻辑；保留已删除模板快照的编辑支持。

**步骤 4：运行通过测试**

运行：`npm test -- --run src/App.test.tsx -t '跨分类|已选信息'`

预期：通过。

### 任务 8：更新 OpenSpec、说明和完整验证

**文件：**
- 修改：`openspec/changes/add-task-extra-info/proposal.md`
- 修改：`openspec/changes/add-task-extra-info/design.md`
- 修改：`openspec/changes/add-task-extra-info/specs/task-extra-info/spec.md`
- 修改：`openspec/changes/add-task-extra-info/tasks.md`
- 修改：`README.md`

**步骤 1：更新中文规格与任务**

为分类 CRUD、删除限制、多固定键值、旧数据兼容和跨分类选择补充需求场景；在 `tasks.md` 添加未完成实施任务，完成时即时勾选。

**步骤 2：执行完整验证**

运行：

```bash
go test -race ./...
cd frontend && npm test -- --run && npm run build
cd .. && openspec validate add-task-extra-info --strict
git diff --check
./scripts/build-linux.sh
test -x build/bin/taskai
```

预期：全部通过，且新的可执行程序位于 `build/bin/taskai`。
