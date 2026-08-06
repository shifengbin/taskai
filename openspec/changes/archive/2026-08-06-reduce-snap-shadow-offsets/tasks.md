## 1. Snap 阴影令牌

- [x] 1.1 在 `frontend/tailwind.config.cjs` 中将 `shadow-snap-sm`、`shadow-snap`、`shadow-snap-lg` 的零模糊偏移分别更新为 1px、2px、3px，并保留阴影颜色。
- [x] 1.2 确认任务项、按钮、图标按钮、表单控件与覆盖层继续通过共享令牌获得对应的默认、交互和覆盖层级。

## 2. 验证

- [x] 2.1 运行前端构建与现有测试，确认令牌调整不影响编译或交互行为。
- [x] 2.2 在亮色和暗色模式下检查任务项、按钮、菜单、Popover、Tooltip 与 Dialog，确认硬投影更小但描边、层级、悬停抬起和可读性仍清晰。
