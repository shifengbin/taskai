# 33 Crayon — 蜡笔波普

## 主题定位

蜡笔涂绘。该方向围绕“Crayon Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

蜡笔波普：蜡笔红、天蓝、明黄、草绿，像打开一盒新蜡笔——Fredoka 圆润童趣，全场最阳光活泼的一款。

## 色彩令牌

主色与强调色：`#FFFCF3`、`#1F1A2E`、`#FF4D4D`、`#36B5FF`、`#FFC93C`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--cy-bg` | `#FFFCF3` |
| 亮色 | `--cy-surface` | `#FFFFFF` |
| 亮色 | `--cy-surface2` | `#FFF0D6` |
| 亮色 | `--cy-ink` | `#1F1A2E` |
| 亮色 | `--cy-mut` | `#6E6878` |
| 亮色 | `--cy-outline` | `#1F1A2E` |
| 亮色 | `--cy-accent` | `#FF4D4D` |
| 亮色 | `--cy-accent2` | `#36B5FF` |
| 亮色 | `--cy-accent3` | `#FFC93C` |
| 亮色 | `--cy-accent4` | `#4CC66B` |
| 亮色 | `--cy-dots` | `rgba(31,26,46,.06)` |
| 亮色 | `--cy-work` | `#2EA84B` |
| 亮色 | `--cy-idle` | `#B5AEBC` |
| 亮色 | `--cy-unread` | `#FF4D4D` |
| 亮色 | `--cy-error` | `#E63946` |
| 暗色 | `--cy-bg` | `#14102B` |
| 暗色 | `--cy-surface` | `#1F1A3A` |
| 暗色 | `--cy-surface2` | `#29224A` |
| 暗色 | `--cy-ink` | `#FFF7E6` |
| 暗色 | `--cy-mut` | `#A99BCB` |
| 暗色 | `--cy-outline` | `#FFF7E6` |
| 暗色 | `--cy-accent` | `#FF6B6B` |
| 暗色 | `--cy-accent2` | `#5CC8FF` |
| 暗色 | `--cy-accent3` | `#FFD45C` |
| 暗色 | `--cy-accent4` | `#5AD98A` |
| 暗色 | `--cy-dots` | `rgba(255,247,230,.06)` |
| 暗色 | `--cy-work` | `#5AD98A` |
| 暗色 | `--cy-idle` | `#5C557A` |
| 暗色 | `--cy-unread` | `#FF6B6B` |
| 暗色 | `--cy-error` | `#FF7A7A` |

## 字体与字重

展示与正文字体分别为：`Fredoka`、`Nunito`。主题 CSS 实际声明的字体栈包括 `'Nunito',sans-serif`、`'Fredoka',sans-serif`、`'Fredoka'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `600`、`700`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。蜡笔涂绘在组件形状上使用圆角 `18px`、`9px`、`12px`、`14px`、`11px 11px 0 0`，阴影或描边处理为 `9px 9px 0 var(--cy-outline)`、`2px 2px 0 var(--cy-outline)`、`3px 3px 0 var(--cy-outline)`、`3px 3px 0 var(--cy-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--cy-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#2EA84B`、空闲 `#B5AEBC`、未读 `#FF4D4D`、异常 `#E63946`；暗色：工作中 `#5AD98A`、空闲 `#5C557A`、未读 `#FF6B6B`、异常 `#FF7A7A`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 33 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；蜡笔涂绘以本页令牌、标题字形和背景处理形成独立辨识度。
