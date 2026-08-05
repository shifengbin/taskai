## 1. 基础设施与令牌

- [x] 1.1 确认工作区未提交改动来源，确保本变更只触及前端且不覆盖既有业务工作。
- [x] 1.2 接入 Tailwind **v3**（项目固定 Vite 3，不兼容 v4 的 `@tailwindcss/vite`，改用 postcss + `tailwind.config.cjs`）+ Radix(slot) + `lucide-react` + `class-variance-authority`/`clsx`/`tailwind-merge`；MUI 暂留，`npm run build` 通过。
- [x] 1.3 配置 `tailwind.config.cjs`：快门波普令牌（color/fontFamily/borderRadius/shadow 引用 `var(--snap-*)`）、暗色 class 策略、内容路径。
- [x] 1.4 建立令牌 CSS：`:root`/`.dark` 的画布/表面/正文/描边/珊瑚/钴蓝/琥珀/紫罗兰/网点变量；根容器在 `colorScheme==='dark'` 时附加 `dark` class。
- [x] 1.5 自托管 Hanken Grotesk、Plus Jakarta Sans、JetBrains Mono 的 woff2（仅实际字重），修正 Nunito `@font-face` 字重声明 bug，中文走系统 Noto Sans SC 回退。
- [x] 1.6 为亮/暗的正文、辅助、钴蓝按钮、珊瑚填充、状态标签、表单标签计算并记录对比度（正文 ≥4.5:1，大号粗体 ≥3:1）。

## 2. 快门波普组件原语库

- [x] 2.1 在 `src/components/ui/` 移植 Snap 基础原语（参照 `design-preview`）：Button（cva variants）、IconButton、Chip、Alert、Divider、ScrollArea。
- [x] 2.2 移植 Radix 交互原语并套 Snap 皮肤：Dialog、Tabs、Accordion、Tooltip、Dropdown Menu、Popover、Switch、Checkbox、ScrollArea、Toast。（**Select/Slider 未使用，按 YAGNI 暂不建**——映射表为推测，实际代码无消费者；将来若引入再补。）
- [x] 2.3 为所有原语统一 2–2.5px 描边、`Npx Npx 0` 硬投影、6–9px 圆角、hover 抬起，并内置 `prefers-reduced-motion` 静态等价与钴蓝焦点环。
- [x] 2.4 重绘 `assets/task-ai-mark.svg` 为快门波普版（实色珊瑚方块 + 墨色粗描边 + 白色光圈/快门），保留引用方式。

## 3. 主题入口与外壳

- [x] 3.1 用 Tailwind 令牌 + dark class 替换 `createAppTheme` 与 `ThemeProvider`，删除 MUI palette/typography/组件覆盖。（过渡期保留薄桥接主题：镜像快门波普令牌到 MUI palette/字体、删除全部组件 styleOverrides 与衬线排版；Phase 7 随 MUI 一并移除 ThemeProvider。）
- [x] 3.2 移植应用外壳：顶栏、品牌区（mark + 标题）、工具按钮（lucide Folder/Settings/Logout）、可拖拽分隔条、启动屏、详情空状态。

## 4. 任务树与导航

- [x] 4.1 移植 `TaskTree`：状态标签页（未执行/执行中/已完成 Tabs）、任务行卡片、终端子项、选中/悬停/焦点/禁用态。（两处偏离已记录：①任务操作菜单采用**自绘定位 `role=menu`** 而非 Radix DropdownMenu——因为同一菜单需同时支持「按钮锚点」和「右键坐标」两种定位，Radix 依赖 Trigger 定位无法覆盖右键坐标场景；②任务描述气泡采用**mouseenter/focus 驱动的自绘 `role=tooltip`** 而非 Radix Tooltip——测试环境无 `matchMedia` polyfill，Radix 的 `(pointer: fine)` 判定不可靠会致 hover 不开。动作按钮的悬停提示暂用原生 `title`，待 6.4 统一。）
- [x] 4.2 任务自定义颜色继续作为左边框与约 4% 不透明极浅背景，不被主题色覆盖。
- [x] 4.3 拖拽预览与插入指示器改为快门波普反馈，保留现有指针阈值、放置判定与排序回调。
- [x] 4.4 移植 `TerminalStatusDot` 的空闲/工作中/未读/错误颜色与脉冲（按开放问题映射），保持角色、标签与减少动效行为。

