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

function themeDocument(meta, theme) {
  const palette = theme.palette.map((value) => '`' + value + '`').join('、') || '页面 CSS 令牌定义的主题色';
  const fonts = theme.fonts.map((value) => '`' + value + '`').join('、') || '系统字体栈';
  const vibes = theme.vibes.join('、') || '单任务、高信息密度';
  const description = theme.description && /[。！？]$/.test(theme.description) ? theme.description : theme.description + '。';
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
    + '## 字体与文字\n\n'
    + '主要字体：' + fonts + '。标题负责建立层级，标签与元数据保持紧凑，以便在任务清单、侧栏和状态区中稳定扫描。\n\n'
    + '## 布局与组件\n\n'
    + '预览使用最小高度 `880px` 的舞台，应用窗口宽度为 `632px`。顶栏高度为 `52px`，侧栏宽度为 `300px`；任务列表、筛选项、状态徽标与操作按钮复用同一密度规则。\n\n'
    + '## 状态与交互\n\n'
    + '每个页面同时呈现浅色与深色应用窗口。主题切换会更新窗口标题和状态区内容；清单中的选择、复选和标签状态用于展示任务的进行、完成与提醒反馈。\n\n'
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
  return '<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n<meta name="viewport" content="width=device-width, initial-scale=1.0">\n<title>' + meta[0] + ' ' + theme.title + ' | 设计预览</title>\n<link rel="stylesheet" href="../../_shared/preview.css">\n<style>\n' + theme.css + '\n</style>\n</head>\n<body>\n<header class="theme-page-head"><a href="../../index.html">设计主题库</a><span>' + meta[0] + ' / ' + theme.title + '</span><a href="design.md">设计文档</a></header>\n<div class="theme-choice" id="themeChoice" role="status" aria-live="polite">尚未选择，点击“选择此方案”可标记当前主题。</div>\n<main class="theme-page-main">\n' + section + '\n</main>\n<footer class="theme-page-foot">TaskAI Design Preview</footer>\n<script>window.taskaiPreviewConfig=' + JSON.stringify(config) + ';</script>\n<script src="../../_shared/preview.js"></script>\n</body>\n</html>\n';
}

