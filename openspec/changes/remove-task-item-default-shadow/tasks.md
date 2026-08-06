## 1. 更新任务项样式

- [x] 1.1 将 `frontend/src/App.css` 中未选中 `.taskai-task-row` 的默认 `box-shadow` 改为仅保留任务色内嵌色条。
- [x] 1.2 保持未选中悬浮、选中默认和选中悬浮的现有位移与外部阴影规则不变。

## 2. 更新测试与规格

- [x] 2.1 更新 `frontend/src/components/TaskTree.test.tsx` 的任务项样式断言，验证默认无外部阴影且保留内嵌色条，并继续验证悬浮和选中阴影。
- [x] 2.2 运行受影响的前端测试与 `openspec validate remove-task-item-default-shadow --strict`，确认实现契约和 OpenSpec 文档均通过验证。
- [x] 2.3 运行前端构建，确认样式变更不会破坏生产构建。
