## 为什么

当前应用基于 MUI v9 + Emotion 构建，使用「庭院」鼠尾草绿视觉系统。用户决定按 `design-preview` 第 29 个方案 **Snap — 快门波普** 重构，并要求**直接替换 UI 库**而非在 MUI 上贴皮：因为 MUI 的 Material 设计语言（柔和圆角、elevation 软阴影、ripple）与新粗野（Neo-brutalism：粗描边 + 错位硬投影 + 抬起）结构性对抗，覆盖成本高且永远在"打补丁"。改用 shadcn 模式（Radix 无样式无障碍原语 + Tailwind + lucide-react）后，快门波普成为组件的默认写法，而非对组件库的对抗。

## 变更内容

- **移除 MUI 家族**：卸载 `@mui/material`、`@mui/icons-material`、`@emotion/react`、`@emotion/styled`，删除 `createAppTheme` 与 `ThemeProvider`。
- **引入 shadcn 技术栈**：Tailwind CSS（快门波普令牌、字体、圆角、暗色 class 策略）+ `@radix-ui/react-*` 无障碍原语（Dialog/Tabs/Accordion/Tooltip/Menu/Popover/Switch/Checkbox/Select/Slider/ScrollArea 等）+ `lucide-react` 粗描边图标 + `class-variance-authority`/`clsx`/`tailwind-merge`。
- **建立快门波普组件库**：参照 `design-preview/index.html`（已是纯 HTML/CSS 写好的整套 Snap 组件源码）移植为自有 React 组件（`src/components/ui/`），每个组件原生体现 2–2.5px 描边、`Npx Npx 0` 硬投影、6–9px 圆角、hover 抬起。
- **令牌架构迁移**：由 `createAppTheme` 的 MUI palette 改为 Tailwind 配置 + `:root`/`.dark` CSS 变量；现有 `colorScheme`（亮/暗）映射到 Tailwind 暗色 class。亮色以近白画布 `#F1F5F4`、墨黑 `#10212B`、珊瑚 `#FF5A4E`、钴蓝 `#1E66F5`；暗色提供对应深夜版令牌。
- **重写应用界面**：`App.tsx`（外壳、任务详情、全部弹窗）、`TaskTree.tsx`、`TerminalView.tsx`、`TerminalStatusDot.tsx` 由 MUI 组件改为 Snap 组件 + Tailwind。
- **字体**：标题 Hanken Grotesk、正文 Plus Jakarta Sans、终端 JetBrains Mono，三套 woff2 离线自托管；中文走系统 Noto Sans SC 回退。
- **终端**：通过 xterm `ITheme` 注入快门波普配色（提示符珊瑚、成功钴蓝、关键字/光标紫罗兰）+ 半色网点底纹；终端行为不变。
- 关键文字与背景对比度 ≥ 4.5:1；保留 `prefers-reduced-motion` 静态等价反馈。
- 保持任务、终端、设置、生命周期、额外信息、菜单和窗口退出的全部业务逻辑、数据结构、API 调用、键盘行为及控件语义不变。

## 能力

### 新增能力

- 无。

### 修改能力

- `pine-night-run-visual-system`：把视觉系统由庭院鼠尾草改为快门波普，并把组件基础由 MUI 迁移到 Radix + Tailwind，补充图标、字体、硬投影皮肤与终端配色要求。（注：能力名为历史命名，沿用既有规格文件不改名。）

## 影响

- **依赖变更**：移除 `@mui/material`、`@mui/icons-material`、`@emotion/react`、`@emotion/styled`；新增 `tailwindcss` 及 `@tailwindcss/vite`、`@radix-ui/react-*`、`lucide-react`、`class-variance-authority`、`clsx`、`tailwind-merge`。
- **重写**：`frontend/src/App.tsx`、`frontend/src/components/TaskTree.tsx`、`frontend/src/components/TerminalView.tsx`、`frontend/src/components/TerminalStatusDot.tsx`。
- **样式**：`App.css`/`style.css` 改为 Tailwind 层 + 快门波普令牌 CSS；移除 `createAppTheme` 与全部 MUI 组件覆盖。
- **新增**：`src/components/ui/` 下 Snap 组件库；`tailwind.config` 与令牌 CSS；`assets/fonts/` 三套字体 woff2。
- **重绘**：`assets/task-ai-mark.svg`。
- **测试**：`App.test.tsx` 及组件测试中基于 MUI class/结构的断言需更新为 Radix/Tailwind 等价；role/aria 的无障碍断言大多可直接复用。
- 不增加后端接口、数据迁移或配置字段；复用现有 `colorScheme`。
