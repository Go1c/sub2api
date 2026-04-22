export default {
  site: {
    name: 'Dragon Code',
    tagline: '企业级 AI 编码中转',
    footer: '© {year} Dragon Code. All rights reserved.'
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
