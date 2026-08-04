## 1. 任务项常态底色

- [x] 1.1 在 `frontend/src/components/TaskTree.tsx` 中为未选中的顶层任务行增加由 `taskColor` 派生的约 4% 同色背景，并保留 4px 左边框。
- [x] 1.2 保持现有选中、悬停、拖拽、搁置和开始任务反馈的背景及描边优先级，不改变任务树交互或布局。

## 2. 自动化验证

- [x] 2.1 扩展 `frontend/src/components/TaskTree.test.tsx`，验证普通任务行使用任务色左边框与约 4% 的同色背景。
- [x] 2.2 补充或更新状态相关断言，确认选中和开始反馈仍使用既有语义样式而非被常态底色覆盖。
- [x] 2.3 运行 `cd frontend && npm test && npm run build`，并在亮色与暗色模式下走查任务文字和操作控件的可读性。
