export default {
  site: {
    name: 'LumioAPI',
    tagline: '企业级 AI 编码中转',
    footer: '© {year} LumioAPI. All rights reserved.'
  },
  marketing: {
    header: {
      home: '首页',
      pricing: '定价',
      status: '服务状态',
      docs: '文档',
      cta: '开始使用',
      ctaAuthed: '进入控制台'
    },
    hero: {
      eyebrow: '开发者首选',
      title: 'AI 编码工作台',
      subtitle: '一个账号、一条线路，统一调用 Claude Code、Codex 和 Gemini CLI。更低价格、更稳链路、更透明计费。',
      primary: '立即体验',
      secondary: '查看定价',
      note: '核心体验'
    },
    why: {
      title: '为什么选择',
      titleBrand: 'LumioAPI',
      subtitle: '专为日常编码而设计的一站式体验 ✦ 更低的价格，更简单的配置，让每位开发者都能用上顶级 AI 编码模型。',
      items: [
        {
          title: '5 分钟完成配置',
          desc: '统一的域名与密钥格式，覆盖主流 IDE 插件与 CLI 工具，新手也能快速上手。'
        },
        {
          title: '与官方一致的精确计费',
          desc: '采用与官方完全一致的计价方式，每一笔费用都经得起核对。'
        },
        {
          title: '每一笔调用，费用一目了然',
          desc: '每一次请求的 Token 数、模型与价格都可回溯，按需调优成本。'
        },
        {
          title: '专注编码，拒绝瞎折腾',
          desc: '一个账号同时使用 Claude、Codex、Gemini，不再在多个控制台之间切换。'
        }
      ]
    },
    pricing: {
      title: '模型定价',
      ctaRegister: '免费注册',
      ctaExperience: '立即体验',
      note: '我们的价格以人民币（¥）计价 · 官方原价以美元（$）标注 · 汇率按 1:7 折算（单位：百万 tokens）',
      cols: {
        model: '模型名',
        group: '分组',
        multiplier: '倍率',
        input: '输入（¥）',
        output: '输出（¥）',
        official: '官方（¥）',
        discount: '折扣',
        openClaw: 'OpenClaw'
      },
      yes: '是',
      no: '否',
      loading: '加载定价…',
      error: '定价暂时不可用'
    },
    footer: {
      sections: [
        {
          title: '产品',
          links: ['LumioAPI 介绍', '价格方案', '登录']
        },
        {
          title: '资源',
          links: ['使用教程', '品牌故事', 'Claude 模型']
        },
        {
          title: '服务承诺',
          links: ['透明定价', '服务状态', '隐私保护', '安全合规']
        },
        {
          title: '解决方案',
          links: ['AI 编程助手', '代码生成', '技术支持']
        },
        {
          title: '关于',
          links: ['关于我们', '联系我们']
        }
      ],
      copyright: '© {year} LumioAPI. All rights reserved.'
    }
  },
  nav: {
    dashboard: '仪表盘',
    keys: 'API 密钥',
    usage: '使用记录',
    recharge: '现在充值',
    redeem: '兑换',
    docs: '文档',
    profile: '个人资料',
    lightMode: '浅色模式',
    darkMode: '深色模式',
    collapse: '收起',
    logout: '退出登录',
    login: '登录',
    register: '注册'
  },
  dashboard: {
    title: '仪表盘',
    subtitle: '欢迎回来！这是您的概览',
    stats: {
      balance: '余额',
      balanceHint: '可用',
      balanceAction: '充值',
      keys: 'API 密钥',
      keysHint: '启用',
      requests: '今日请求',
      requestsTotal: '总计 {v}',
      spend: '今日消耗',
      spendTotal: '总计',
      tokenToday: '今日 Token',
      tokenTotal: '累计 Token',
      tokenIO: '输入 {i} / 输出 {o}',
      perf: '性能指标',
      perfRpm: 'RPM',
      perfTpm: 'TPM',
      latency: '平均响应',
      latencyHint: '平均响应'
    },
    invite: {
      title: '邀请金',
      invited: '已邀请用户',
      totalBonus: '累计邀请金',
      monthBonus: '本月邀请金',
      code: '我的邀请码及邀请链接',
      copy: '复制',
      copied: '已复制',
      note: '邀请好友注册，好友每次充值时您将获得好友充值金额 5% 的邀请金'
    },
    filter: {
      range: '时间范围',
      granularity: '粒度',
      last7: '近 7 天',
      last30: '近 30 天',
      last90: '近 90 天',
      byDay: '按天',
      byHour: '按小时'
    },
    charts: {
      model: '模型分布',
      token: 'Token 使用趋势',
      empty: '暂无数据',
      emptyFields: ['模型', '请求', 'Token', '实际', '标准']
    },
    recent: {
      title: '最近使用',
      range: '近 7 天',
      empty: '暂无数据'
    },
    quick: {
      title: '快捷操作',
      newKey: '创建 API 密钥',
      newKeyHint: '生成新的 API 密钥',
      viewUsage: '查看使用记录',
      viewUsageHint: '查询使用记录',
      recharge: '现在充值',
      rechargeHint: '充值账户余额',
      docs: '查看文档',
      docsHint: '打开使用指南'
    }
  },
  auth: {
    login: {
      title: '欢迎回来',
      email: '邮箱',
      password: '密码',
      submit: '登录',
      toRegister: '还没账号？立即注册'
    },
    register: {
      title: '创建账号',
      email: '邮箱',
      password: '密码',
      submit: '注册',
      toLogin: '已有账号？去登录'
    }
  },
  keys: {
    title: 'API 密钥',
    create: '创建 API 密钥',
    name: '名称',
    prefix: '前缀',
    created: '创建时间',
    status: '状态',
    actions: '操作'
  },
  usage: {
    title: '使用记录',
    subtitle: '最近 30 天使用情况',
    totalRequests: '总请求',
    totalTokens: '总 Token',
    cost: '总消耗'
  },
  groups: {
    title: '分组',
    name: '名称',
    capacity: '容量',
    members: '成员',
    usage: '用量'
  },
  profile: {
    title: '个人资料',
    email: '邮箱',
    plan: '套餐',
    joinedAt: '注册时间'
  },
  common: {
    loading: '加载中…',
    empty: '暂无数据',
    save: '保存',
    cancel: '取消',
    confirm: '确认',
    back: '返回'
  }
}