function galleryHtml(themes) {
  const cards = themes.map((meta) => '<a class="theme-card" href="themes/' + meta[0] + '-' + meta[1] + '/index.html"><span class="theme-card-index">' + meta[0] + '</span><strong>' + meta[2] + '</strong><span>' + meta[3] + '</span></a>').join('\n');
  return '<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n<meta name="viewport" content="width=device-width, initial-scale=1.0">\n<title>TaskAI 设计主题库</title>\n<link rel="stylesheet" href="_shared/preview.css">\n</head>\n<body class="theme-gallery">\n<header class="gallery-head"><div><p>TaskAI</p><h1>36 套设计主题</h1><span>每个方向均提供独立的浅色、深色预览与设计说明。</span></div></header>\n<main class="theme-grid">\n' + cards + '\n</main>\n</body>\n</html>\n';
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

  if (!skipShared) {
    const firstThemeIndex = style.search(/\/\* =+\n\s*01 ·/);
    const sectionIndex = style.search(/\/\* =+\n\s*SECTION \/ DESIGN-NOTE CARDS/);
    const sharedCss = (style.slice(0, firstThemeIndex) + style.slice(sectionIndex)).replace('.dot.working::before,.dot.unread::before,.d-nebula .live::before', '.dot.working::before,.dot.unread::before,.live::before') + '\n.theme-page-head,.gallery-head{max-width:1320px;margin:0 auto;padding:28px 32px;display:flex;align-items:center;gap:16px}.theme-page-head a{color:var(--muted);text-decoration:none}.theme-page-head a:last-child{margin-left:auto}.theme-choice{max-width:1256px;margin:0 auto 4px;padding:10px 14px;background:var(--paper);border:1px solid var(--line);color:var(--muted);font-size:13px}.theme-page-main{max-width:1320px;margin:0 auto}.theme-page-foot{padding:32px;color:var(--muted);text-align:center}.theme-gallery{min-height:100vh}.gallery-head{display:block;border-bottom:1px solid var(--line)}.gallery-head p{margin:0 0 8px;color:var(--accent);font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}.gallery-head h1{margin:0 0 10px;font-size:36px}.gallery-head span{color:var(--muted)}.theme-grid{max-width:1320px;margin:0 auto;padding:32px;display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:12px}.theme-card{min-height:150px;padding:18px;border:1px solid var(--line);border-radius:6px;background:var(--paper);color:var(--ink);text-decoration:none;display:flex;flex-direction:column;gap:8px;transition:transform .16s ease,border-color .16s ease}.theme-card:hover{transform:translateY(-2px);border-color:var(--accent)}.theme-card-index{color:var(--muted);font:700 11px var(--mono)}.theme-card strong{font-size:16px}.theme-card span:last-child{margin-top:auto;color:var(--muted);font-size:13px}\n';
    const symbols = source.match(/<svg width="0" height="0"[\s\S]*?<\/svg>/)?.[0] ?? '';
    const terminal = sourceScript.match(/const TERM_OUT = `[\s\S]*?`;/)?.[0] ?? '';
    const app = sourceScript.match(/function appHTML\(\)\{[\s\S]*?\n\}/)?.[0] ?? '';
    const sharedJs = 'const svgSymbols=' + JSON.stringify(symbols) + ';\n'
      + 'document.body.insertAdjacentHTML(\'afterbegin\', svgSymbols);\n'
      + terminal + '\n' + app + '\n'
      + 'const config=window.taskaiPreviewConfig||{};\n'
      + 'document.querySelectorAll(\'.stage[data-variant]\').forEach((stage)=>{stage.innerHTML=appHTML();const mock=stage.closest(\'.mockup\');const name=mock.querySelector(\'.brand-name\');const winTitle=mock.querySelector(\'.win-title\');const bar=mock.querySelector(\'.stage-bar\');if(config.brandHtml)name.innerHTML=config.brandHtml;else name.textContent=config.brandText||config.fallback||name.textContent;if(config.windowHtml)winTitle.innerHTML=config.windowHtml;else winTitle.textContent=config.windowText||config.fallback||winTitle.textContent;if(config.extraClass){const extra=document.createElement(\'span\');extra.className=config.extraClass;extra.textContent=config.extraText;bar.appendChild(extra);}});\n'
      + 'const choice=document.getElementById(\'themeChoice\');document.querySelectorAll(\'.pick\').forEach((button)=>button.addEventListener(\'click\',()=>{document.querySelectorAll(\'.pick\').forEach((item)=>item.classList.toggle(\'chosen\',item===button));if(choice)choice.textContent=\'已选择 \'+(config.number||\'当前主题\')+\'，可返回主题总览继续比较。\';}));\n'
      + 'if(new URLSearchParams(location.search).get(\'show\')){document.querySelectorAll(\'.theme-page-head,.theme-choice,.theme-page-foot\').forEach((element)=>{element.style.display=\'none\';});}\n';
    await mkdir(join(previewDir, '_shared'), { recursive: true });
    await writeFile(join(previewDir, '_shared', 'preview.css'), sharedCss);
    await writeFile(join(previewDir, '_shared', 'preview.js'), sharedJs);
  }

  if (!skipGallery) await writeFile(join(previewDir, 'index.html'), galleryHtml(themeMetadata));

  if (!skipThemes) {
    for (const meta of themeMetadata) {
      const number = Number(meta[0]);
      if (range && (number < range[0] || number > range[1])) continue;
      const css = themeCss.get(meta[0]);
      const section = sections.get(meta[0]);
      const theme = parseTheme(section, css);
      theme.section = section;
      theme.css = isolateThemeCss(theme.css, theme.cssClass);
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
