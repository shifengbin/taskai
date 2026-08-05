## 背景

TaskAI 是基于 React 18、MUI v9、Emotion 与 xterm 的 Wails 桌面任务工作台。主题入口为 `App.tsx` 的 `createAppTheme(colorScheme)`，输出 MUI palette/shape/typography 与逐组件覆盖；`App.css` 以 CSS 变量补充；图标来自 `@mui/icons-material/*Outlined`；终端用 xterm 独立主题 API。视觉血脉已两轮（松林夜跑 → 庭院鼠尾草），本次第三轮（庭院 → 快门波普 Snap）。

用户要求**直接替换 UI 库**。核心理由：MUI 的 Material 语言与新粗野结构性对抗，皮肤层会变成无止境的 `!important` 覆盖；改用无样式原语后，快门波普成为组件的天然形态。`design-preview/index.html` 第 29 方案 `d-snap` 已是纯 HTML/CSS 写好的整套 Snap 组件源码——它是移植目标，不是参考图。

## 目标 / 非目标

**目标：**

- 把组件基础由 MUI 迁移到 Radix 无样式原语 + Tailwind + lucide，并完全移除 MUI/Emotion。
- 以 Tailwind 配置 + CSS 变量承载快门波普亮/暗令牌，替代 `createAppTheme`。
- 移植 `design-preview` 的 Snap 组件为自有 React 组件库，原生体现粗描边 + 硬投影 + 抬起。
- 重写应用外壳、任务树、详情、终端与全部弹窗，保持信息架构与业务逻辑。
- 满足可访问性（4.5:1、可见焦点、键盘）与 `prefers-reduced-motion`。

**非目标：**

- 不改任务、终端、生命周期、额外信息、设置与退出的业务规则、数据模型或 API。
- 不新增设置项、服务端接口、第三套主题模式或主题切换器。
- 不重排操作路径、不变更控件语义、不删功能。
- 不把 `design-preview/index.html` 耦合进应用。

## 决策

### 1. 用 shadcn 模式（Radix + Tailwind + lucide）取代 MUI

Radix 提供无样式、可访问的原语（焦点陷阱、键盘、ARIA、Portal 都内置），样式 100% 归我们；Tailwind 让"描边 + 硬投影 + 抬起"成为原子化的一行；`design-preview` 已用同样的几何写出 Snap，移植路径直接。lucide 与 `design-preview` 的图标同源。随后完全移除 `@mui/*` 与 `@emotion/*`。

备选 Mantine——不采用，它仍带自身设计主张，新粗野每处仍需覆写，没摆脱"库替你决定外观"。备选纯手写——不采用，焦点陷阱/键盘/ARIA 自证成本过高，Radix 已免费提供。

### 2. 组件映射 MUI → Radix / Tailwind

| MUI 组件 | Snap 替代 |
|---|---|
| Button / IconButton | 自有 Button（cva variants）+ lucide 图标 |
| Dialog / DialogTitle / DialogActions | Radix Dialog |
| Tabs / Tab | Radix Tabs |
| Accordion / AccordionSummary | Radix Accordion |
| Tooltip | Radix Tooltip |
| Menu / MenuItem | Radix Dropdown Menu / Menu |
| Popover | Radix Popover |
| Switch | Radix Switch |
| Checkbox | Radix Checkbox |
| Select | Radix Select |
| Slider | Radix Slider |
| TextField / OutlinedInput | 原生 input + Tailwind |
| Chip | 自有 span + cva |
| Alert | 自有 Alert |
| Snackbar | 自有 Toast（或 sonner） |
| Box / Stack / Paper / AppBar | div/header + Tailwind flex·grid |
| Divider | border |
| ScrollArea（长表单） | Radix ScrollArea |

### 3. 令牌架构：Tailwind 配置 + CSS 变量，替代 createAppTheme

