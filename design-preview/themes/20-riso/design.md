# 20 Riso — 孔版印刷

## 主题定位

孔版印刷。该方向围绕“Risograph Print”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

粗框硬投影的另一种气质：靛蓝与珊瑚橙的"错位叠印"双投影，模拟孔版印刷的套色偏差——柔纸底色配高饱和填充标签，复古鲜活又不失清晰。

## 色彩令牌

主色与强调色：`#1B3A8B`、`#FF5A5F`、`#F4EFE6`、`#2E6BD0`、`#1F9D55`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--ri-bg` | `#F4EFE6` |
| 亮色 | `--ri-surface` | `#FBF7F0` |
| 亮色 | `--ri-surface2` | `#E7DCCB` |
| 亮色 | `--ri-ink` | `#1B3A8B` |
| 亮色 | `--ri-mut` | `#5E6E8C` |
| 亮色 | `--ri-line` | `rgba(27,58,139,.16)` |
| 亮色 | `--ri-frame` | `#1B3A8B` |
| 亮色 | `--ri-accent` | `#FF5A5F` |
| 亮色 | `--ri-accent2` | `#2E6BD0` |
| 亮色 | `--ri-accent3` | `#FF8A6B` |
| 亮色 | `--ri-work` | `#1F9D55` |
| 亮色 | `--ri-idle` | `#B7AE9B` |
| 亮色 | `--ri-unread` | `#2E6BD0` |
| 亮色 | `--ri-error` | `#FF5A5F` |
| 暗色 | `--ri-bg` | `#10182E` |
| 暗色 | `--ri-surface` | `#18223F` |
| 暗色 | `--ri-surface2` | `#202D4E` |
| 暗色 | `--ri-ink` | `#E7EEFF` |
| 暗色 | `--ri-mut` | `#94A2C2` |
| 暗色 | `--ri-line` | `rgba(231,238,255,.16)` |
| 暗色 | `--ri-frame` | `#E7EEFF` |
| 暗色 | `--ri-accent` | `#FF7A7F` |
| 暗色 | `--ri-accent2` | `#5C8CF0` |
| 暗色 | `--ri-accent3` | `#FF9E84` |
| 暗色 | `--ri-work` | `#39D97E` |
| 暗色 | `--ri-idle` | `#4A5675` |
| 暗色 | `--ri-unread` | `#5C8CF0` |
| 暗色 | `--ri-error` | `#FF7A7F` |

## 字体与字重

展示与正文字体分别为：`Fraunces`、`Work Sans`。主题 CSS 实际声明的字体栈包括 `'Work Sans',sans-serif`、`'Fraunces',serif`、`'Work Sans'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `600`、`700`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。孔版印刷在组件形状上使用圆角 `13px`、`8px`、`9px`、`10px`、`7px`，阴影或描边处理为 `5px 5px 0 var(--ri-accent), 9px 9px 0 var(--ri-frame)`、`2.5px 2.5px 0 var(--ri-frame)`、`3.5px 3.5px 0 var(--ri-frame)`、`inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#1F9D55`、空闲 `#B7AE9B`、未读 `#2E6BD0`、异常 `#FF5A5F`；暗色：工作中 `#39D97E`、空闲 `#4A5675`、未读 `#5C8CF0`、异常 `#FF7A7F`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 20 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Hard-shadow family 以硬边框、明确的层叠阴影和高对比状态块建立操作优先级；孔版印刷通过自己的主色、字形和装饰节奏维持可区分性。
