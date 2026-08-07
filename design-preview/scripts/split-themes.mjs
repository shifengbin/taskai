import { copyFile, mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import process from 'node:process';

const root = process.cwd();
const previewDir = join(root, 'design-preview');
const sourcePath = join(previewDir, '_source-gallery.html');
const argumentsSet = new Set(process.argv.slice(2));
const rangeArgument = [...argumentsSet].find((argument) => argument.startsWith('--range='));
const range = rangeArgument ? rangeArgument.slice('--range='.length).split('-').map(Number) : null;
const skipThemes = argumentsSet.has('--skip-themes');
const skipShared = argumentsSet.has('--skip-shared');
const skipGallery = argumentsSet.has('--skip-gallery');
const refreshDocs = argumentsSet.has('--refresh-docs');

const themeMetadata = [
  ['01', 'monolith', 'MONOLITH', '深色指挥舱'],
  ['02', 'atelier', 'ATELIER', '工作室画布'],
  ['03', 'nebula', 'NEBULA', '赛博指挥中心'],
  ['04', 'wabi', 'WABI', '日式禅意极简'],
  ['05', 'pop', 'POP', '孟菲斯印刷'],
  ['06', 'deco', 'DECO', '装饰艺术'],
  ['07', 'comic', 'COMIC', '漫画分镜'],
  ['08', 'holo', 'HOLO', '全息科技'],
  ['09', 'arcade', 'ARCADE', '八位街机'],
  ['10', 'citrus', 'CITRUS', '热带明快'],
  ['11', 'blocks', 'BLOCKS', '几何原色'],
  ['12', 'sunny', 'SUNNY', '明亮消费品'],
  ['13', 'candy', 'CANDY', '糖果泡泡'],
  ['14', 'doodle', 'DOODLE', '手绘马克笔'],
  ['15', 'y2k', 'Y2K', '千禧铬金属'],
  ['16', 'groovy', 'GROOVY', '七十年代复古'],
  ['17', 'disco', 'DISCO', '电子俱乐部'],
  ['18', 'burst', 'BURST', '彩纸庆典'],
  ['19', 'acid', 'ACID', '酸性实验室'],
  ['20', 'riso', 'RISO', '孔版印刷'],
  ['21', 'memphis', 'MEMPHIS', '八十年代孟菲斯'],
  ['22', 'sunset', 'SUNSET', '落日大道'],
  ['23', 'concrete', 'CONCRETE', '工业混凝土'],
  ['24', 'neon', 'NEON', '霓虹灯牌'],
  ['25', 'bolt', 'BOLT', '电光科技'],
  ['26', 'gummy', 'GUMMY', '软糖泡泡'],
  ['27', 'stamp', 'STAMP', '印刷海报'],
  ['28', 'punch', 'PUNCH', '冲击海报'],
  ['29', 'snap', 'SNAP', '清爽编辑'],
  ['30', 'brio', 'BRIO', '热带活力'],
  ['31', 'boom', 'BOOM', '爆炸漫画'],
  ['32', 'zap', 'ZAP', '电力波形'],
  ['33', 'crayon', 'CRAYON', '蜡笔涂绘'],
  ['34', 'riot', 'RIOT', '朋克杂志'],
  ['35', 'juice', 'JUICE', '热带果汁'],
  ['36', 'pixel', 'PIXEL', '像素糖果'],
];

function between(value, start, end) {
  return value.slice(value.indexOf(start) + start.length, value.indexOf(end, value.indexOf(start) + start.length));
}

function escapeRegExp(value) {
  return value.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
}

function extractThemeCss(style) {
  const marker = /\/\* ={3,}\n\s*(\d{2}) · ([A-Z0-9]+) — [^\n]+\n\s*={3,} \*\//g;
  const matches = [...style.matchAll(marker)];
  const sectionStart = style.search(/\/\* =+\n\s*SECTION \/ DESIGN-NOTE CARDS/);
  return new Map(matches.map((match, index) => [
    match[1],
    style.slice(match.index, index + 1 < matches.length ? matches[index + 1].index : sectionStart),
  ]));
}

function extractSections(html) {
  return new Map([...html.matchAll(/<section class="study reveal" id="s(\d+)">[\s\S]*?<\/section>/g)].map((match) => [String(match[1]).padStart(2, '0'), match[0]]));
}

function extractConfig(sourceScript, cssClass, title) {
  const escaped = escapeRegExp(cssClass);
  const branch = new RegExp("(?:if|else if)\\(c\\.contains\\('" + escaped + "'\\)\\)\\{([\\s\\S]*?)\\n  \\}(?=else if|\\s*\\}\\);)").exec(sourceScript)?.[1] ?? '';
  const brandText = /name\.textContent = '([^']+)'/.exec(branch)?.[1] ?? '';
  const brandHtml = /name\.innerHTML = '([^']+)'/.exec(branch)?.[1] ?? '';
  const windowText = /winTitle\.textContent = '([^']+)'/.exec(branch)?.[1] ?? '';
  const windowHtml = /winTitle\.innerHTML = '([^']+)'/.exec(branch)?.[1] ?? '';
  const extraClass = /sys\.className='([^']+)'/.exec(branch)?.[1] ?? '';
  const extraText = /sys\.textContent='([^']+)'/.exec(branch)?.[1] ?? '';
  return { brandText, brandHtml, windowText, windowHtml, extraClass, extraText, fallback: title };
}

