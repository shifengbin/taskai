# 额外信息管理检索与布局 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 改善额外信息模板参数布局、按模板折叠管理信息，并在管理页和任务选择页支持按名称检索。

**架构：** 仅调整 React 组件状态和展示层。信息仍通过 `templateId` 归属模板，`name` 固定字段仍是唯一的展示和搜索来源；任务快照继续保存完整固定字段，但任务表单仅编辑动态参数。

**技术栈：** React、TypeScript、MUI、Vitest。

---

### 任务 1：补充管理页与任务筛选的失败测试

**文件：**

- 修改：`frontend/src/App.test.tsx`

**步骤：**

1. 为“新增信息”补充测试：点击全局入口后先选择模板，随后显示固定字段默认值并可保存。
2. 为任务信息选择补充测试：搜索当前分类中的名称、切换分类后保留搜索词、清空后继续选择其他分类信息。
3. 断言候选项、已选标签只展示名称，勾选后的编辑区只包含动态参数，不显示固定字段。
4. 运行 `cd frontend && npm test -- --run src/App.test.tsx`，确认测试先因现有界面失败。

### 任务 2：重构信息管理和信息编辑流程

**文件：**

- 修改：`frontend/src/App.tsx`

**步骤：**

1. 新增管理页搜索、展开模板集合和新建信息所选模板的状态。
2. 使用折叠面板按 `templateId` 渲染信息；Git 首次默认展开，搜索命中自动展开其所属分组。
3. 将全局“新增信息”改为模板选择对话框步骤，选中模板后用默认值创建草稿；编辑已有信息保持直接打开。
4. 将模板动态参数编辑行改成响应式网格与独立操作行。
5. 运行任务 1 中的前端测试，确认通过。

### 任务 3：收敛任务信息的展示与搜索

**文件：**

- 修改：`frontend/src/App.tsx`

**步骤：**

1. 在 `TaskExtraInfoEditor` 中增加当前分类的名称搜索输入，并使分类切换不重置该值或已选项。
2. 将已选标签改为仅显示 `displayName`。
3. 仅渲染动态参数输入，保留固定字段在提交快照中但不在任务表单显示。
4. 将动态参数行调整为响应式布局，确保新增参数在窄宽度下不重叠。
5. 运行 `cd frontend && npm test -- --run src/App.test.tsx`。

### 任务 4：完成全量验证

**文件：**

- 如测试发现必要问题，修改：`frontend/src/App.tsx`、`frontend/src/App.test.tsx`

**步骤：**

1. 运行 `go test ./... -count=1`。
2. 运行 `cd frontend && npm test -- --run && npm run build`。
3. 运行 `openspec validate refine-extra-info-template-flow --strict --no-interactive`。
4. 运行 `./scripts/build-linux.sh`，并以 `test -x build/bin/taskai` 验证产物存在。
5. 运行 `git diff --check`，处理本轮引入的格式问题。