亮色令牌：画布 `#F1F5F4`、主表面 `#FFFFFF`、次表面 `#E3EAE9`、正文 `#10212B`、辅助 `#5A6E78`、描边 `#10212B`、珊瑚 `#FF5A4E`、钴蓝 `#1E66F5`、琥珀 `#F5B700`、紫罗兰 `#8B5CF6`、网点 `rgba(16,33,43,.06)`。暗色：画布 `#0B151A`、主表面 `#16242B`、次表面 `#1F3038`、正文 `#E6EEF1`、辅助 `#8AA0A8`、描边 `#E6EEF1`、珊瑚 `#FF7A6E`、钴蓝 `#5C8CFF`、琥珀 `#FFCD33`、紫罗兰 `#A98CFF`、网点 `rgba(230,238,241,.06)`。

`:root` 与 `.dark` 定义这些 CSS 变量；`tailwind.config` 的 `theme.extend.colors`/`fontFamily`/`borderRadius` 引用 `var(--snap-*)`。现有 `colorScheme` 切换映射为根节点的 `dark` class（Tailwind class 策略）。

### 4. primary 用钴蓝、品牌/强调用珊瑚——由对比度决定

珊瑚 `#FF5A4E` 配白字约 3.6:1，不满足普通正文 4.5:1，只适合大号/粗体或纯填充；钴蓝 `#1E66F5` 配白字约 5.0:1，达标。故 primary（按钮、焦点环、选中态、详情值左边框）= 钴蓝；品牌、标题带、品牌标记、终端活动标签等大面积填充或大号粗体 = 珊瑚，配白字时字重 ≥ 700 且字号足够，或改墨色字。琥珀=警告、紫罗兰=信息/终端关键字、错误用比珊瑚更深的可辨识红（如 `#E0341B`）。

### 5. 图标换 lucide，与 design-preview 同源

13 处 `@mui/icons-material/*Outlined` 换成 `lucide-react`（Add→Plus、Settings→Settings、Delete→Trash2、ExpandMore→ChevronDown、Folder→Folder、Logout→LogOut、Help→HelpCircle、TaskAlt→CheckCircle、DoneAll→ListChecks、ArrowUp/Down、UnfoldMore/Less→Maximize2/Minimize2），统一 `strokeWidth={2.25}`。随组件迁移完成即移除 `@mui/icons-material`。

### 6. 字体三套离线自托管，CJK 系统回退

Hanken Grotesk（标题/品牌）、Plus Jakarta Sans（正文/UI）、JetBrains Mono（终端）的 woff2 自托管到 `assets/fonts/`，仅打包实际字重。subtitle1/h5 不再用宋体，统一 Hanken Grotesk。中文走系统 Noto Sans SC。修正既有 Nunito `@font-face` 声明 `400 900` 却只 bundle regular 的潜伏 bug。

### 7. 终端按快门波普配色，行为不变

`TerminalView` 从令牌读色注入 xterm `ITheme`：前景墨色、光标/关键字紫罗兰、提示符珊瑚、成功钴蓝。**亮色模式终端背景用浅次表面 `#E3EAE9` + 半色网点；暗色模式用更深的夜色次表面 `#1F3038` + 半色网点。** 若亮色模式下命令输出可读性不足，亮色终端回退为深色表面（仅改 xterm 背景色，不动其它）。只更新 xterm options，终端实例、输入输出、尺寸、剪贴板、焦点、生命周期不变。

### 8. 迁移策略：搭脚手架 → 建原语 → 垂直切片移植 → 拆 MUI

1. 在保留 MUI 的前提下接入 Tailwind + Radix + 令牌 CSS + 字体，确认构建通过。
2. 在 `src/components/ui/` 移植 Snap 组件原语（参照 `design-preview`）。
3. 按外壳 → 任务树 → 详情/终端 → 覆盖层的顺序，逐屏用 Snap 组件替换 MUI（期间 MUI 与 Radix 可临时共存）。
4. 全部迁移完成后移除 MUI/Emotion、`createAppTheme` 与 App.css 的 MUI 覆盖。

