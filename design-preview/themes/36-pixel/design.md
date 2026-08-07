# 36 Pixel — 像素糖果

## 主题定位

像素糖果。该方向围绕“8-bit Candy Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

像素糖果：品红、青、明黄、草绿的街机糖果色，Titan One 厚重圆体配 Chakra Petch——复古游戏机的甜。

## 色彩令牌

主色与强调色：`#F0F4FF`、`#0D0B2A`、`#FF2BB1`、`#00C2FF`、`#FFD400`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--px-bg` | `#F0F4FF` |
| 亮色 | `--px-surface` | `#FFFFFF` |
| 亮色 | `--px-surface2` | `#E2EAFB` |
| 亮色 | `--px-ink` | `#0D0B2A` |
| 亮色 | `--px-mut` | `#5A5680` |
| 亮色 | `--px-outline` | `#0D0B2A` |
| 亮色 | `--px-accent` | `#FF2BB1` |
| 亮色 | `--px-accent2` | `#00C2FF` |
| 亮色 | `--px-accent3` | `#FFD400` |
| 亮色 | `--px-accent4` | `#22C55E` |
| 亮色 | `--px-dots` | `rgba(13,11,42,.06)` |
| 亮色 | `--px-work` | `#15803D` |
| 亮色 | `--px-idle` | `#AEABC8` |
| 亮色 | `--px-unread` | `#FF2BB1` |
| 亮色 | `--px-error` | `#E11D48` |
| 暗色 | `--px-bg` | `#0B0A20` |
| 暗色 | `--px-surface` | `#141233` |
| 暗色 | `--px-surface2` | `#1D1A45` |
| 暗色 | `--px-ink` | `#EAF0FF` |
| 暗色 | `--px-mut` | `#8E8AC0` |
| 暗色 | `--px-outline` | `#EAF0FF` |
| 暗色 | `--px-accent` | `#FF5CC4` |
| 暗色 | `--px-accent2` | `#33D0FF` |
| 暗色 | `--px-accent3` | `#FFDF3A` |
| 暗色 | `--px-accent4` | `#3DD97A` |
| 暗色 | `--px-dots` | `rgba(234,240,255,.06)` |
| 暗色 | `--px-work` | `#3DD97A` |
| 暗色 | `--px-idle` | `#4E4A70` |
| 暗色 | `--px-unread` | `#FF5CC4` |
| 暗色 | `--px-error` | `#FF5C7A` |

## 字体与字重

展示与正文字体分别为：`Titan`、`Chakra`。主题 CSS 实际声明的字体栈包括 `'Chakra Petch',sans-serif`、`'Titan One',sans-serif`、`'Titan One'`、`'Chakra Petch'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `400`、`600`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。像素糖果在组件形状上使用圆角 `8px`、`6px`、`9px`、`6px 6px 0 0`，阴影或描边处理为 `8px 8px 0 var(--px-outline)`、`2px 2px 0 var(--px-outline)`、`3px 3px 0 var(--px-outline)`、`3px 3px 0 var(--px-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--px-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#15803D`、空闲 `#AEABC8`、未读 `#FF2BB1`、异常 `#E11D48`；暗色：工作中 `#3DD97A`、空闲 `#4E4A70`、未读 `#FF5CC4`、异常 `#FF5C7A`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 36 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；像素糖果以本页令牌、标题字形和背景处理形成独立辨识度。
