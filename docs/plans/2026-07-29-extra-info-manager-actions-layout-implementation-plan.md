# 额外信息管理操作栏布局实施计划

> **供 Codex 执行：** 按照测试先行方式逐项完成本计划。

**目标：** 将额外信息的两个创建动作并列展示，并扩展模板和信息编辑弹窗的桌面宽度。

**架构：** 仅修改 `App` 内额外信息管理弹窗的局部布局和两个已有 `Dialog` 的 `maxWidth` 属性。通过语义测试标识验证操作栏归属，通过 MUI 对话框宽度类验证 `md` 宽度，不引入状态、存储和接口修改。

**技术栈：** React、TypeScript、MUI、Vitest、Testing Library。

---

### 任务 1：为操作栏和编辑弹窗宽度添加失败测试

**文件：**
- 修改：`frontend/src/App.test.tsx`
- 修改：`frontend/src/App.tsx`

**步骤 1：编写失败测试**

在额外信息管理用例附近添加两个独立断言：

```tsx
it('将新增模板和新增信息放在同一管理操作栏', async () => {
  // 打开管理弹窗
  const actions = screen.getByTestId('extra-info-manager-actions')
  expect(within(actions).getByRole('button', {name: '新增模板'})).toBeInTheDocument()
  expect(within(actions).getByRole('button', {name: '新增信息'})).toBeInTheDocument()
})

it('模板和信息编辑弹窗使用中等宽度', async () => {
  // 分别打开新增模板与新增信息
  expect(screen.getByRole('dialog', {name: '新增模板'})).toHaveClass('MuiDialog-paperWidthMd')
  expect(screen.getByRole('dialog', {name: '新增信息'})).toHaveClass('MuiDialog-paperWidthMd')
})
```

**步骤 2：运行测试确认失败**

运行：`cd frontend && npm test -- --run src/App.test.tsx -t '管理操作栏|中等宽度'`

预期：失败，当前页面没有操作栏标识且两个弹窗仍为 `MuiDialog-paperWidthSm`。

### 任务 2：实现紧凑操作栏和较宽编辑弹窗

**文件：**
- 修改：`frontend/src/App.tsx:892-1006`

**步骤 1：实现最小布局修改**

将管理弹窗顶部改为包含操作说明和两个创建按钮的横向容器：

```tsx
<Box data-testid="extra-info-manager-actions" sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, flexWrap: 'wrap'}}>
  <Typography variant="subtitle2">管理操作</Typography>
  <Box sx={{display: 'flex', gap: 1}}>
    <Button variant="contained" size="small">新增模板</Button>
    <Button variant="contained" size="small">新增信息</Button>
  </Box>
</Box>
```

保留原有按钮回调与禁用条件；删除分类模板区右侧和信息说明行中的旧按钮。将两个额外信息编辑 `Dialog` 的 `maxWidth` 从 `sm` 改为 `md`。

**步骤 2：运行测试确认通过**

运行：`cd frontend && npm test -- --run src/App.test.tsx -t '管理操作栏|中等宽度'`

预期：两个新增按钮位于同一操作栏，两个弹窗使用 `MuiDialog-paperWidthMd`。

**步骤 3：提交实现**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: 优化额外信息管理操作布局"
```

### 任务 3：执行回归和构建验证

**文件：**
- 无源码修改。

**步骤 1：运行前端全量测试**

运行：`cd frontend && npm test -- --run`

预期：全部测试通过。

**步骤 2：运行 Go 测试**

运行：`go test ./... -count=1`

预期：全部包通过。

**步骤 3：构建 Linux 应用**

运行：`./scripts/build-linux.sh`

预期：生成 `build/bin/taskai`。

**步骤 4：检查工作区**

运行：`git diff --check && git status --short`

预期：无格式问题；仅保留预期的提交内容。