不做大爆炸式一次性重写，以垂直切片保证每屏可独立验收与回退。

### 9. 可读性作为验收条件

正文/辅助/按钮/状态/表单文字与背景对比度 ≥ 4.5:1（大号粗体 ≥ 3:1）；钴蓝达标，珊瑚填充上的文字确保粗体大号或墨色。焦点环（钴蓝 3px）在亮暗与任务自定义色之上可见。危险操作用语义错误色 + 文字标签。

**令牌对比度实测（WCAG 相对亮度）：**

| 文字/背景 | 亮色 | 暗色 | 结论 |
|---|---|---|---|
| 正文 ink/surface | 16.7:1 | 12.9:1 | ✓ 达标 |
| 辅助 muted/surface | 5.35:1 | 5.65:1 | ✓ 达标 |
| 钴蓝按钮 白字/cobalt | 4.91:1 | 3.16:1 | 亮色✓；暗色需粗体大号（≥3:1） |
| 珊瑚填充 白字/coral | 3.08:1 | 2.54:1 | 仅大号粗体；改用墨字 5.40:1（亮） |
| 错误 白字/error | 4.49:1 | — | 临界，用粗体或墨字 |
| 焦点环 cobalt/surface | 高 | 高 | ✓ 可见 |

结论：正文、辅助、钴蓝按钮（亮色）达标；**珊瑚与暗色钴蓝作为填充时只承载粗体大号文字或改用墨字**，与决策 4 一致。错误色文字用粗体或配墨字以保证 ≥4.5:1。

## 风险 / 权衡

- [这是迄今最大变更，App.tsx 2900 行 + 3 组件全重写] → 用垂直切片分屏迁移，每屏可独立验收与回退；xterm 与业务逻辑零改动。
- [测试大量基于 MUI class/结构] → role/aria 无障碍断言大多复用（Radix 同样可访问）；MUI 特有 class 断言逐项更新为 Radix/Tailwind 等价；行为断言不动。
- [Tailwind + Radix 引入新工具链] → 项目固定在 Vite 3，不兼容 Tailwind v4 的 `@tailwindcss/vite`（需 Vite≥5）；改用 Tailwind **v3** + postcss + `tailwind.config.cjs`，零 Vite 升级、保全 175 测试安全网。Radix 按需 tree-shake，净包体预计小于 MUI。未来升级 Vite 后可平滑切到 v4。
- [珊瑚对比度不足] → primary 用钴蓝；珊瑚限大号粗体或填充。
- [三套字体增大包体] → 仅打包实际字重，CJK 走系统回退不打包。
- [两套主题系统临时共存] → 仅在迁移过渡期，完成后彻底移除 MUI，避免长期双轨。

## 迁移计划

1. 接入 Tailwind + Radix + lucide + 令牌 CSS + 离线字体（MUI 暂留），确认构建。
2. 移植 Snap 组件原语库。
3. 令牌入口迁移：`colorScheme` → Tailwind dark class，移除 `createAppTheme`。
4. 按外壳 → 任务树 → 详情/终端 → 覆盖层逐屏移植。
5. 移除 MUI/Emotion 与残留覆盖，更新测试。
6. 验证：前端测试、`npm run build`、`openspec validate --strict`、项目构建脚本；离线 + 减少动效 + 亮/暗逐屏验收。

## 已决事项

- 状态点配色：working=钴蓝、unread=琥珀、error=红、idle=muted。
- 终端底色：亮色模式用快门波普浅次表面 `#E3EAE9` + 半色网点；暗色模式用更深的夜色次表面；若亮色模式可读性不足，回退为深色表面。
- 长任务表单滚动：沿用既有“弹窗内滚动、操作常驻”方案，用 Radix ScrollArea 实现，行为不变。

## 开放问题

- 无。