function parseTheme(section, css) {
  const cssClass = /<div class="mockup (d-[^ ]+) light">/.exec(section)?.[1] ?? '';
  const title = /<h2[^>]*>([^<]+)<\/h2>/.exec(section)?.[1] ?? '';
  const subtitle = /<div class="s-sub">([^<]+)<\/div>/.exec(section)?.[1] ?? '';
  const description = /<div class="s-tag">([^<]+)<\/div>/.exec(section)?.[1] ?? '';
  const fonts = [...section.matchAll(/<span class="ff"[^>]*>([^<]+)<\/span>/g)].map((match) => match[1]);
  const palette = [...section.matchAll(/<span style="background:([^";]+)[^>]*><\/span>/g)].map((match) => match[1]);
  const vibes = [...section.matchAll(/<div class="vibes">([\s\S]*?)<\/div>/g)].flatMap((match) => [...match[1].matchAll(/<b>([^<]+)<\/b>/g)].map((vibe) => vibe[1]));
  return { cssClass, title, subtitle, description, fonts, palette, vibes, css };
}

function isolateThemeCss(css, cssClass) {
  return css.replace(/([^{}]+)\{([^{}]*)\}/g, (rule, selectors, declaration) => {
    const kept = selectors.split(',').filter((selector) => !/\.d-[a-z0-9-]+/i.test(selector) || selector.includes('.' + cssClass));
    return kept.length ? kept.join(',') + '{' + declaration + '}' : '';
  });
}

function tokenRows(themeCss, cssClass) {
  const escaped = escapeRegExp(cssClass);
  const blocks = [...themeCss.matchAll(new RegExp('\\.' + escaped + '\\.(light|dark)\\{([^}]*)\\}', 'gi'))];
  return blocks.flatMap((block) => [...block[2].matchAll(/--([a-z0-9-]+):([^;}\n]+)(?:;|(?=$))/gi)]
    .map((token) => '| ' + (block[1].toLowerCase() === 'light' ? '亮色' : '暗色') + ' | `--' + token[1] + '` | `' + token[2].trim() + '` |'))
    .join('\n');
}

function cssPropertyValues(css, property, limit = 5) {
  const values = [...css.matchAll(new RegExp(property + ':([^;}]+)', 'gi'))]
    .map((match) => match[1].trim())
    .filter((value, index, list) => value && list.indexOf(value) === index);
  return values.slice(0, limit);
}

function stateSummary(themeCss, cssClass) {
  const escaped = escapeRegExp(cssClass);
  const blocks = [...themeCss.matchAll(new RegExp('\\.' + escaped + '\\.(light|dark)\\{([^}]*)\\}', 'gi'))];
  const labels = { work: '工作中', idle: '空闲', unread: '未读', error: '异常' };
  return blocks.map((block) => {
    const values = Object.entries(labels).flatMap(([token, label]) => {
      const value = new RegExp('--[a-z0-9-]*' + token + ':([^;}\\n]+)', 'i').exec(block[2])?.[1]?.trim();
      return value ? [label + ' `' + value + '`'] : [];
    });
    return (block[1].toLowerCase() === 'light' ? '亮色' : '暗色') + '：' + values.join('、');
  }).join('；');
}

