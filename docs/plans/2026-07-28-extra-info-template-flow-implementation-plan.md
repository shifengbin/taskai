# 额外信息模板与信息流实施计划

> **执行要求：** 按本计划逐项完成；每项先写失败测试，再写最小实现并运行对应验证。本文档仅记录实施计划，不在本次变更中直接实现产品代码。

**目标：** 将额外信息拆分为模板、可复用信息和任务快照；提供内置 Git 模板，使任务固定字段只读、动态参数可变。

**架构：** Go 领域层分别建模模板定义、信息固定值和任务快照。仓储在加载 JSON 时迁移当前单层模板并补种 Git 模板。Wails 绑定将模板和信息分别暴露给 React；管理界面先维护模板再维护信息，任务表单从信息创建只读固定字段和可编辑动态参数快照。

**技术栈：** Go、Wails v2、React、TypeScript、MUI、Vitest、OpenSpec。

---

### 任务 1：定义三层领域模型与规则

**文件：**

- 修改：`internal/task/model.go`
- 修改：`internal/task/model_test.go`

**步骤：**

1. 新增失败测试：模板含固定字段默认值和动态参数定义；信息必须有 `name` 值；任务快照只接受信息提供的固定字段；动态参数可附加且键不冲突。
2. 运行 `go test ./internal/task -count=1`，确认测试首先失败。
3. 将当前单层 `ExtraInfoTemplate` 拆分为模板定义与信息值模型；保留任务 `ExtraInfo` 作为完整快照。
4. 在规范化与构造函数中实现 `name`、键唯一性、必填参数及固定字段来源校验。
5. 实现 Git 内置项保护和非内置模板自动 `name` 字段规则。
6. 再次运行 `go test ./internal/task -count=1`。

### 任务 2：实现仓储结构、Git 补种和数据迁移

**文件：**

- 修改：`internal/storage/repository.go`
- 修改：`internal/storage/repository_test.go`

**步骤：**

1. 编写失败测试，覆盖空数据补种 Git、模板与信息的 CRUD、模板存在信息时禁止删除，以及删除信息不改变任务。
2. 增加旧 JSON 的迁移用例：单层模板、旧单键值、同分类不同结构和重复加载的幂等性。
3. 运行 `go test ./internal/storage -count=1`，确认新测试失败。
4. 在 `Data` 中新增独立模板与信息集合，读取时保证非空、执行迁移并原子写回。
5. 迁移中保留旧展示名称为 `name` 值；结构冲突时创建不同模板，而不丢弃字段或参数。
6. 运行 `go test ./internal/storage -count=1`。

### 任务 3：更新应用层、生命周期与 HTTP 输出

**文件：**

- 修改：`internal/application/contracts.go`
- 修改：`internal/lifecycle/service.go`
- 修改：`internal/lifecycle/service_test.go`
- 修改：`app.go`
- 修改：`app_test.go`

**步骤：**

1. 添加失败测试，验证模板/信息绑定、从信息创建快照、任务级动态参数以及删除来源后的历史任务编辑。
2. 运行 `go test . ./internal/lifecycle -count=1`。
3. 暴露模板与信息的列表、保存、删除方法；创建和编辑任务从信息和模板组装快照，并由服务端拒绝固定字段修改。
4. 更新任务详情映射，按 `catalogue` 聚合，并平铺 `name`、固定字段和动态参数值。
5. 运行 `go test . ./internal/lifecycle -count=1` 及 `go test . -run 'TestApp.*ExtraInfo' -count=1`。

### 任务 4：更新 Wails 类型和前端 API

**文件：**

- 修改：`frontend/src/types.ts`
- 修改：`frontend/src/api.ts`
- 重新生成：`frontend/wailsjs/go/main/App.d.ts`
- 重新生成：`frontend/wailsjs/go/main/App.js`
- 重新生成：`frontend/wailsjs/go/models.ts`

**步骤：**

1. 在 `frontend/src/App.test.tsx` 增加模板、信息与任务快照的绑定模拟和失败用例。
2. 运行 `cd frontend && npm test -- --run src/App.test.tsx`。
3. 更新前端领域类型和 API 包装，使模板、信息、任务快照分别构造为对应的 Wails 模型。
4. 通过项目既有 Wails 生成流程刷新绑定，避免手工编辑生成文件。
5. 再次运行 `cd frontend && npm test -- --run src/App.test.tsx`。

### 任务 5：重构额外信息管理界面

**文件：**

- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**

1. 增加失败测试，覆盖 Git 模板初始内容、内置字段不可编辑、自定义模板自动 `name`、固定字段默认值和基于模板创建信息。
2. 运行 `cd frontend && npm test -- --run src/App.test.tsx -t '额外信息|Git'`。
3. 将原“分类 + 信息模板”管理区重构为模板管理和信息管理两层：模板用于定义，信息表单用于填写固定字段。
4. 在信息列表显示 `name` 字段值，并在创建表单中用模板默认值初始化输入框。
5. 再次运行相关 Vitest 用例。

### 任务 6：重构任务附加信息表单

**文件：**

- 修改：`frontend/src/App.tsx`
- 修改：`frontend/src/App.test.tsx`

**步骤：**

1. 增加失败测试，覆盖按模板选择信息、固定字段无编辑控件、`branch` 必填、任务级新增动态参数和跨分类保留选择。
2. 运行 `cd frontend && npm test -- --run src/App.test.tsx -t '动态参数|固定字段|任务'`。
3. 用信息替代旧模板复选项，已选区域展示 `name` 值；固定字段使用只读展示，动态参数区提供添加、编辑、删除。
4. 保持已删除信息或模板来源的历史快照可编辑动态参数。
5. 再次运行完整 `cd frontend && npm test -- --run src/App.test.tsx`。

### 任务 7：更新说明并完成验证

**文件：**

- 修改：`README.md`
- 修改：`openspec/changes/refine-extra-info-template-flow/tasks.md`

**步骤：**

1. 更新 README，说明先建模板、再建信息、最后在任务中选择信息的流程，以及 Git 字段与动态参数的边界。
2. 每完成一个 OpenSpec 工作项，立即在 `tasks.md` 勾选对应复选框。
3. 运行：

```bash
go test -race ./...
cd frontend && npm test -- --run && npm run build
cd .. && openspec validate refine-extra-info-template-flow --strict --no-interactive
git diff --check
./scripts/build-linux.sh
test -x build/bin/taskai
```

4. 仅在用户明确要求时再创建提交；本次不执行 Git 提交。