## 5. 终端与任务详情

- [x] 5.1 `TerminalView` 从令牌读色注入 xterm `ITheme`（背景+网点、前景、光标、关键字紫罗兰、提示符珊瑚、成功钴蓝），叠加半色网点底纹。
- [x] 5.2 移植终端标签头、终端 meta、关闭操作；不改输入输出、尺寸、焦点、剪贴板与生命周期。
- [x] 5.3 移植任务详情容器、标题、字段卡、详情值左边框，保留既有布局与事件。（详情容器/标题/字段卡/额外信息/环境变量/工作目录块与 `TaskDetailValue`（加钴蓝左边框 `border-l-2 border-snap-cobalt/40`）已全部换为 Snap `<div>/<section>/<span>`；状态 `Chip`、空态/生命周期 `Alert`、复制命令链 `Button` 及 `Alert` 内的 `Typography` 暂留 MUI，待 6.4 统一。）

## 6. 覆盖层与表单

- [x] 6.1 移植任务编辑（新建/编辑）对话框，长表单用 Radix ScrollArea 在弹窗内滚动，取消/保存常驻。
- [x] 6.2 移植额外信息管理/编辑、模板编辑、菜单项编辑对话框。
- [x] 6.3 移植设置、状态帮助、结束确认、退出确认对话框。
- [x] 6.4 统一 Menu、Popover、Snackbar/Toast、Alert、Accordion、Chip、TextField、Checkbox、Switch 的快门波普外观与可见焦点，不改打开/关闭/提交/校验/删除逻辑。（App.tsx 已无任何 MUI 组件用法，仅剩 ThemeProvider/CssBaseline/createAppTheme 桥接与 13 个未使用的 `@mui/icons-material` 导入，留待 7.1 移除。测试契约随迁移：①jsdom 不展开 `borderTop` 简写，断言元素改用 `borderTopStyle/Width/Color` 长写内联；②禁用删除按钮的提示从 MUI Tooltip(`role=tooltip`) 改为原生 `title`，测试断言 `toHaveAttribute('title', ...)`；③原生 `<select>` 用 `user.selectOptions` 取代「点开 + 点 option」，多 select 共存时用 `within(select)` 收窄 option 查询；④「分类模板」折叠区改 `forceMount` 以匹配原「可滚动纵向列表始终挂载」行为，折叠态 `value` 用 `''` 而非 `undefined`（Radix single 受控）。）

## 7. 清理 MUI、测试与交付验证

