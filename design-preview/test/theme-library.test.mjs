import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const previewRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
);

const themes = [
  ['01', 'monolith'], ['02', 'atelier'], ['03', 'nebula'], ['04', 'wabi'],
  ['05', 'pop'], ['06', 'deco'], ['07', 'comic'], ['08', 'holo'],
  ['09', 'arcade'], ['10', 'citrus'], ['11', 'blocks'], ['12', 'sunny'],
  ['13', 'candy'], ['14', 'doodle'], ['15', 'y2k'], ['16', 'groovy'],
  ['17', 'disco'], ['18', 'burst'], ['19', 'acid'], ['20', 'riso'],
  ['21', 'memphis'], ['22', 'sunset'], ['23', 'concrete'], ['24', 'neon'],
  ['25', 'bolt'], ['26', 'gummy'], ['27', 'stamp'], ['28', 'punch'],
  ['29', 'snap'], ['30', 'brio'], ['31', 'boom'], ['32', 'zap'],
  ['33', 'crayon'], ['34', 'riot'], ['35', 'juice'], ['36', 'pixel'],
];

test('each of the 36 themes has an isolated page and Chinese design document', () => {
  for (const [number, slug] of themes) {
    const themeRoot = path.join(previewRoot, 'themes', `${number}-${slug}`);

    assert.equal(
      existsSync(path.join(themeRoot, 'index.html')),
      true,
      `${number}-${slug} 缺少独立预览页面`,
    );
    assert.equal(
      existsSync(path.join(themeRoot, 'design.md')),
      true,
      `${number}-${slug} 缺少设计文档`,
    );
  }
});

test('the gallery links every theme page and no longer describes only 24 themes', () => {
  const gallery = readFileSync(path.join(previewRoot, 'index.html'), 'utf8');

  assert.doesNotMatch(gallery, /二十四套方向|二十四套方案/);
  for (const [number, slug] of themes) {
    assert.match(
      gallery,
      new RegExp(`themes/${number}-${slug}/index\\.html`),
      `${number}-${slug} 缺少总览链接`,
    );
  }
  assert.equal((gallery.match(/class="theme-card-palette"/g) ?? []).length, 36);
  assert.match(gallery, /class="theme-card-palette" aria-label="主题代表色"><i style="background:/);
});

test('each detail page is self-contained through shared assets and documents its design system', () => {
  const requiredDocumentSections = [
    '## 主题定位',
    '## 色彩令牌',
    '## 字体与字重',
    '## 布局与间距',
    '## 主要组件规则',
    '## 状态表现',
    '## 交互反馈',
    '## 可读性与无障碍',
    '## React + MUI 实现映射',
  ];

  for (const [number, slug] of themes) {
    const themeRoot = path.join(previewRoot, 'themes', `${number}-${slug}`);
    const page = readFileSync(path.join(themeRoot, 'index.html'), 'utf8');
    const document = readFileSync(path.join(themeRoot, 'design.md'), 'utf8');
    const cssSlug = number === '01' ? 'mono' : slug;
    const inlineStyle = page.match(/<style>([\s\S]*?)<\/style>/)?.[1] ?? '';

    assert.match(page, /href="\.\.\/\.\.\/_shared\/preview\.css"/);
    assert.match(page, /src="\.\.\/\.\.\/_shared\/preview\.js"/);
    assert.match(page, new RegExp(`mockup d-${cssSlug} light`));
    assert.match(page, new RegExp(`mockup d-${cssSlug} dark`));
    assert.doesNotMatch(inlineStyle, new RegExp(`\\.d-(?!${cssSlug}(?:[.\\s:{]))`));
    assert.doesNotMatch(page, /class="theme-choice"/);
    assert.match(document, /\| 亮色 \|/);
    assert.match(document, /\| 暗色 \|/);
    const tokenBlocks = [...inlineStyle.matchAll(new RegExp(`\\.d-${cssSlug}\\.(light|dark)\\{([^}]*)\\}`, 'gi'))];
    for (const block of tokenBlocks) {
      const mode = block[1].toLowerCase() === 'light' ? '亮色' : '暗色';
      const tokens = [...block[2].matchAll(/--([a-z0-9-]+):([^;}\n]+)(?:;|(?=$))/gi)];
      for (const token of tokens) {
        assert.ok(
          document.includes(`| ${mode} | \`--${token[1]}\` | \`${token[2].trim()}\` |`),
          `${number}-${slug} 的 ${mode} 令牌 --${token[1]} 未同步到设计文档`,
        );
      }
    }
    for (const section of requiredDocumentSections) assert.ok(document.includes(section));
  }
});

test('shared styles define a self-contained gallery and mobile page navigation', () => {
  const sharedCss = readFileSync(path.join(previewRoot, '_shared', 'preview.css'), 'utf8');

  assert.match(sharedCss, /\.theme-card-palette i\{/);
  assert.match(sharedCss, /@media \(max-width:700px\)\{\.theme-page-head/);
  assert.doesNotMatch(sharedCss, /var\(--(?:paper|line|accent|mono|muted|ink)\)/);
});
