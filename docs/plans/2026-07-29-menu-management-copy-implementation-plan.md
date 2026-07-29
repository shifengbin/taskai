# 菜单管理文案调整实施计划

> **执行要求：** 使用 `@superpowers:executing-plans` 按任务逐项实施。

**目标：** 将设置中维护任务菜单的入口命名为“菜单管理”，并保持任务运行时的“任务操作”名称不变。

**架构：** 只调整 `App.tsx` 的显示文案，不改内部 `menu` Tab 值、配置模型或持久化流程。前端组件测试通过设置弹窗验证新名称及其说明；README 同步列出新的设置 Tab 名称。

**技术栈：** React、TypeScript、Material UI、Vitest、Wails。

---

### 任务 1：锁定设置入口文案

**文件：**
- 修改：`frontend/src/App.test.tsx`

**步骤 1：编写失败测试**

在设置测试附近新增用例，打开设置后断言存在“菜单管理”Tab，不存在“任务操作”Tab；进入新 Tab 后断言显示“右键菜单与任务操作下拉菜单共用此顺序。系统项仅可调序。”。

**步骤 2：运行测试并确认失败**

运行：`npm test -- App.test.tsx -t "将任务菜单配置显示为菜单管理"`

预期：测试因找不到“菜单管理”Tab 失败。

### 任务 2：调整设置文案

**文件：**
- 修改：`frontend/src/App.tsx:1061`
- 修改：`README.md:15`

**步骤 1：实现最小修改**

将 `value="menu"` 的 Tab 标签从“任务操作”改为“菜单管理”；保留该设置区说明中的“任务操作下拉菜单”，以指向任务树运行时按钮。README 的四个设置 Tab 名称同步为“菜单管理”。

**步骤 2：运行针对性测试并确认通过**

运行：`npm test -- App.test.tsx -t "将任务菜单配置显示为菜单管理"`

预期：测试通过。

### 任务 3：回归与构建验证

**文件：**
- 无额外修改。

**步骤 1：执行前端测试**

运行：`npm test`

预期：所有前端测试通过。

**步骤 2：执行项目编译脚本**

运行：`./scripts/build-linux.sh`

预期：Linux 应用编译成功，并生成 `build/bin/taskai`。

**步骤 3：提交变更**

运行：`git add frontend/src/App.tsx frontend/src/App.test.tsx README.md docs/plans/2026-07-29-menu-management-copy-implementation-plan.md && git commit -m "feat: 将任务操作设置改为菜单管理"`

提交前只暂存上述文件，保留工作区其他未提交改动。
