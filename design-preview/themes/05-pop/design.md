# 05 Pop — 活力印刷

## 主题定位

孟菲斯印刷。该方向围绕“Risograph / Memphis Maximalist”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

粗黑描边、错位投影、半色网点、贴纸卡片——把任务管理做得像一本快乐的小册子。

## 色彩令牌

主色与强调色：`#FBF6EC`、`#1B1A2E`、`#FF5470`、`#2EC4B6`、`#FFB13B`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--p-bg` | `#FBF6EC` |
| 亮色 | `--p-surface` | `#FFFFFF` |
| 亮色 | `--p-surface2` | `#FFF1D6` |
| 亮色 | `--p-ink` | `#1B1A2E` |
| 亮色 | `--p-mut` | `#6E6E86` |
| 亮色 | `--p-outline` | `#1B1A2E` |
| 亮色 | `--p-accent` | `#FF5470` |
| 亮色 | `--p-accent2` | `#2EC4B6` |
| 亮色 | `--p-accent3` | `#FFB13B` |
| 亮色 | `--p-accent4` | `#5D5FEF` |
| 亮色 | `--p-dots` | `rgba(27,26,46,.06)` |
| 亮色 | `--p-work` | `#2EC4B6` |
| 亮色 | `--p-idle` | `#B9B5C9` |
| 亮色 | `--p-unread` | `#5D5FEF` |
| 亮色 | `--p-error` | `#FF5470` |
| 暗色 | `--p-bg` | `#1B1A2E` |
| 暗色 | `--p-surface` | `#27254A` |
| 暗色 | `--p-surface2` | `#2F2C50` |
| 暗色 | `--p-ink` | `#F4F1FF` |
| 暗色 | `--p-mut` | `#9C98C0` |
| 暗色 | `--p-outline` | `#0B0A16` |
| 暗色 | `--p-accent` | `#FF5470` |
| 暗色 | `--p-accent2` | `#2EC4B6` |
| 暗色 | `--p-accent3` | `#FFB13B` |
| 暗色 | `--p-accent4` | `#8A8CFF` |
| 暗色 | `--p-dots` | `rgba(244,241,255,.06)` |
| 暗色 | `--p-work` | `#38E6D6` |
| 暗色 | `--p-idle` | `#565279` |
| 暗色 | `--p-unread` | `#8A8CFF` |
| 暗色 | `--p-error` | `#FF6B85` |

## 字体与字重

展示与正文字体分别为：`Bricolage`、`DM Sans`。主题 CSS 实际声明的字体栈包括 `'DM Sans',sans-serif`、`'Bricolage Grotesque'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `700`、`800`、`500`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。孟菲斯印刷在组件形状上使用圆角 `20px`、`8px`、`9px`、`13px`、`11px 11px 0 0`，阴影或描边处理为 `9px 9px 0 var(--p-outline)`、`2px 2px 0 var(--p-outline)`、`3px 3px 0 var(--p-outline)`、`3px 3px 0 var(--p-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--p-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#2EC4B6`、空闲 `#B9B5C9`、未读 `#5D5FEF`、异常 `#FF5470`；暗色：工作中 `#38E6D6`、空闲 `#565279`、未读 `#8A8CFF`、异常 `#FF6B85`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 05 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。
