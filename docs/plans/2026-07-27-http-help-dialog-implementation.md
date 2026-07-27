# HTTP 接口说明弹窗 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将设置中的 HTTP 接口说明改造成按配置、查询、更新和规则分组的易读弹窗。

**Architecture:** 保持现有 `statusHelpOpen` 状态和 `Dialog` 结构，只替换弹窗内容为本地可复用的小型展示组件与 MUI 布局。接口文字直接对应现有 HTTP 契约，不更改后端、Wails 绑定或状态逻辑。

**Tech Stack:** React、TypeScript、MUI、Vitest、Testing Library。

---

### Task 1: 覆盖结构化 HTTP 说明

**Files:**
- Modify: `frontend/src/App.test.tsx:451-479`
- Modify: `frontend/src/App.tsx:827-841`

**Step 1: 编写失败测试**

在现有“在设置中配置 HTTP 状态管理并查看接口说明”测试中，打开说明弹窗后断言以下可见内容：

```ts
expect(await screen.findByText('服务与设置')).toBeInTheDocument()
expect(screen.getByText('查询接口')).toBeInTheDocument()
expect(screen.getByText('状态更新')).toBeInTheDocument()
expect(screen.getByText('GET /api/v1/tasks?status=pending|running|completed')).toBeInTheDocument()
expect(screen.getByText('GET /api/v1/tasks/:taskId')).toBeInTheDocument()
```

**Step 2: 运行测试确认失败**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：测试因新的结构化分组标题和接口条目尚不存在而失败。

**Step 3: 最小实现**

在 `App.tsx` 的 HTTP 说明弹窗中：

```tsx
<Typography variant="overline">服务与设置</Typography>
<Alert severity="info">服务仅监听 127.0.0.1，且无需鉴权。</Alert>
<Box component="section" sx={{display: 'grid', gap: 1}}>
  <Typography variant="subtitle2">查询接口</Typography>
  <ApiEndpoint method="GET" path="/api/v1/tasks?status=pending|running|completed" description="按生命周期筛选任务；省略 status 返回全部任务。"/>
</Box>
```

定义仅在该文件使用的 `ApiEndpoint` 展示组件：方法 `Chip`、等宽路径和正文说明可折行排列。为服务说明、环境变量、查询、更新、状态规则分别建立 `section`，将已有的 `curl` 示例移动到相应的更新区块。

**Step 4: 运行通过测试**

运行：

```bash
cd frontend && npm test -- --run src/App.test.tsx
```

预期：所有 `App` 测试通过。

**Step 5: 完整验证**

运行：

```bash
cd frontend && npm test && npm run build
git diff --check
```

预期：前端测试、TypeScript 构建和差异检查全部通过。
