## 1. 开发准备

- [x] 1.1 在仓库 `.worktrees/` 下创建隔离的 `collapse-terminal-font-picker` worktree 分支，并确认不带入无关工作区改动。

## 2. 前端交互与测试

- [x] 2.1 先扩展 `frontend/src/App.test.tsx`：覆盖默认仅显示当前字体摘要、展开后显示单选候选、选择后自动收起，以及取消不持久化。
- [x] 2.2 为字体发现失败和已保存字体不可用的场景补充或调整测试，确认收起摘要仍保留当前选择与状态提示。
- [x] 2.3 在 `frontend/src/App.tsx` 增加仅用于设置弹窗的受控字体选择器展开状态，并在打开设置时重置为收起。
- [x] 2.4 使用已有 `SnapAccordion` 渲染当前字体摘要和展开的候选列表；摘要与列表均从 `fontOptions` 派生，并保留加载、失败、默认回退与不可用字体的行为。
- [x] 2.5 将候选项选择处理改为更新 `terminalFontFamily` 草稿后立即收起列表，保留 `radiogroup`、`radio`、触发器展开状态和既有保存/取消语义。

## 3. 验证与合并

- [x] 3.1 在 worktree 中运行受影响的前端测试，并运行 `cd frontend && npm test && npm run build`。
- [x] 3.2 将当前基线分支合并到 worktree 分支，解决冲突后再次验证变更。
- [ ] 3.3 将已验证的 worktree 分支合并回当前分支，并使用 `scripts/` 中适用的编译脚本完成项目编译。
