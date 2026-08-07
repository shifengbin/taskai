# 29 Snap — 快门波普

## 主题定位

清爽编辑。该方向围绕“Clean Editorial Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

快门波普：明快干净的粗描边 + 硬投影，珊瑚与钴蓝撞色、紧凑几何字——像杂志封面的清爽版面。

## 色彩令牌

主色与强调色：`#F1F5F4`、`#10212B`、`#FF5A4E`、`#1E66F5`、`#F5B700`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--sn-bg` | `#F1F5F4` |
| 亮色 | `--sn-surface` | `#FFFFFF` |
| 亮色 | `--sn-surface2` | `#E3EAE9` |
| 亮色 | `--sn-ink` | `#10212B` |
| 亮色 | `--sn-mut` | `#5A6E78` |
| 亮色 | `--sn-outline` | `#10212B` |
| 亮色 | `--sn-accent` | `#FF5A4E` |
| 亮色 | `--sn-accent2` | `#1E66F5` |
| 亮色 | `--sn-accent3` | `#F5B700` |
| 亮色 | `--sn-accent4` | `#8B5CF6` |
| 亮色 | `--sn-dots` | `rgba(16,33,43,.06)` |
| 亮色 | `--sn-work` | `#16A34A` |
| 亮色 | `--sn-idle` | `#9CB0B6` |
| 亮色 | `--sn-unread` | `#8B5CF6` |
| 亮色 | `--sn-error` | `#FF5A4E` |
| 暗色 | `--sn-bg` | `#0B151A` |
| 暗色 | `--sn-surface` | `#16242B` |
| 暗色 | `--sn-surface2` | `#1F3038` |
| 暗色 | `--sn-ink` | `#E6EEF1` |
| 暗色 | `--sn-mut` | `#8AA0A8` |
| 暗色 | `--sn-outline` | `#E6EEF1` |
| 暗色 | `--sn-accent` | `#FF7A6E` |
| 暗色 | `--sn-accent2` | `#5C8CFF` |
| 暗色 | `--sn-accent3` | `#FFCD33` |
| 暗色 | `--sn-accent4` | `#A98CFF` |
| 暗色 | `--sn-dots` | `rgba(230,238,241,.06)` |
| 暗色 | `--sn-work` | `#34D36B` |
| 暗色 | `--sn-idle` | `#4E6068` |
| 暗色 | `--sn-unread` | `#A98CFF` |
| 暗色 | `--sn-error` | `#FF7A6E` |

## 字体与字重

展示与正文字体分别为：`Hanken`、`Jakarta`。主题 CSS 实际声明的字体栈包括 `'Plus Jakarta Sans',sans-serif`、`'Hanken Grotesk',sans-serif`、`'Hanken Grotesk'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `800`、`700`、`500`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。清爽编辑在组件形状上使用圆角 `6px`、`7px`、`9px`、`8px 8px 0 0`，阴影或描边处理为 `8px 8px 0 var(--sn-outline)`、`2px 2px 0 var(--sn-outline)`、`3px 3px 0 var(--sn-outline)`、`3px 3px 0 var(--sn-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--sn-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#16A34A`、空闲 `#9CB0B6`、未读 `#8B5CF6`、异常 `#FF5A4E`；暗色：工作中 `#34D36B`、空闲 `#4E6068`、未读 `#A98CFF`、异常 `#FF7A6E`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 29 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；清爽编辑以本页令牌、标题字形和背景处理形成独立辨识度。
