## 1. 组件行为测试

- [x] 1.1 在 `frontend/src/components/TerminalName.test.tsx` 增加行内会话详情方式的失败测试，覆盖精确文本 `前端调试(npm run dev:zsh)`、启动命令缺省文本、不渲染 Tooltip、无别名时保持实际名称以及编辑时只使用别名。
- [x] 1.2 在 `frontend/src/components/TerminalView.test.tsx` 将右侧默认提示定位测试改为行内会话详情测试，并断言标题中没有固定的 `原标题` 文字。
- [x] 1.3 运行 `cd frontend && npm test -- TerminalName.test.tsx TerminalView.test.tsx`，确认新增断言在实现前按预期失败且任务树既有提示测试不受影响。

## 2. 右侧标题行内展示

- [x] 2.1 为 `frontend/src/components/TerminalName.tsx` 增加调用位置可选的详情展示方式；默认保留现有两行 Tooltip，行内方式仅在非空别名且未编辑时生成 `别名(实际标题:启动命令)`。
- [x] 2.2 在 `frontend/src/components/TerminalView.tsx` 为右侧标题启用行内会话详情方式，保留现有单行裁剪、双击编辑和标题栏布局。
- [x] 2.3 运行 `cd frontend && npm test -- TerminalName.test.tsx TerminalView.test.tsx TaskTree.test.tsx`，确认右侧标题和任务树提示测试全部通过。

## 3. 回归与交付验证

- [x] 3.1 运行 `cd frontend && npm test && npm run build`，确认完整前端测试与生产构建通过。
- [x] 3.2 使用 `wails dev` 启动应用，通过浏览器验证右侧别名标题显示实际标题和启动命令、没有固定的 `原标题` 文字、悬浮不出现提示、双击编辑正常，且任务树仍显示两行悬浮提示；验证后关闭调试进程。
- [ ] 3.3 使用 `scripts` 下的项目编译脚本生成并打开可执行程序，复验右侧标题在亮色、暗色和较窄窗口下的显示与裁剪，等待用户确认。
- [ ] 3.4 按项目工作区流程完成分支合并后重新编译验证，同步实施记录和 OpenSpec 任务状态，并在功能确认后归档本变更。
