# 35 Juice — 果味波普

## 主题定位

热带果汁。该方向围绕“Tropical Fruit Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

果味波普：橙、粉、青柠、紫的水果撞色，Modak 超重圆体配 Quicksand——多汁、圆润、最饱满的大色块。

## 色彩令牌

主色与强调色：`#FFF4E6`、`#2A1133`、`#FF6B35`、`#F72585`、`#80E000`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--jc-bg` | `#FFF4E6` |
| 亮色 | `--jc-surface` | `#FFFFFF` |
| 亮色 | `--jc-surface2` | `#FFE2CC` |
| 亮色 | `--jc-ink` | `#2A1133` |
| 亮色 | `--jc-mut` | `#6E5A78` |
| 亮色 | `--jc-outline` | `#2A1133` |
| 亮色 | `--jc-accent` | `#FF6B35` |
| 亮色 | `--jc-accent2` | `#F72585` |
| 亮色 | `--jc-accent3` | `#80E000` |
| 亮色 | `--jc-accent4` | `#7209B7` |
| 亮色 | `--jc-dots` | `rgba(42,17,51,.06)` |
| 亮色 | `--jc-work` | `#3DAA14` |
| 亮色 | `--jc-idle` | `#B6A0C8` |
| 亮色 | `--jc-unread` | `#F72585` |
| 亮色 | `--jc-error` | `#E5383B` |
| 暗色 | `--jc-bg` | `#190F24` |
| 暗色 | `--jc-surface` | `#241533` |
| 暗色 | `--jc-surface2` | `#2E1C40` |
| 暗色 | `--jc-ink` | `#FFEBE0` |
| 暗色 | `--jc-mut` | `#B59CC0` |
| 暗色 | `--jc-outline` | `#FFEBE0` |
| 暗色 | `--jc-accent` | `#FF8A5C` |
| 暗色 | `--jc-accent2` | `#FF5CA8` |
| 暗色 | `--jc-accent3` | `#9EE840` |
| 暗色 | `--jc-accent4` | `#B55CDB` |
| 暗色 | `--jc-dots` | `rgba(255,235,224,.06)` |
| 暗色 | `--jc-work` | `#9EE840` |
| 暗色 | `--jc-idle` | `#5C4C6E` |
| 暗色 | `--jc-unread` | `#FF5CA8` |
| 暗色 | `--jc-error` | `#FF6B6B` |

## 字体与字重

展示与正文字体分别为：`Modak`、`Quicksand`。主题 CSS 实际声明的字体栈包括 `'Quicksand',sans-serif`、`'Modak',sans-serif`、`'Modak'`、`'Quicksand'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `400`、`700`、`600`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。热带果汁在组件形状上使用圆角 `20px`、`10px`、`14px`、`16px`、`13px 13px 0 0`，阴影或描边处理为 `9px 9px 0 var(--jc-outline)`、`2px 2px 0 var(--jc-outline)`、`3px 3px 0 var(--jc-outline)`、`3px 3px 0 var(--jc-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--jc-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#3DAA14`、空闲 `#B6A0C8`、未读 `#F72585`、异常 `#E5383B`；暗色：工作中 `#9EE840`、空闲 `#5C4C6E`、未读 `#FF5CA8`、异常 `#FF6B6B`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 35 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；热带果汁以本页令牌、标题字形和背景处理形成独立辨识度。