function themeDocument(meta, theme) {
  const palette = theme.palette.map((value) => '`' + value + '`').join('、') || '页面 CSS 令牌定义的主题色';
  const fonts = theme.fonts.map((value) => '`' + value + '`').join('、') || '系统字体栈';
  const vibes = theme.vibes.join('、') || '单任务、高信息密度';
  const description = theme.description && /[。！？]$/.test(theme.description) ? theme.description : theme.description + '。';
  const cssFonts = cssPropertyValues(theme.css, 'font-family').map((value) => '`' + value + '`').join('、') || '公共骨架的系统字体回退栈';
  const weights = cssPropertyValues(theme.css, 'font-weight').map((value) => '`' + value + '`').join('、') || '标题与任务名称沿用公共骨架的 `600`';
  const radii = cssPropertyValues(theme.css, 'border-radius').map((value) => '`' + value + '`').join('、') || '沿用基础组件的紧凑圆角';
  const shadows = cssPropertyValues(theme.css, 'box-shadow').map((value) => '`' + value + '`').join('、') || '不额外覆盖公共阴影';
  const states = stateSummary(theme.css, theme.cssClass);
  const number = Number(meta[0]);
  const familyNote = number >= 19 && number <= 24
    ? '\n## 家族特性\n\nHard-shadow family 以硬边框、明确的层叠阴影和高对比状态块建立操作优先级；' + meta[3] + '通过自己的主色、字形和装饰节奏维持可区分性。\n'
    : number >= 25
      ? '\n## 家族特性\n\nPop family 共享高饱和主色、粗体展示字和直接的状态反馈；' + meta[3] + '以本页令牌、标题字形和背景处理形成独立辨识度。\n'
      : '';
  return '# ' + meta[0] + ' ' + theme.title + '\n\n'
    + '## 主题定位\n\n'
    + meta[3] + '。该方向围绕“' + theme.subtitle + '”组织，适合需要快速浏览任务状态并频繁切换工作上下文的桌面应用。\n\n'
    + '## 视觉关键词\n\n'
    + (theme.description ? description : vibes + '。') + '\n\n'
    + '## 色彩令牌\n\n'
    + '主色与强调色：' + palette + '。页面同时提供浅色与深色样式，具体令牌如下。\n\n'
    + '| 模式 | 变量 | 值 |\n| --- | --- | --- |\n' + tokenRows(theme.css, theme.cssClass) + '\n\n'
    + '## 字体与字重\n\n'
    + '展示与正文字体分别为：' + fonts + '。主题 CSS 实际声明的字体栈包括 ' + cssFonts + '；本主题覆盖的字重为 ' + weights + '。标题、品牌名、任务名称和等宽终端内容据此形成从扫描标题到密集日志的层级。\n\n'
    + '## 布局与间距\n\n'
    + '预览舞台最小宽度为 `880px`，应用窗口高度为 `632px`。顶栏为 `52px`，侧栏为 `300px`；顶栏横向内边距 `16px`，任务行内边距 `10px 12px 10px 10px`，终端输出区内边距 `18px 20px`。窄视口保留横向滚动，不压缩任务与终端的操作密度。\n\n'
    + '## 主要组件规则\n\n'
    + '应用框架由顶栏、品牌标记、三段任务标签、任务树、终端标签和终端输出区组成。' + meta[3] + '在组件形状上使用圆角 ' + radii + '，阴影或描边处理为 ' + shadows + '。按钮、任务行和终端标签沿用主题变量的前景、表面和强调色，以保持同一视觉语言。\n\n'
    + '## 状态表现\n\n'
    + (states || '工作中、空闲、未读和异常状态使用主题 CSS 中对应的状态变量。') + '。工作中与未读状态点带有脉冲环；在 `prefers-reduced-motion` 下动画关闭但状态点仍保持可见。\n\n'
    + '## 交互反馈\n\n'
    + '任务行和终端项保留悬停反馈，主题按钮点击后只在当前静态预览中切换 `chosen` 状态；不会调用 TaskAI API 或修改持久化数据。截图查询参数可隐藏页面外壳以聚焦当前主题。\n\n'
    + '## 可读性与无障碍\n\n'
    + '正文与背景使用主题定义的前景、边框和强调色令牌保持层级区分。交互元素保留可辨识的轮廓、文本标签与状态色，避免仅依赖单一颜色传达状态。\n\n'
    + '## React + MUI 实现映射\n\n'
    + '可用 `CssVarsProvider` 承载令牌；应用框架映射为 `AppBar`、`Drawer` 和 `Box`，列表映射为 `List`/`ListItemButton`，任务行为映射为 `Checkbox`、`Chip`、`IconButton` 与 `Tooltip`。主题的色彩令牌应进入 MUI `palette` 和组件变体，布局尺寸保持为显式设计 token。\n\n'
    + '## 预览实现\n\n'
    + '本页引用 `../../_shared/preview.css` 与 `../../_shared/preview.js`，仅内联 ' + meta[0] + ' 的主题 CSS 与配置，因此可直接以 `file://` 打开，无需生产接口或应用运行时。\n'
    + familyNote;
}

