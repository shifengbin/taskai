# 26 Gummy — 软糖气泡

## 主题定位

软糖泡泡。该方向围绕“Gummy Bubble Pop”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。

## 视觉关键词

软糖气泡波普：圆润的胶囊描边、葡萄紫与蜜桃粉撞色、半色网点——把任务管理做成一罐软糖。

## 色彩令牌

主色与强调色：`#F5F0FB`、`#2A1B3D`、`#9B5DE5`、`#FF9F68`、`#36D39E`。页面同时提供浅色与深色样式，具体令牌如下。

| 模式 | 变量 | 值 |
| --- | --- | --- |
| 亮色 | `--gu-bg` | `#F5F0FB` |
| 亮色 | `--gu-surface` | `#FFFFFF` |
| 亮色 | `--gu-surface2` | `#ECE2F7` |
| 亮色 | `--gu-ink` | `#2A1B3D` |
| 亮色 | `--gu-mut` | `#6E5E86` |
| 亮色 | `--gu-outline` | `#2A1B3D` |
| 亮色 | `--gu-accent` | `#9B5DE5` |
| 亮色 | `--gu-accent2` | `#FF9F68` |
| 亮色 | `--gu-accent3` | `#36D39E` |
| 亮色 | `--gu-accent4` | `#48BFE3` |
| 亮色 | `--gu-dots` | `rgba(42,27,61,.07)` |
| 亮色 | `--gu-work` | `#1FA377` |
| 亮色 | `--gu-idle` | `#B6AACB` |
| 亮色 | `--gu-unread` | `#9B5DE5` |
| 亮色 | `--gu-error` | `#FF6B8B` |
| 暗色 | `--gu-bg` | `#1B1426` |
| 暗色 | `--gu-surface` | `#271E36` |
| 暗色 | `--gu-surface2` | `#302643` |
| 暗色 | `--gu-ink` | `#F3EDFB` |
| 暗色 | `--gu-mut` | `#9C8FB3` |
| 暗色 | `--gu-outline` | `#F3EDFB` |
| 暗色 | `--gu-accent` | `#B47CFF` |
| 暗色 | `--gu-accent2` | `#FFB58A` |
| 暗色 | `--gu-accent3` | `#54E6B0` |
| 暗色 | `--gu-accent4` | `#6FCCF0` |
| 暗色 | `--gu-dots` | `rgba(243,237,251,.06)` |
| 暗色 | `--gu-work` | `#54E6B0` |
| 暗色 | `--gu-idle` | `#5C4F73` |
| 暗色 | `--gu-unread` | `#B47CFF` |
| 暗色 | `--gu-error` | `#FF7E9A` |

## 字体与字重

展示与正文字体分别为：`M+ Rounded`、`Nunito`。主题 CSS 实际声明的字体栈包括 `'Nunito',sans-serif`、`'M PLUS Rounded 1c',sans-serif`、`'M PLUS Rounded 1c'`、`'JetBrains Mono',monospace`；本主题覆盖的字重为 `800`、`700`、`600`。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。

## 布局与间距

预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。

## 主要组件规则

应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。软糖泡泡在组件形状上使用圆角 `22px`、`50%`、`999px`、`16px`、`14px`，阴影或描边处理为 `9px 9px 0 var(--gu-outline)`、`2px 2px 0 var(--gu-outline)`、`3px 3px 0 var(--gu-outline)`、`3px 3px 0 var(--gu-outline), inset 6px 0 0 var(--c)`、`4px 4px 0 var(--gu-outline), inset 6px 0 0 var(--c)`。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。

## 状态表现

亮色：工作中 `#1FA377`、空闲 `#B6AACB`、未读 `#9B5DE5`、异常 `#FF6B8B`；暗色：工作中 `#54E6B0`、空闲 `#5C4F73`、未读 `#B47CFF`、异常 `#FF7E9A`。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。

## 交互反馈

任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 26 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。

## 家族特性

Pop family 共享高饱和主色、粗体展示字和直接的状态反馈；软糖泡泡以本页令牌、标题字形和背景处理形成独立辨识度。
