## 1. 左侧面板与任务树布局

- [ ] 1.1 将 `App` 左侧任务面板调整为固定工具栏和可收缩剩余区域的两行栅格，移除依赖 `calc()` 的高度传递，并为相关容器补齐最小高度约束。
- [ ] 1.2 调整 `TaskTree` 导航与任务列表的高度和溢出规则，使三个状态标签下的长列表仅在列表区域内垂直滚动。
- [ ] 1.3 为任务列表添加跨 WebView 的隐藏滚动条样式，同时保留原生滚动交互并避免横向溢出。

## 2. 回归测试与验证

- [ ] 2.1 补充前端组件测试，覆盖三种状态下长任务列表的可滚动容器约束、隐藏滚动条样式及滚动后既有任务树交互。
- [ ] 2.2 运行 `cd frontend && npm test && npm run build`，确认前端测试与 TypeScript/Vite 构建通过。
- [ ] 2.3 运行 `openspec validate fix-task-list-scrolling --strict`，确认变更产物符合 OpenSpec schema。
- [ ] 2.4 运行 `./scripts/build-linux.sh`，确认项目 Linux 构建成功并生成 `build/bin/taskai`。
