## 1. 终端恢复与重绘

- [x] 1.1 调整 `TerminalView` 的初始化顺序：在 xterm 打开后先按内容容器适配尺寸，再回放该终端的累计输出，同时保留后续布局稳定后的尺寸校正与现有 PTY 尺寸同步。
- [x] 1.2 在初始历史输出解析完成后安全地刷新所有可见 xterm 行；快速切换或组件卸载后不得操作已释放实例，并保持自动聚焦、输入、剪贴板和实时增量输出行为。
- [x] 1.3 扩展终端内容容器的尺寸观察处理，使窗口或面板宽度变化后执行尺寸适配并刷新可见行，而不让普通实时输出触发整屏刷新。

## 2. 前端回归测试

- [x] 2.1 扩展 `TerminalView` 测试 mock，支持可控制的初始写入完成回调、`refresh` 调用和可触发的 `ResizeObserver`。
- [x] 2.2 新增终端 A -> B -> A 切换回归测试，验证没有窗口尺寸变化时，每个重新挂载实例都会完整刷新含静态历史输出的可见行。
- [x] 2.3 新增容器尺寸变化与后续实时输出测试，验证尺寸观察会适配并刷新，普通增量输出仍不会触发额外整屏刷新。

## 3. 验证

- [x] 3.1 运行 `cd frontend && npm test -- --run src/components/TerminalView.test.tsx`，确认终端恢复与重绘测试通过。
- [x] 3.2 运行 `cd frontend && npm test -- --run src/App.test.tsx` 和 `npm run build`，确认应用集成行为及 TypeScript/Vite 构建通过。
- [x] 3.3 运行 `openspec validate fix-terminal-switch-rendering --strict`，确认变更工件符合 schema。
- [x] 3.4 运行 `./scripts/build-linux.sh`，并在 Linux Wails 应用中手工验证：多个已有终端反复切换后，静态区域不会空白且无需调整窗口大小。