function pageHtml(meta, theme, config) {
  const section = theme.section.replace('class="study reveal"', 'class="study reveal in"');
  return '<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n<meta name="viewport" content="width=device-width, initial-scale=1.0">\n<title>' + meta[0] + ' ' + theme.title + ' | 设计预览</title>\n<link rel="stylesheet" href="../../_shared/preview.css">\n<style>\n' + theme.css + '\n</style>\n</head>\n<body>\n<header class="theme-page-head"><a href="../../index.html">设计主题库</a><span>' + meta[0] + ' / ' + theme.title + '</span><a href="design.md">设计文档</a></header>\n<main class="theme-page-main">\n' + section + '\n</main>\n<footer class="theme-page-foot">TaskAI Design Preview</footer>\n<script>window.taskaiPreviewConfig=' + JSON.stringify(config) + ';</script>\n<script src="../../_shared/preview.js"></script>\n</body>\n</html>\n';
}

function galleryHtml(themes) {
  const cards = themes.map(({ meta, theme }) => {
    const palette = theme.palette.slice(0, 5).map((color) => '<i style="background:' + color + '"></i>').join('');
    return '<a class="theme-card" href="themes/' + meta[0] + '-' + meta[1] + '/index.html"><span class="theme-card-index">' + meta[0] + '</span><strong>' + meta[2] + '</strong><span class="theme-card-description">' + meta[3] + '</span><span class="theme-card-palette" aria-label="主题代表色">' + palette + '</span></a>';
  }).join('\n');
  return '<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n<meta name="viewport" content="width=device-width, initial-scale=1.0">\n<title>TaskAI 设计主题库</title>\n<link rel="stylesheet" href="_shared/preview.css">\n</head>\n<body class="theme-gallery">\n<header class="gallery-head"><div><p>TaskAI</p><h1>36 套设计主题</h1><span>每个方向均提供独立的浅色、深色预览与设计说明。</span></div></header>\n<p class="gallery-choice">选择一个方向进入独立预览，比较其亮色与暗色工作台。</p>\n<main class="theme-grid">\n' + cards + '\n</main>\n</body>\n</html>\n';
}