- [x] 7.1 移除 `@mui/material`、`@mui/icons-material`、`@emotion/react`、`@emotion/styled`；删除 `createAppTheme` 与 `App.css`/`style.css` 中全部 MUI 覆盖与残留固定色。（已完成：App.tsx 删除整段 `@mui/material`+13 个 `@mui/icons-material` 导入、`createAppTheme` 函数、`theme` useMemo 与两处 `ThemeProvider`/`CssBaseline` 包装；package.json 移除 4 个依赖，`npm install` 共清理 48 个包；App.css 删除全部 `.Mui*` 覆盖与已由 Tailwind 接管的死类（topbar/task-tree/terminal-row/detail-empty/startup 等），把仍需提供容器底色的 `taskai-app/sidebar-shell/sidebar-header/content-pane/task-row/detail-card/terminal` 从庭院 `--taskai-*` 迁到快门波普 `--snap-*` 令牌，删除庭院调色板与固定色 `#edf0ea`；style.css 本就无 MUI。`npm run build` 通过（JS 包 903.96→813.90 KiB），`npx vitest run` 174/175 通过——唯一失败仍是 7.2 待修的分隔条精确色断言。）
- [x] 7.2 更新前端测试：role/aria 无障碍断言复用，MUI 特有 class/结构断言改为 Radix/Tailwind 等价；补亮暗主题入口、任务树、终端状态与关键交互的回归测试。（已完成：①修复延后的分隔条精确色断言——`App.test.tsx`「暗色模式…分隔条」由断言庭院精确色 `rgb(60,75,65)` 改为断言快门波普描边工具类 `toHaveClass('bg-snap-outline/25')`（jsdom 无法计算 Tailwind color-mix 透明度，故不再断言计算色），并改名为「暗色模式分隔条使用快门波普描边色」；②全量 grep 确认测试文件已无任何 `.Mui*` 断言（Phase 4–6 已就地迁移：`TaskTree.test.tsx` 图标 testid→`menuitem svg`、`.MuiChip-root`→`.taskai-lifecycle-chip`、选中态精确底色→`taskai-task-row--selected` 类）；③新增 2 个主题入口回归测试——亮色根容器 `data-color-scheme=light` 且无 `dark` 类、暗色 `data-color-scheme=dark` 且有 `dark` 类（激活 style.css 暗色令牌）。`npx vitest run` 177/177 全绿。）
- [x] 7.3 离线逐屏检查亮色、暗色、任务详情、空状态、活动终端、拖拽、错误提示与每类弹窗的视觉层级、可读性与减少动效。（Wails 桌面应用离线无法跑起 Go 后端渲染真实 GUI，改为代码级逐屏审计：①全量 grep 确认所有 `.tsx`（非测试）零硬编码色——颜色一律来自 `--snap-*` 令牌或 Tailwind 工具类，亮/暗两套令牌由 `:root`/`.dark` 驱动；②焦点环统一——`focusRing` 为唯一钴蓝 3px `focus-visible:ring`，鼠标点击不显示；③减少动效——全局 `@media (prefers-reduced-motion)`（App.css/style.css）压制所有过渡/动画时长，三处 hover 抬起（Button/IconButton/Dialog 关闭钮）逐一配 `motion-reduce:translate-x/y-0` 兜底（本次补齐 Dialog 关闭钮缺失的兜底）；④Alert 用 `role=alert`、2px 描边、severity 着色图标（info/success=钴蓝、warning=琥珀、error=红，与锁定决策一致）；⑤终端 xterm 主题镜像快门波普调色板（亮色底 `#E3EAE9`+网点、暗色底 `#16242B`，前景墨色/光标钴蓝——ITheme 需显式串故用令牌原值）；⑥空状态（启动屏、任务详情未选）均 snap 令牌居中 + muted 文案；⑦main.tsx 自托管 Hanken/Plus Jakarta/JetBrains Mono 字体 + style.css，无 MUI。`npm run build` 与 177/177 测试通过。）
- [x] 7.4 运行 `cd frontend && npm test && npm run build`、`openspec validate --strict` 与项目构建脚本，记录结果并仅修复本变更引入的问题。（全绿：①`npx vitest run` 177/177 通过（9 个测试文件）；②`npm run build` 通过，前端 JS 包 813.90 KiB（较 Phase 6 的 903.96 KiB 减约 90 KiB）、CSS 45.16 KiB，仅遗留既有的 chunk 体积告警与 Radix `use client` 提示；③`openspec validate --changes --strict` 1 passed / 0 failed；④`go build ./...` 通过（本变更纯前端，Go 未受影响）；⑤`wails build -platform windows/amd64` 成功产出 `build/bin/taskai.exe`（1m9s，含绑定生成→前端依赖安装→前端编译→资产打包→Go/CGO 编译全链路）。本变更未引入任何需修复的失败。）
