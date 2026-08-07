# 31 Boom — 爆炸波普

## 主题定位

爆炸漫画。该方向围绕“Explosion Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

爆炸波普：品红、电黄、钴蓝、青柠四色高能冲撞，配 Bagel Fat One 超粗圆体——视觉上的一记重拳。

## 色彩令牌

主色与强调色：`#FFF8E1`、`#141414`、`#FF2D87`、`#FFD60A`、`#2D6BFF`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--bm-bg` | `#FFF8E1` |
| 亮色 | `--bm-surface` | `#FFFFFF` |
| 亮色 | `--bm-surface2` | `#FFEFD0` |
| 亮色 | `--bm-ink` | `#141414` |
| 亮色 | `--bm-mut` | `#6B6B6B` |
| 亮色 | `--bm-outline` | `#141414` |
| 亮色 | `--bm-accent` | `#FF2D87` |
| 亮色 | `--bm-accent2` | `#FFD60A` |
| 亮色 | `--bm-accent3` | `#2D6BFF` |
| 亮色 | `--bm-accent4` | `#06D6A0` |
| 亮色 | `--bm-dots` | `rgba(20,20,20,.06)` |
| 亮色 | `--bm-work` | `#0E9F6E` |
| 亮色 | `--bm-idle` | `#B0B0B0` |
| 亮色 | `--bm-unread` | `#FF2D87` |
| 亮色 | `--bm-error` | `#FF3B3B` |
| 暗色 | `--bm-bg` | `#160C2E` |
| 暗色 | `--bm-surface` | `#22143F` |
| 暗色 | `--bm-surface2` | `#2C1A4D` |
| 暗色 | `--bm-ink` | `#FFE9F4` |
| 暗色 | `--bm-mut` | `#B59BCB` |
| 暗色 | `--bm-outline` | `#FFE9F4` |
| 暗色 | `--bm-accent` | `#FF5CAB` |
| 暗色 | `--bm-accent2` | `#FFE03A` |
| 暗色 | `--bm-accent3` | `#5C8CFF` |
| 暗色 | `--bm-accent4` | `#2EE8B0` |
| 暗色 | `--bm-dots` | `rgba(255,233,244,.06)` |
| 暗色 | `--bm-work` | `#2EE8B0` |
| 暗色 | `--bm-idle` | `#6E5C95` |
| 暗色 | `--bm-unread` | `#FF5CAB` |
| 暗色 | `--bm-error` | `#FF6B7A` |

## 字体与字重

展示与正文字体分别为：`Boom`、`Baloo`。主题 CSS 实际声明的字体栈包括 `'Baloo 2',sans-serif`、`'Bagel Fat One',sans-serif`、`'Bagel Fat One'`、`'Baloo 2'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `700`、`400`、`600`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。爆炸漫画在组件形状上使用圆角 `16px`、`8px`、`10px`、`14px`、`11px`，阴影或描边处理为 `9px 9px 0 var(--bm-outline)`、`2px 2px 0 var(--bm-outline)`、`3px 3px 0 var(--bm-outline)`、`3px 3px 0 var(--bm-outline), inset 5px 0 0 var(--c)`、`4px 4px 0 var(--bm-outline), inset 5px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#0E9F6E`、空闲 `#B0B0B0`、未读 `#FF2D87`、异常 `#FF3B3B`；暗色：工作中 `#2EE8B0`、空闲 `#6E5C95`、未读 `#FF5CAB`、异常 `#FF6B7A`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 31 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；爆炸漫画以本页令牌、标题字形和背景处理形成独立辨识度。