async function main() {
  try {
    await readFile(sourcePath, 'utf8');
  } catch {
    await copyFile(join(previewDir, 'index.html'), sourcePath);
  }
  const source = await readFile(sourcePath, 'utf8');
  const style = between(source, '<style>', '</style>');
  const sourceScript = between(source, '<script>', '</script>');
  const themeCss = extractThemeCss(style);
  const sections = extractSections(source);
  if (themeCss.size !== 36 || sections.size !== 36) throw new Error('无法完整解析原始主题页');
  const parsedThemes = themeMetadata.map((meta) => {
    const theme = parseTheme(sections.get(meta[0]), themeCss.get(meta[0]));
    theme.section = sections.get(meta[0]);
    theme.css = isolateThemeCss(theme.css, theme.cssClass);
    return { meta, theme };
  });

  if (!skipShared) {
    const firstThemeIndex = style.search(/\/\* =+\n\s*01 ·/);
    const sectionIndex = style.search(/\/\* =+\n\s*SECTION \/ DESIGN-NOTE CARDS/);
    const previewChromeCss = [
      '\n.theme-page-head,.gallery-head{max-width:1320px;margin:0 auto;padding:28px 32px;display:flex;align-items:center;gap:16px}',
      '.theme-page-head a{color:var(--g-mut);text-decoration:none}.theme-page-head a:last-child{margin-left:auto}.theme-page-main{max-width:1320px;margin:0 auto}.theme-page-foot{padding:32px;color:var(--g-mut);text-align:center}',
      '.theme-gallery{min-height:100vh}.gallery-head{display:block;border-bottom:1px solid var(--g-line)}.gallery-head p{margin:0 0 8px;color:#9aa0ff;font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}.gallery-head h1{margin:0 0 10px;font-size:36px}.gallery-head span{color:var(--g-mut)}',
      '.gallery-choice{max-width:1256px;margin:0 auto;padding:0 32px 4px;color:var(--g-mut);font-size:13px}.theme-grid{max-width:1320px;margin:0 auto;padding:32px;display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:12px}',
      '.theme-card{min-height:160px;padding:18px;border:1px solid var(--g-line);border-radius:6px;background:rgba(255,255,255,.035);color:var(--g-ink);text-decoration:none;display:flex;flex-direction:column;gap:8px;transition:transform .16s ease,border-color .16s ease,background .16s ease}.theme-card:hover{transform:translateY(-2px);border-color:rgba(154,160,255,.65);background:rgba(255,255,255,.065)}.theme-card-index{color:var(--g-mut);font:700 11px var(--g-mono)}.theme-card strong{font-size:16px}.theme-card-description{color:var(--g-mut);font-size:13px}.theme-card-palette{display:flex;gap:5px;margin-top:auto}.theme-card-palette i{display:block;width:17px;height:17px;border-radius:3px;box-shadow:inset 0 0 0 1px rgba(255,255,255,.16)}',
      '@media (max-width:700px){.theme-page-head{padding:16px;justify-content:space-between}.theme-page-head span{display:none}.theme-page-head a:last-child{margin-left:0}.gallery-head{padding:24px 20px}.gallery-head h1{font-size:30px}.gallery-choice{padding:0 20px 4px}.theme-grid{padding:20px;grid-template-columns:1fr}}\n',
    ].join('');
    const sharedCss = (style.slice(0, firstThemeIndex) + style.slice(sectionIndex)).replace('.dot.working::before,.dot.unread::before,.d-nebula .live::before', '.dot.working::before,.dot.unread::before,.live::before') + previewChromeCss;
    const symbols = source.match(/<svg width="0" height="0"[\s\S]*?<\/svg>/)?.[0] ?? '';
    const terminal = sourceScript.match(/const TERM_OUT = `[\s\S]*?`;/)?.[0] ?? '';
    const app = sourceScript.match(/function appHTML\(\)\{[\s\S]*?\n\}/)?.[0] ?? '';
    const sharedJs = 'const svgSymbols=' + JSON.stringify(symbols) + ';\n'
      + 'document.body.insertAdjacentHTML(\'afterbegin\', svgSymbols);\n'
      + terminal + '\n' + app + '\n'
      + 'const config=window.taskaiPreviewConfig||{};\n'
      + 'document.querySelectorAll(\'.stage[data-variant]\').forEach((stage)=>{stage.innerHTML=appHTML();const mock=stage.closest(\'.mockup\');const name=mock.querySelector(\'.brand-name\');const winTitle=mock.querySelector(\'.win-title\');const bar=mock.querySelector(\'.stage-bar\');if(config.brandHtml)name.innerHTML=config.brandHtml;else name.textContent=config.brandText||config.fallback||name.textContent;if(config.windowHtml)winTitle.innerHTML=config.windowHtml;else winTitle.textContent=config.windowText||config.fallback||winTitle.textContent;if(config.extraClass){const extra=document.createElement(\'span\');extra.className=config.extraClass;extra.textContent=config.extraText;bar.appendChild(extra);}});\n'
      + 'document.querySelectorAll(\'.pick\').forEach((button)=>button.addEventListener(\'click\',()=>{document.querySelectorAll(\'.pick\').forEach((item)=>item.classList.toggle(\'chosen\',item===button));}));\n'
      + 'if(new URLSearchParams(location.search).get(\'show\')){document.querySelectorAll(\'.theme-page-head,.theme-page-foot\').forEach((element)=>{element.style.display=\'none\';});}\n';
    await mkdir(join(previewDir, '_shared'), { recursive: true });
    await writeFile(join(previewDir, '_shared', 'preview.css'), sharedCss);
    await writeFile(join(previewDir, '_shared', 'preview.js'), sharedJs);
  }

  if (!skipGallery) await writeFile(join(previewDir, 'index.html'), galleryHtml(parsedThemes));

  if (!skipThemes) {
    for (const { meta, theme } of parsedThemes) {
      const number = Number(meta[0]);
      if (range && (number < range[0] || number > range[1])) continue;
      const target = join(previewDir, 'themes', meta[0] + '-' + meta[1]);
      await mkdir(target, { recursive: true });
      await writeFile(join(target, 'index.html'), pageHtml(meta, theme, { ...extractConfig(sourceScript, theme.cssClass, theme.title), number: meta[0] }));
      const documentPath = join(target, 'design.md');
      try {
        if (!refreshDocs) await readFile(documentPath, 'utf8');
      } catch {
        await writeFile(documentPath, themeDocument(meta, theme));
      }
      if (refreshDocs) await writeFile(documentPath, themeDocument(meta, theme));
    }
  }
}

main().catch((error) => { console.error(error); process.exitCode = 1; });
