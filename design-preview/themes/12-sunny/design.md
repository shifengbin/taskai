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

## 字体与文字

主要字体：`Lexend`、`Plus Jakarta`。标题负责建立层级，标签与元数据保持紧凑，以便在任务清单、侧栏和状态区中稳定扫描。

## 布局与组件

预览使用最小高度 `880px` 的舞台，应用窗口宽度为 `632px`。顶栏高度为 `52px`，侧栏宽度为 `300px`；任务列表、筛选项、状态徽标与操作按钮复用同一密度规则。

## 状态与交互

每个页面同时呈现浅色与深色应用窗口。主题切换会更新窗口标题和状态区内容；清单中的选择、复选和标签状态用于展示任务的进行、完成与提醒反馈。

## 可读性与无障碍

正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。

## React + MUI 实现映射

可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。

## 预览实现

本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 12 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。
