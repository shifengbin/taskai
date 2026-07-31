export type PreviewMode = 'light' | 'dark' | 'modal'

export interface ThemeConcept {
  id: string
  name: string
  tagline: string
  colors: readonly [string, string, string]
  task: string
  terminal: string
}

export const themeConcepts: readonly ThemeConcept[] = [
  {
    id: 'juice', name: '果汁俱乐部', tagline: '蜜桃汽水 · 番茄辣酱 · 钴蓝杯垫',
    colors: ['#ff725c', '#ffca3a', '#2354d8'], task: '把发布清单榨成一杯果汁', terminal: 'pnpm ship --sparkle',
  },
  {
    id: 'night-market', name: '夜市电台', tagline: '荧光信号 · 玫粉频率 · 深夜节目单',
    colors: ['#08f7d1', '#ff4fa3', '#10111d'], task: '让通知频道亮起来', terminal: 'codex --station 98.6',
  },
  {
    id: 'puzzle', name: '拼图总部', tagline: '明黄拼片 · 靛蓝标题 · 砖红批注',
    colors: ['#ffd23f', '#2747c7', '#e85d4a'], task: '拼好本周的交付地图', terminal: 'git status --short',
  },
  {
    id: 'lava', name: '熔岩赛道', tagline: '熔岩冲刺 · 酸性绿 · 终点黑旗',
    colors: ['#ff5a36', '#b8ff3c', '#161616'], task: '冲过部署前最后一圈', terminal: 'npm run release',
  },
  {
    id: 'pool', name: '泳池派对', tagline: '湖蓝水面 · 珊瑚浮标 · 柠檬日光',
    colors: ['#1ab6d9', '#ff765f', '#ffe04b'], task: '把需求漂到已完成泳道', terminal: 'docker compose up',
  },
  {
    id: 'candy', name: '糖纸档案', tagline: '葡萄档案袋 · 薄荷索引 · 奶油便签',
    colors: ['#7b3ff2', '#4ed8b6', '#fff2d9'], task: '归档这次设计冲刺', terminal: 'git commit -m "wrap"',
  },
]
