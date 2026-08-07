# 12 Sunny — 渐变明媚

## 主题定位

明亮消费品。该方向围绕“Vibrant Gradient Consumer”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

落日橘粉与海洋蓝青的渐变点缀，白色卡片 + 柔和阴影——明媚又不失专业，关键操作用渐变实心按钮，醒目好认。

## 色彩令牌

主色与强调色：`linear-gradient(135deg,#FF7A45,#FF3D8B)`、`linear-gradient(135deg,#3B82F6,#06B6D4)`、`#6D5EF5`、`#FBF6FF`、`#15111F`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--g-bg` | `#FBF6FF` |
| 亮色 | `--g-surface` | `#FFFFFF` |
| 亮色 | `--g-surface2` | `#F4EEFF` |
| 亮色 | `--g-ink` | `#241B3A` |
| 亮色 | `--g-mut` | `#6E6488` |
| 亮色 | `--g-line` | `#ECE3FA` |
| 亮色 | `--g-accent` | `#FF3D8B` |
| 亮色 | `--g-accent2` | `#6D5EF5` |
| 亮色 | `--g-grad-sun` | `linear-gradient(135deg,#FF7A45,#FF3D8B)` |
| 亮色 | `--g-grad-ocean` | `linear-gradient(135deg,#3B82F6,#06B6D4)` |
| 亮色 | `--g-work` | `#16A34A` |
| 亮色 | `--g-idle` | `#C3BBD6` |
| 亮色 | `--g-unread` | `#6D5EF5` |
| 亮色 | `--g-error` | `#F43F5E` |
| 暗色 | `--g-bg` | `#15111F` |
| 暗色 | `--g-surface` | `#1F1830` |
| 暗色 | `--g-surface2` | `#291F3E` |
| 暗色 | `--g-ink` | `#F4EFFE` |
| 暗色 | `--g-mut` | `#9C93BE` |
| 暗色 | `--g-line` | `#332A4B` |
| 暗色 | `--g-accent` | `#FF5C9C` |
| 暗色 | `--g-accent2` | `#8B7CFF` |
| 暗色 | `--g-grad-sun` | `linear-gradient(135deg,#FF8A5C,#FF4D94)` |
| 暗色 | `--g-grad-ocean` | `linear-gradient(135deg,#4D8DFF,#1FCDE0)` |
| 暗色 | `--g-work` | `#22C55E` |
| 暗色 | `--g-idle` | `#564C72` |
| 暗色 | `--g-unread` | `#8B7CFF` |
| 暗色 | `--g-error` | `#FB4869` |

## 字体与字重

展示与正文字体分别为：`Lexend`、`Plus Jakarta`。主题 CSS 实际声明的字体栈包括 `'Plus Jakarta Sans',sans-serif`、`'Lexend'`、`'Plus Jakarta Sans'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `500`、`700`、`600`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。明亮消费品在组件形状上使用圆角 `20px`、`10px`、`11px`、`3px`、`13px`，阴影或描边处理为 `0 30px 70px -34px rgba(80,40,140,.4)`、`0 6px 16px -4px rgba(255,61,139,.5)`、`inset 4px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#16A34A`、空闲 `#C3BBD6`、未读 `#6D5EF5`、异常 `#F43F5E`；暗色：工作中 `#22C55E`、空闲 `#564C72`、未读 `#8B7CFF`、异常 `#FB4869`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 12 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。
