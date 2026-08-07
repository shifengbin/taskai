# TaskAI 设计主题库

本目录包含 36 套可直接打开的静态设计预览。根目录的 `index.html` 用作主题总览；每个主题独立保存在 `themes/编号-slug/`，其中包含：

- `index.html`：同页展示亮色与暗色应用窗口，且可通过 `file://` 直接查看。
- `design.md`：主题定位、色彩令牌、排版、组件、交互、可读性和 React + MUI 映射说明。

共享结构、组件样式和交互逻辑位于 `_shared/`；各主题页面仅保留自己的 CSS 令牌、样式与预览配置。`_source-gallery.html` 是拆分时保留的原始单页来源，`scripts/split-themes.mjs` 可按编号区间重新生成页面和设计文档。
