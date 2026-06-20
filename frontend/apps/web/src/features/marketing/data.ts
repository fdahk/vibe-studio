// 营销/作品集落地页的结构化双语数据（内容本身就是双语，由当前语言挑选，不走 i18n key）。
export type Lang = 'zh' | 'en'
export interface Bi {
  zh: string
  en: string
}

export function pick<T>(lang: Lang, v: { zh: T; en: T }): T {
  return lang === 'zh' ? v.zh : v.en
}

export const PROFILE = {
  name: { zh: '涂将', en: 'Tu Jiang' },
  pinyin: 'TU JIANG',
  age: { zh: '20 岁', en: '20 y/o' },
  school: { zh: '东华理工大学 · 28 届本科', en: 'East China Univ. of Tech · Class of 2028' },
  intent: { zh: '全栈开发 · 偏前端', en: 'Full-stack · Front-end leaning' },
  slogan: { zh: 'Hello!', en: 'Hello!' },
  eyebrow: { zh: '个人主页 · 开放求职', en: 'Personal site · Open to work' },
  intro: {
    zh: '独立做过实时通讯、知识库 agent、博客与电商等项目，正在自研对标扣子的 AI 全栈开发平台。',
    en: 'Built realtime chat, KB agents, blogs and e-commerce solo; now building an AI full-stack platform.',
  },
  award: { zh: '蓝桥杯算法竞赛 · 国奖', en: 'Lanqiao Algorithm Contest · National Award' },
  phone: '+86 185 7919 4952',
  email: '3235159187@qq.com',
  github: 'fdahk',
  juejin: 'fdahk',
} as const

export interface WorkItem {
  role: Bi
  company: Bi
  period: string
  location: Bi
  highlights: { zh: string[]; en: string[] }
  stack: string[]
}

export const WORKS: WorkItem[] = [
  {
    role: { zh: '全栈开发', en: 'Full-stack Engineer' },
    company: { zh: '北京智源人工智能研究院', en: 'Beijing Academy of AI (BAAI)' },
    period: '2025.09 — 2026.05',
    location: { zh: '北京', en: 'Beijing' },
    highlights: {
      zh: [
        'React / Next / Shopify / Tailwind 营销展示与数据分析页',
        'Flutter 端 CV 模型多场景应用与常规业务',
        'Express / FastAPI 后端，含 RAG agent 系统',
        'Prometheus + 飞书 Hook 搭建服务监测分析平台',
      ],
      en: [
        'Marketing & analytics pages on React / Next / Shopify / Tailwind',
        'Flutter apps integrating CV models across scenarios',
        'Express / FastAPI backends including a RAG agent system',
        'Service observability via Prometheus + Feishu hooks',
      ],
    },
    stack: ['React', 'Next', 'Flutter', 'Express', 'FastAPI', 'Prometheus'],
  },
  {
    role: { zh: '全栈开发', en: 'Full-stack Engineer' },
    company: { zh: '上海妙妙宠科技', en: 'Shanghai MiaoMiao Pet Tech' },
    period: '2025.08 — 2025.09',
    location: { zh: '上海', en: 'Shanghai' },
    highlights: {
      zh: [
        'Android / iOS 双端应用 + Node Express 后端 + 内部网页维护',
        '地图寻宠定位、智能录音宠物翻译、软硬件交互模块',
        '部署于腾讯云',
      ],
      en: [
        'Android / iOS apps + Node Express backend + internal sites',
        'Map-based pet locator, voice translation, hardware integration',
        'Deployed on Tencent Cloud',
      ],
    },
    stack: ['Flutter', 'Node', 'Express', 'Tencent Cloud'],
  },
  {
    role: { zh: '微信小程序前端', en: 'WeChat Mini-program Front-end' },
    company: { zh: '成都小来空间科技', en: 'Chengdu Xiaolai Space Tech' },
    period: '2025.07 — 2025.08',
    location: { zh: '成都', en: 'Chengdu' },
    highlights: {
      zh: ['微信小程序前端开发', '展示并桥接影视服务商与终端用户的功能模块'],
      en: [
        'WeChat mini-program front-end development',
        'Modules bridging film/TV providers with end users',
      ],
    },
    stack: ['Uniapp', 'Vue 3'],
  },
]

export interface SkillGroup {
  title: Bi
  items: string[]
}

export const SKILLS: SkillGroup[] = [
  {
    title: { zh: '前端', en: 'Front-end' },
    items: ['Vue', 'React', 'Flutter', 'Uniapp', 'TypeScript', 'Tailwind', 'SCSS'],
  },
  {
    title: { zh: '后端', en: 'Back-end' },
    items: ['Go', 'Node (Express / Nest)', 'Python (FastAPI / Flask)'],
  },
  {
    title: { zh: '工程化', en: 'Tooling' },
    items: ['Vite', 'Git', 'Docker', 'GitHub Actions', 'Nginx', 'Linux', 'Shell'],
  },
  {
    title: { zh: '数据与中间件', en: 'Data & Middleware' },
    items: ['MySQL', 'MongoDB', 'Redis', 'PostgreSQL', 'RabbitMQ'],
  },
]
