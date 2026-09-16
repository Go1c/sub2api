// Fork i18n gap pack: restores admin keys missing from monothithic after upstream merges.
// Loaded last among admin modular overlays (deep-merge fill).
export default {
  accounts: {
      errorHistory: {
            columns: {
                    count: '次数',
                    message: '错误信息',
                    model: '模型',
                    source: '来源',
                    statusCode: '状态码',
                    time: '时间',
                    userEmail: '用户邮箱'
                  },
            dupHint: '该错误在短时间内重复了 {count} 次',
            empty: '暂无错误记录',
            subtitle: '最近 20 条错误记录',
            title: '错误历史'
          },
      fromModel: '源模型',
      messages: {
            accountCreated: '账号创建成功'
          },
      toModel: '目标模型',
      upstreamBalance: {
            failed: '请求失败',
            label: '上游余额',
            success: '请求成功'
          }
    },
  affiliates: {
      records: {
            fingerprint: '指纹',
            ip: 'IP',
            result: '结果',
            signupBonusAmount: '注册赠送',
            signupBonusAt: '注册时间',
            signupBonusGranted: '已赠送'
          }
    },
  channels: {
      emptyModelsInPricing: '定价条目未选择模型',
      form: {
            imageInputPrice: '图片输入单价'
          },
      noGroupsSelected: '未选择分组'
    },
  ops: {
      runtime: {
            metricThresholds: '指标阈值',
            metricThresholdsHint: '用于运行态健康判定的阈值配置',
            requestErrorRateMaxPercent: '请求错误率上限 (%)',
            requestErrorRateMaxPercentHint: '超过该错误率视为不健康',
            slaMinPercent: 'SLA 最低可用率 (%)',
            slaMinPercentHint: '低于该可用率告警',
            ttftP99MaxMs: 'TTFT P99 上限 (ms)',
            ttftP99MaxMsHint: '首 Token 延迟 P99 阈值',
            upstreamErrorRateMaxPercent: '上游错误率上限 (%)',
            upstreamErrorRateMaxPercentHint: '上游错误率超过该值告警'
          },
      settings: {
            accountAlertCooldownMinutes: '冷却时间（分钟）',
            accountAlertIntervalMinutes: '扫描间隔（分钟）',
            accountAlertMaxRows: '最多账号数',
            accountAlertMaxUsers: '影响用户邮箱数',
            accountAlertMinCount: '触发次数',
            accountAlertWindowMinutes: '统计窗口（分钟）',
            accountErrorAlert: '账号异常 Telegram 告警',
            enableAccountErrorAlert: '开启账号异常告警',
            telegramBotToken: 'Telegram Bot Token',
            telegramChatId: 'Telegram Chat ID',
            validation: {
                    accountAlertCooldownRange: '账号异常告警冷却时间必须在 0-10080 分钟之间',
                    accountAlertIntervalRange: '账号异常告警扫描间隔必须在 1-1440 分钟之间',
                    accountAlertMaxRowsRange: '账号异常告警最多账号数必须在 1-50 之间',
                    accountAlertMaxUsersRange: '账号异常告警影响用户邮箱数必须在 0-10 之间',
                    accountAlertMinCountRange: '账号异常告警触发次数必须在 1-100000 之间',
                    accountAlertWindowRange: '账号异常告警统计窗口必须在 1-1440 分钟之间',
                    telegramBotTokenRequired: '开启账号异常告警时必须填写 Telegram Bot Token',
                    telegramChatIdRequired: '开启账号异常告警时必须填写 Telegram Chat ID'
                  }
          },
      userRequestMonitor: {
            adminOnly: '仅管理员',
            capture: {
                    actions: '操作',
                    bytes: '字节数',
                    contentType: 'Content-Type',
                    endpoint: '端点',
                    expiresAt: '过期时间',
                    model: '模型',
                    requestId: '请求 ID',
                    time: '时间'
                  },
            capturesTitle: '请求捕获',
            close: '关闭',
            copied: '已复制',
            copy: '复制请求体',
            copyFailed: '复制失败',
            create: '开始监控',
            createFailed: '创建监控失败',
            createTitle: '创建监控',
            created: '监控已创建',
            creating: '创建中...',
            deleteCapture: '删除捕获',
            deleteConfirm: '确定删除用户 {email} 的请求监控及其全部捕获吗？此操作不可撤销。',
            deleteFailed: '删除捕获失败',
            deleteMonitor: '删除',
            deleteMonitorFailed: '删除监控失败',
            deleted: '捕获已删除',
            description: '为单个用户启动限时监控，并捕获之后的原始请求体。',
            detail: '详情',
            detailTitle: '请求体详情',
            download: '下载',
            downloadFailed: '下载捕获失败',
            downloaded: '捕获导出已开始',
            durationMinutes: '持续时间（分钟）',
            empty: '暂无用户请求监控。',
            invalidForm: '请输入有效的用户 ID 和监控设置',
            loadCaptureFailed: '加载捕获详情失败',
            loadCapturesFailed: '加载捕获失败',
            loadFailed: '加载请求监控失败',
            monitorDeleted: '监控已删除',
            noCaptures: '暂无捕获。',
            rateLimit: '每分钟最大捕获数',
            rawWarning: '警告：请求体会以原始内容保存，不会脱敏。捕获失败只记录日志，不会阻断用户请求。',
            refresh: '刷新',
            retentionDays: '保留天数',
            sampleRate: '采样率 %',
            searchPlaceholder: '按邮箱搜索',
            status: {
                    active: '进行中',
                    all: '全部',
                    expired: '已过期',
                    stopped: '已停止'
                  },
            stop: '停止',
            stopFailed: '停止监控失败',
            stopped: '监控已停止',
            table: {
                    actions: '操作',
                    captures: '捕获数',
                    endsAt: '结束时间',
                    limits: '限速 / 采样',
                    status: '状态',
                    user: '用户'
                  },
            title: '用户请求监控',
            truncated: '已截断',
            userId: '用户 ID',
            userIdPlaceholder: '例如 123',
            viewCaptures: '查看捕获'
          }
    },
  settings: {
      features: {
            affiliate: {
                    balanceGate: {
                              description: '防止注册赠送余额被刷后直接消耗服务；仅余额计费模式生效，不影响订阅模式。',
                              minBalance: '最低当前余额',
                              minBalanceHint: '账户余额必须大于该值才可使用余额服务。',
                              minRecharge: '最低历史充值',
                              minRechargeHint: '历史充值必须大于该值才可使用余额服务。邀请赠送不会计入历史充值。',
                              title: '余额使用门控'
                            },
                    tiers: {
                              description: '等级固定为 L1-L4。邀请人数和邀请充值总额同时达标后，系统按最高达标等级发放返利。',
                              hint: '返利比例留空表示该等级不生效；保存后用户页与返利发放均以后端配置为准。',
                              level: '等级',
                              minInvitees: '邀请人数 >=',
                              minRecharge: '邀请充值 >=',
                              rebateRate: '返利比例',
                              title: '阶梯返利配置',
                              unconfigured: '未配置'
                            }
                  }
          },
      payment: {
            providerMapay: '码支付'
          },
      registration: {
            invitationRegistrationMode: '邀请注册验证方式',
            invitationRegistrationModeAffiliateLink: '用户邀请链接',
            invitationRegistrationModeBoth: '两者都可',
            invitationRegistrationModeHint: '选择用户注册时可使用的邀请凭证。',
            invitationRegistrationModeRedeemCode: '管理员邀请码'
          },
      site: {
            frontendLocales: '前端可用语言',
            frontendLocalesHint: '控制用户可以在前端语言切换器中选择哪些语言。',
            sitePages: {
                    add: '添加页面',
                    content: 'Markdown 内容',
                    contentHint: '支持 Markdown，保存后可通过 /doc/文档 访问。',
                    contentPlaceholder: '# 文档\n\n在这里编写 Markdown 内容。',
                    description: '配置顶部导航中的文档、服务条款、隐私协议，可选择 Markdown 页面或站内嵌入链接页面。',
                    invalidLink: '链接模式需要填写完整的 http(s) 地址',
                    invalidSlug: '公开页面路径必须以 doc/ 开头，且不能包含 ?、#、反斜杠、连续斜杠或 ..',
                    itemLabel: '页面 #{n}',
                    key: '页面标识',
                    keyPlaceholder: '例如：docs',
                    linkHint: '链接模式下，点击导航会直接打开该 http(s) 地址。访问路径仍可用于 /doc/... 页面内嵌。',
                    linkPlaceholder: 'https://docs.example.com',
                    linkUrl: '链接地址',
                    mode: '页面模式',
                    modeLink: '链接',
                    modeMarkdown: 'Markdown',
                    required: '公开页面需要填写标识、标题和访问路径',
                    slug: '访问路径',
                    slugPlaceholder: 'doc/文档',
                    title: '公开页面',
                    titleLabel: '页面标题',
                    titlePlaceholder: '例如：文档'
                  }
          },
      openaiFastPolicy: {
            addUserId: '添加用户 ID',
            removeUserId: '移除用户 ID',
            userIdPlaceholder: '例如 123'
          }
    },
  subscriptions: {
      columns: {
            exhaustedAt: '耗尽时间'
          },
      createdEnd: '创建结束',
      createdStart: '创建开始',
      emailFilter: '邮箱',
      failedToResetWeeklyLimit: '重置周限失败',
      limit: '上限',
      planId: '套餐 ID',
      quotaPool: '额度池',
      recent30dWaste: '近30天浪费',
      remaining: '剩余',
      resetWeeklyLimit: '重置周限',
      resetWeeklyLimitConfirm: '确定只重置 \'{user}\' 当前周限用量吗？仅每周用量归零；总额度、累计已用额度、每日用量、订阅状态和到期时间均不会改变。',
      resetWeeklyLimitTitle: '重置每周用量限制',
      status: {
            suspended: '已暂停'
          },
      used: '已用',
      wasteStats: {
            byPlan: '按套餐统计',
            dailyBreakdown: '每日明细',
            description: '按周期和套餐追踪未使用的订阅额度。',
            end: '结束日期',
            failedToLoad: '加载浪费统计失败',
            period: '周期',
            plan: '套餐',
            start: '开始日期',
            subscriptions: '订阅数',
            title: '订阅浪费统计',
            totalQuota: '总额度',
            totalUsed: '已使用',
            totalWaste: '浪费额度',
            trend: '趋势',
            unknownPlan: '未知套餐',
            userId: '用户 ID',
            wasteRate: '浪费率',
            weeklyBreakdown: '每周明细'
          },
      weeklyLimitResetSuccess: '周限用量已重置'
    },
  usage: {
      billingModeVideo: '按次(视频)',
      tokenRanking: {
            subtitle: '按当前筛选与时间范围统计每个用户的 Token 用量',
            rowHint: '点击查看该用户的用量明细',
            userCount: '共 {count} 位用户',
            columns: {
                    user: '用户',
                    requests: '请求数',
                    inputTokens: '输入 Token',
                    outputTokens: '输出 Token',
                    cacheTokens: '缓存 Token',
                    totalTokens: '总 Token',
                    cost: '费用'
                  }
          }
    },
  users: {
      typeBalancePayment: '余额支付订阅',
      typePromoBalance: '余额（优惠码）',
      typeSubscriptionPayment: '订阅消费',
      typeWalletDebit: '外部钱包扣款',
      passwordCopied: '密码已复制'
    }
}
